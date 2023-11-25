package librtmp

import (
	"encoding/binary"
	"fmt"
	"github.com/yangjiechina/avformat"
	"github.com/yangjiechina/avformat/libflv"
	"github.com/yangjiechina/avformat/utils"
	"net"
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
	OnVideo(data []byte, ts uint32)

	OnAudio(data []byte, ts uint32)
}

type Stack struct {
	buffer         utils.ByteBuffer
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
}

func NewStack(handler OnEventHandler) *Stack {
	utils.Assert(handler != nil)
	return &Stack{parser: NewParser(), handler: handler}
}

func (s *Stack) SetOnPublishHandler(handler OnPublishHandler) {
	s.publisherHandler = handler
}

func (s *Stack) SetOnTransDeMuxerHandler(handler avformat.OnTransDeMuxerHandler) {
	s.parser.handler = handler
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
		}
	}

	return i, nil
}

func (s *Stack) Input(conn net.Conn, data []byte) error {
	tmp := data

	if s.handshakeBufferSize > 0 {
		min := utils.MinInt(len(data), MaxHandshakeBufferSize-s.handshakeBufferSize)
		copy(s.handshakeBuffer[s.handshakeBufferSize:], data[:min])
		tmp = s.handshakeBuffer
		s.handshakeBufferSize = 0
	}

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

		s.parser.Reset()
		consume += n
	}

	return nil
}

func (s *Stack) SendChunks(conn net.Conn, chunks ...Chunk) error {
	var tmp [1024]byte
	var n int
	for i := 0; i < len(chunks); i++ {
		consume := chunks[i].ToBytes2(tmp[n:], s.parser.chunkSize)
		n += consume
	}

	_, err := conn.Write(tmp[:n])
	return err
}

func (s *Stack) ProcessMessage(conn net.Conn, chunk *Chunk) error {
	switch chunk.tid {
	case MessageTypeIDSetChunkSize:
		//s.parser.chunkSize = utils.BytesToInt(chunk.data)
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
		if s.parser.handler == nil {
			s.publisherHandler.OnAudio(chunk.data[:chunk.Length], chunk.Timestamp)
		} else {
			s.publisherHandler.OnAudio(nil, chunk.Timestamp)
		}

		break
	case MessageTypeIDVideo:
		if s.parser.handler == nil {
			s.publisherHandler.OnVideo(chunk.data[:chunk.Length], chunk.Timestamp)
		} else {
			s.publisherHandler.OnVideo(nil, chunk.Timestamp)
		}

		break
	case MessageTypeIDDataAMF0:
		//onMetaData
		amf0, err := libflv.DoReadAFM0(chunk.data[:chunk.size])
		if err != nil {
			return err
		}
		println(amf0)
		break
	case MessageTypeIDDataAMF3:
		break
	case MessageTypeIDCommandAMF0, MessageTypeIDSharedObjectAMF0:
		amf0, err := libflv.DoReadAFM0(chunk.data[:chunk.size])
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

			var tmp [64]byte
			writer := libflv.NewAMF0Writer()
			writer.AddString(MessageResult)
			writer.AddNumber(transactionId)
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

			err = s.SendChunks(conn, acknow, bandwidth, setChunkSize, response)
			s.parser.chunkSize = ChunkSize
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

			for i := 2; i < len(amf0); i++ {
				str, ok := amf0[i].(string)
				if ok {
					s.app = str
					break
				}
			}

			return s.SendChunks(conn, response)
		} else if MessagePublish == command {
			//stream
			for i := 2; i < len(amf0); i++ {
				str, ok := amf0[i].(string)
				if ok {
					s.stream = str
					break
				}
			}

			stream := s.app + "/" + s.stream
			state := make(chan utils.HookState, 1)
			s.handler.OnPublish(s.app, stream, state)

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

			stream := s.app + "/" + s.stream
			state := make(chan utils.HookState, 1)
			s.handler.OnPlay(s.app, stream, state)

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

}
