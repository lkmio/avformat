package librtmp

import (
	"encoding/binary"
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
	"net"

	"github.com/yangjiechina/avformat/libflv"
	"github.com/yangjiechina/avformat/utils"
)

const (
	// MaxHandshakeBufferSize 握手数据最大的缓存大小，超过该大小返回解析错误，防止客户端乱发数据
	MaxHandshakeBufferSize = 40960
)

// 事先缓存需要使用的参数
var windowSize []byte
var chunkSize []byte
var peerBandwidth []byte

func init() {
	windowSize_ := [4]byte{}
	chunkSize_ := [4]byte{}
	peerBandwidth_ := [5]byte{}
	windowSize = windowSize_[:]
	chunkSize = chunkSize_[:]
	peerBandwidth = peerBandwidth_[:]
	binary.BigEndian.PutUint32(windowSize, WindowSize)
	binary.BigEndian.PutUint32(chunkSize, ChunkSize)
	binary.BigEndian.PutUint32(peerBandwidth, ChunkSize)
	peerBandwidth[4] = 0x2 //dynamic
}

type OnEventHandler interface {
	OnPublish(app, stream string, response chan utils.HookState)

	OnPlay(app, stream string, response chan utils.HookState)
}

type OnPublishHandler interface {
	// OnPartPacket 部分音视频包回调
	OnPartPacket(index int, mediaType utils.AVMediaType, data []byte, first bool)

	// OnVideo 完整音视频帧回调
	OnVideo(index int, data []byte, ts uint32)

	OnAudio(index int, data []byte, ts uint32)
}

type Stack struct {
	handshakeState HandshakeState
	parser         *Parser

	//握手数据缓存(一次没有接受完，或者客户端乱发数据的情况才会使用)
	handshakeBuffer []byte
	//超过
	handshakeBufferSize int

	app              string
	stream           string
	handler          OnEventHandler
	publisherHandler OnPublishHandler

	metaData map[string]interface{}

	audioStreamIndex int
	videoStreamIndex int
	audioTimestamp   uint32
	videoTimestamp   uint32
}

func NewStack(handler OnEventHandler) *Stack {
	utils.Assert(handler != nil)
	return &Stack{parser: NewParser(), handler: handler,
		audioStreamIndex: -1,
		videoStreamIndex: -1}
}

func (s *Stack) SetOnPublishHandler(handler OnPublishHandler) {
	s.parser.partPacketCB = func(chunk *Chunk, data []byte) {
		if MessageTypeIDAudio == chunk.tid {
			if s.audioStreamIndex == -1 {
				s.audioStreamIndex = libbufio.MaxInt(s.videoStreamIndex, -1) + 1
			}

			s.publisherHandler.OnPartPacket(s.audioStreamIndex, utils.AVMediaTypeAudio, data, chunk.size == 0)
		} else {
			if s.videoStreamIndex == -1 {
				s.videoStreamIndex = libbufio.MaxInt(s.audioStreamIndex, -1) + 1
			}

			s.publisherHandler.OnPartPacket(s.videoStreamIndex, utils.AVMediaTypeVideo, data, chunk.size == 0)
		}
	}
	s.publisherHandler = handler
}

func (s *Stack) DoHandshake(conn net.Conn, data []byte) (int, error) {
	length, i := len(data), 0
	for i < length {
		if HandshakeStateUninitialized == s.handshakeState {
			s.handshakeState = HandshakeStateVersionSent
			if data[i] < VERSION {
				fmt.Printf("unkonw rtmp version:%d", data[i])
			}

			i++
		} else if HandshakeStateVersionSent == s.handshakeState {
			if length-i < HandshakePacketSize {
				break
			}

			//time
			_ = binary.BigEndian.Uint32(data[i:])
			//zero
			_ = binary.BigEndian.Uint32(data[i+4:])

			s0s1s2 := make([]byte, 1+HandshakePacketSize*2)
			//random bytes
			i += HandshakePacketSize
			bytes := data[i-HandshakePacketSize : i]
			n := GenerateS0S1S2(s0s1s2, bytes)
			utils.Assert(n == len(s0s1s2))

			_, err := conn.Write(s0s1s2)
			if err != nil {
				return i, err
			}
			s.handshakeState = HandshakeStateAckSent
		} else if HandshakeStateAckSent == s.handshakeState {
			if length-i >= HandshakePacketSize {
				s.handshakeState = HandshakeStateDone
				i += HandshakePacketSize
			}
			break
		}
	}

	return i, nil
}

func (s *Stack) Input(conn net.Conn, data []byte) error {
	tmp := data

	if s.handshakeBufferSize > 0 {
		min := libbufio.MinInt(len(data), MaxHandshakeBufferSize-s.handshakeBufferSize)
		copy(s.handshakeBuffer[s.handshakeBufferSize:], data[:min])
		tmp = s.handshakeBuffer[:s.handshakeBufferSize+min]
		s.handshakeBufferSize = 0
	}

	//握手
	if HandshakeStateDone != s.handshakeState {

		n, err := s.DoHandshake(conn, tmp)
		if err != nil {
			return err
		}

		rest := len(tmp) - n
		if rest == 0 {
			return nil
		}

		//还有多余的数据没有解析完毕
		if HandshakeStateDone == s.handshakeState {
			tmp = tmp[n:]
		} else {
			if rest+s.handshakeBufferSize > MaxHandshakeBufferSize {
				return fmt.Errorf("handshake failed, exceeding the maximum limit of %d", MaxHandshakeBufferSize)
			}

			if s.handshakeBuffer == nil {
				s.handshakeBuffer = make([]byte, MaxHandshakeBufferSize)
			}

			copy(s.handshakeBuffer[s.handshakeBufferSize:], tmp[n:])
			s.handshakeBufferSize += rest
			return nil
		}
	}

	//读取并处理消息
	var consume int
	for consume < len(tmp) {
		chunk, n, err := s.parser.ReadChunk(tmp[consume:])
		if err != nil {
			return err
		}

		if chunk == nil {
			return nil
		}

		err = s.ProcessMessage(conn, chunk)
		if err != nil {
			return err
		}

		chunk.Reset()
		s.parser.Reset()
		consume += n
	}

	return nil
}

func (s *Stack) SendChunks(conn net.Conn, chunks ...Chunk) error {
	var tmp [1024]byte
	var n int
	for i := 0; i < len(chunks); i++ {
		consume := chunks[i].ToBytes2(tmp[n:], s.parser.localChunkSize)
		n += consume
	}

	_, err := conn.Write(tmp[:n])
	return err
}

func (s *Stack) ProcessMessage(conn net.Conn, chunk *Chunk) error {
	switch chunk.tid {
	case MessageTypeIDSetChunkSize:
		s.parser.remoteChunkSize = int(binary.BigEndian.Uint32(chunk.data))
		break
	case MessageTypeIDAbortMessage:
		break
	case MessageTypeIDAcknowledgement:
		break
	case MessageTypeIDUserControlMessage:
		/*event := binary.BigEndian.Uint16(chunk.data)
		value := binary.BigEndian.Uint32(chunk.data[2:])
		processUserControlMessage(UserControlMessageEvent(event), value)*/
		break
	case MessageTypeIDWindowAcknowledgementSize:
		//windowSize = utils.BytesToInt(data)
		break
	case MessageTypeIDSetPeerBandWith:
		//bandwidth = binary.BigEndian.Uint32(chunk.data)
		//limit type 0-hard/1-soft/2-dynamic
		_ = chunk.data[4:]
		//p.sendWindowAcknowledgementSize()
		break
	case MessageTypeIDAudio:
		if ChunkType0 == chunk.type_ {
			s.audioTimestamp = chunk.Timestamp
		} else {
			s.audioTimestamp += chunk.Timestamp
		}

		if s.publisherHandler == nil {
			s.publisherHandler.OnAudio(s.audioStreamIndex, chunk.data[:chunk.Length], s.audioTimestamp)
		} else {
			s.publisherHandler.OnAudio(s.audioStreamIndex, nil, s.audioTimestamp)
		}

		break
	case MessageTypeIDVideo:
		if ChunkType0 == chunk.type_ {
			s.videoTimestamp = chunk.Timestamp
		} else {
			s.videoTimestamp += chunk.Timestamp
		}

		if s.publisherHandler == nil {
			s.publisherHandler.OnVideo(s.videoStreamIndex, chunk.data[:chunk.Length], s.videoTimestamp)
		} else {
			s.publisherHandler.OnVideo(s.videoStreamIndex, nil, s.videoTimestamp)
		}

		break
	case MessageTypeIDDataAMF0:
		//onMetaData
		amf0, err := libflv.DoReadAMF0(chunk.data[:chunk.size])
		if err != nil {
			return err
		}
		if len(amf0) < 3 {
			return nil
		}

		if str, ok := amf0[1].(string); ok && "onMetaData" == str {
			metaData, ok := amf0[2].(map[string]interface{})
			if !ok {
				return fmt.Errorf("failed to parse the meatadata of rtmp")
			}

			s.metaData = metaData
		}
		break
	case MessageTypeIDDataAMF3:
		break
	case MessageTypeIDCommandAMF0, MessageTypeIDSharedObjectAMF0:
		amf0, err := libflv.DoReadAMF0(chunk.data[:chunk.size])
		if err != nil {
			return err
		}

		if len(amf0) == 0 {
			return fmt.Errorf("invaild data: %s", string(chunk.data[:chunk.size]))
		}

		command, ok := amf0[0].(string)
		if !ok {
			return fmt.Errorf("the first element of amf0 must be of type command: %s", string(chunk.data[:chunk.size]))
		}
		transactionId, ok := amf0[1].(float64)

		if !ok {
			return fmt.Errorf("the second element of amf0 must be id of transaction: %s", string(chunk.data[:chunk.size]))
		}

		if MessageConnect == command {
			//window size
			//chunk size
			//bandwidth
			acknow := Chunk{
				type_:     ChunkType0,
				csid:      ChunkStreamIdNetwork,
				Timestamp: 0,
				tid:       MessageTypeIDWindowAcknowledgementSize,
				sid:       0,
				Length:    len(windowSize),

				data: windowSize,
				size: len(windowSize),
			}

			bandwidth := Chunk{
				type_:     ChunkType0,
				csid:      ChunkStreamIdNetwork,
				Timestamp: 0,
				tid:       MessageTypeIDSetPeerBandWith,
				sid:       0,
				Length:    len(peerBandwidth),

				data: peerBandwidth,
				size: len(peerBandwidth),
			}

			setChunkSize := Chunk{
				type_:     ChunkType0,
				csid:      ChunkStreamIdNetwork,
				Timestamp: 0,
				tid:       MessageTypeIDSetChunkSize,
				sid:       0,
				Length:    len(chunkSize),

				data: chunkSize,
				size: len(chunkSize),
			}

			var tmp [512]byte
			writer := libflv.NewAMF0Writer()
			writer.AddString(MessageResult)
			//always equal to 1 for the connect command
			writer.AddNumber(transactionId)
			//https://en.wikipedia.org/wiki/Real-Time_Messaging_Protocol#Connect
			properties := &libflv.AMF0Object{}
			properties.AddStringProperty("fmsVer", "FMS/3,5,5,2004")
			properties.AddNumberProperty("capabilities", 31)
			properties.AddNumberProperty("mode", 1)

			information := &libflv.AMF0Object{}
			information.AddStringProperty("level", "status")
			information.AddStringProperty("code", "NetConnection.Connect.Success")
			information.AddStringProperty("description", "Connection succeeded")
			information.AddNumberProperty("clientId", 0)
			information.AddNumberProperty("objectEncoding", 3.0)

			writer.AddObject(properties)
			writer.AddObject(information)
			//writer
			n := writer.ToBytes(tmp[:])

			response := Chunk{
				type_:     ChunkType0,
				csid:      ChunkStreamIdSystem,
				Timestamp: 0,
				tid:       MessageTypeIDCommandAMF0,
				sid:       0,
				Length:    n,

				data: tmp[:n],
				size: n,
			}

			for i := 2; i < len(amf0); i++ {
				obj, ok := amf0[i].(map[string]interface{})
				if !ok {
					continue
				}

				app, ok := obj["app"]
				if !ok {
					continue
				}

				if str, ok := app.(string); ok {
					s.app = str
				}
			}

			err = s.SendChunks(conn, acknow, bandwidth, response, setChunkSize)
			s.parser.localChunkSize = ChunkSize
			return err
		} else if MessageFcPublish == command {

		} else if MessageReleaseStream == command {

		} else if MessageCreateStream == command {
			writer := libflv.NewAMF0Writer()
			writer.AddString(MessageResult)
			writer.AddNumber(transactionId)
			writer.AddNull()
			writer.AddNumber(0)

			var tmp [128]byte
			n := writer.ToBytes(tmp[:])

			response := Chunk{
				type_:     ChunkType0,
				csid:      ChunkStreamIdSystem,
				Timestamp: 0,
				tid:       MessageTypeIDCommandAMF0,
				sid:       0,
				Length:    n,

				data: tmp[:n],
				size: n,
			}

			return s.SendChunks(conn, response)
		} else if MessagePublish == command {
			//stream
			for i := 2; i < len(amf0); i++ {
				if str, ok := amf0[i].(string); ok {
					s.stream = str
					break
				}
			}

			state := make(chan utils.HookState, 1)
			s.handler.OnPublish(s.app, s.stream, state)

			//在未收到响应之前，拒绝接受任何消息
			select {
			case response := <-state:
				if utils.HookStateOK == response {
					return s.sendStatus(conn, transactionId, "status", "NetStream.Play.Start", "Start publishing")
				} else {
					return s.sendStatus(conn, transactionId, "error", "NetStream.Publish.BadName", "Already publishing")
				}
			}

		} else if MessagePlay == command {
			for i := 2; i < len(amf0); i++ {
				str, ok := amf0[i].(string)
				if ok {
					s.stream = str
					break
				}
			}

			state := make(chan utils.HookState, 1)
			s.handler.OnPlay(s.app, s.stream, state)

			//在未收到响应之前，拒绝接受任何消息
			select {
			case response := <-state:
				if utils.HookStateOK == response {
					return s.sendStatus(conn, transactionId, "status", "NetStream.Play.Start", "Start live")
				} else {
					return s.sendStatus(conn, transactionId, "error", "NetStream.Publish.BadName", "Already publishing")
				}
			}

		} else if MessageResult == command {

		} else if MessageError == command {

		}

		break
	case MessageTypeIDCommandAMF3:
		break
	//case MessageTypeIDSharedObjectAMF0:
	//	break
	case MessageTypeIDSharedObjectAMF3:
		break
	case MessageTypeIDAggregateMessage:
		//unsupported
		break
	}

	return nil
}

func (s *Stack) sendStatus(conn net.Conn, transactionId float64, level, code, description string) error {

	amf0Writer := libflv.NewAMF0Writer()
	amf0Writer.AddString(MessageOnStatus)
	amf0Writer.AddNumber(transactionId)
	amf0Writer.AddNull()

	object := &libflv.AMF0Object{}
	object.AddStringProperty("level", level)
	object.AddStringProperty("code", code)
	object.AddStringProperty("description", description)

	amf0Writer.AddObject(object)

	var tmp [128]byte
	n := amf0Writer.ToBytes(tmp[:])

	response := Chunk{
		type_:     ChunkType0,
		csid:      ChunkStreamIdSystem,
		Timestamp: 0,
		tid:       MessageTypeIDCommandAMF0,
		sid:       0,
		Length:    n,

		data: tmp[:n],
		size: n,
	}

	return s.SendChunks(conn, response)
}

func (s *Stack) Close() {
	s.handler = nil
	s.publisherHandler = nil
}

func (s *Stack) MetaData() map[string]interface{} {
	return s.metaData
}
