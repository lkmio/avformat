package librtmp

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/lkmio/avformat/libbufio"
	"net"

	"github.com/lkmio/avformat/libflv"
	"github.com/lkmio/avformat/utils"
)

type HandshakeState byte

const (
	ParserStateInit              = ParserState(0)
	ParserStateBasicHeader       = ParserState(1)
	ParserStateTimestamp         = ParserState(2)
	ParserStateMessageLength     = ParserState(3)
	ParserStateStreamType        = ParserState(4)
	ParserStateStreamId          = ParserState(5)
	ParserStateExtendedTimestamp = ParserState(6)
	ParserStatePayload           = ParserState(7)

	// MaxHandshakeBufferSize 握手数据最大的缓存大小，超过该大小返回解析错误，防止客户端乱发数据
	MaxHandshakeBufferSize = 40960
)

// 事先缓存需要使用的参数
var windowSize []byte
var chunkSize []byte
var peerBandwidth []byte

func init() {
	windowSize = make([]byte, 4)
	chunkSize = make([]byte, 4)
	peerBandwidth = make([]byte, 5)

	binary.BigEndian.PutUint32(windowSize, WindowSize)
	binary.BigEndian.PutUint32(chunkSize, ChunkSize)
	binary.BigEndian.PutUint32(peerBandwidth, WindowSize)
	peerBandwidth[4] = 0x2 //dynamic
}

type OnEventHandler interface {
	OnPublish(app, stream string) utils.HookState

	OnPlay(app, stream string) utils.HookState
}

type OnPublishHandler interface {
	// OnPartPacket 部分音视频包回调
	OnPartPacket(index int, mediaType utils.AVMediaType, data []byte, first bool)

	// OnVideo 完整音视频帧回调
	OnVideo(index int, data []byte, ts uint32)

	OnAudio(index int, data []byte, ts uint32)
}

type Stack struct {
	handshakeState HandshakeState // 握手状态
	parser         *Parser        // chunk解析器

	handshakeBuffer     []byte // 握手数据缓存(一次没有接受完，或者客户端乱发数据的情况才会使用)
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
	playStreamId     uint32

	receiveDataSize         uint32
	receiveDataSizeTotal    uint32
	acknowledgementDataSize uint32
	conn                    net.Conn
}

func (s *Stack) SetOnPublishHandler(handler OnPublishHandler) {
	s.parser.onPartPacketCB = func(chunk *Chunk, data []byte) {
		if MessageTypeIDAudio == chunk.TypeID {
			if s.audioStreamIndex == -1 {
				s.audioStreamIndex = libbufio.MaxInt(s.videoStreamIndex, -1) + 1
			}

			s.publisherHandler.OnPartPacket(s.audioStreamIndex, utils.AVMediaTypeAudio, data, chunk.Size == 0)
		} else {
			if s.videoStreamIndex == -1 {
				s.videoStreamIndex = libbufio.MaxInt(s.audioStreamIndex, -1) + 1
			}

			s.publisherHandler.OnPartPacket(s.videoStreamIndex, utils.AVMediaTypeVideo, data, chunk.Size == 0)
		}
	}
	s.publisherHandler = handler
}

func (s *Stack) DoHandshake(data []byte) (int, error) {
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

			_, err := s.conn.Write(s0s1s2)
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

func (s *Stack) Input(data []byte) error {
	length := len(data)
	tmp := data

	if s.handshakeBufferSize > 0 {
		min := libbufio.MinInt(len(data), MaxHandshakeBufferSize-s.handshakeBufferSize)
		copy(s.handshakeBuffer[s.handshakeBufferSize:], data[:min])
		tmp = s.handshakeBuffer[:s.handshakeBufferSize+min]
		s.handshakeBufferSize = 0
	}

	s.receiveDataSize += uint32(length)
	s.receiveDataSizeTotal += uint32(length)
	if s.receiveDataSize > WindowSize/2 {
		bytes := [4]byte{}
		binary.BigEndian.PutUint32(bytes[:], s.receiveDataSizeTotal)
		acknowledgement := Chunk{
			Type:           ChunkType0,
			ChunkStreamID_: ChunkStreamIdNetwork,
			Timestamp:      0,
			TypeID:         MessageTypeIDAcknowledgement,
			StreamID:       0,
			Length:         4,

			Body: bytes[:],
			Size: 4,
		}

		s.SendChunks(acknowledgement)
		s.receiveDataSize = 0
	}

	//握手
	if HandshakeStateDone != s.handshakeState {
		n, err := s.DoHandshake(tmp)
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

	// 读取并处理消息
	var consume int
	for consume < len(tmp) {
		chunk, n, err := s.parser.ReadChunk(tmp[consume:])
		if err != nil {
			return err
		} else if chunk == nil {
			return nil
		}

		err = s.ProcessMessage(chunk)
		if err != nil {
			return err
		}

		chunk.Reset()
		s.parser.Reset()
		consume += n
	}

	return nil
}

func (s *Stack) SendChunks(chunks ...Chunk) error {
	var tmp [1024]byte
	var n int
	for i := 0; i < len(chunks); i++ {
		consume := chunks[i].Marshal(tmp[n:], s.parser.localChunkMaxSize)
		n += consume
	}

	_, err := s.conn.Write(tmp[:n])
	return err
}

func (s *Stack) ProcessMessage(chunk *Chunk) error {
	switch chunk.TypeID {
	case MessageTypeIDSetChunkSize:
		s.parser.remoteChunkMaxSize = int(binary.BigEndian.Uint32(chunk.Body))
		break
	case MessageTypeIDAbortMessage:
		break
	case MessageTypeIDAcknowledgement:
		break
	case MessageTypeIDUserControlMessage:
		/*event := binary.BigEndian.Uint16(chunk.Body)
		value := binary.BigEndian.Uint32(chunk.Body[2:])
		processUserControlMessage(UserControlMessageEvent(event), value)*/
		break
	case MessageTypeIDWindowAcknowledgementSize:
		//windowSize = utils.BytesToInt(Body)
		break
	case MessageTypeIDSetPeerBandWith:
		//bandwidth = binary.BigEndian.Uint32(chunk.Body)
		//limit type 0-hard/1-soft/2-dynamic
		_ = chunk.Body[4:]
		//p.sendWindowAcknowledgementSize()
		break
	case MessageTypeIDAudio:
		if ChunkType0 == chunk.Type {
			s.audioTimestamp = chunk.Timestamp
		} else {
			s.audioTimestamp += chunk.Timestamp
		}

		if s.parser.onPartPacketCB == nil {
			s.publisherHandler.OnAudio(s.audioStreamIndex, chunk.Body[:chunk.Length], s.audioTimestamp)
		} else {
			s.publisherHandler.OnAudio(s.audioStreamIndex, nil, s.audioTimestamp)
		}

		break
	case MessageTypeIDVideo:
		// type0是绝对时间戳, 其余是相对时间戳
		if ChunkType0 == chunk.Type {
			s.videoTimestamp = chunk.Timestamp
		} else {
			s.videoTimestamp += chunk.Timestamp
		}

		if s.parser.onPartPacketCB == nil {
			s.publisherHandler.OnVideo(s.videoStreamIndex, chunk.Body[:chunk.Length], s.videoTimestamp)
		} else {
			s.publisherHandler.OnVideo(s.videoStreamIndex, nil, s.videoTimestamp)
		}

		break
	case MessageTypeIDDataAMF0:
		//onMetaData
		amf0, err := libflv.DoReadAMF0(chunk.Body[:chunk.Size])
		if err != nil {
			return err
		} else if len(amf0) < 3 {
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
		amf0, err := libflv.DoReadAMF0(chunk.Body[:chunk.Size])
		if err != nil {
			return err
		}

		if len(amf0) == 0 {
			return fmt.Errorf("invaild Body: %s", hex.EncodeToString(chunk.Body[:chunk.Size]))
		}

		command, ok := amf0[0].(string)
		if !ok {
			return fmt.Errorf("the first element of amf0 must be of type command: %s", hex.EncodeToString(chunk.Body[:chunk.Size]))
		}

		transactionId, ok := amf0[1].(float64)
		if !ok {
			return fmt.Errorf("the second element of amf0 must be id of transaction: %s", hex.EncodeToString(chunk.Body[:chunk.Size]))
		}

		if MessageConnect == command {

			//	|              Handshaking done               |
			//	|                     |                       |
			//	|                     |                       |
			//	|                     |                       |
			//	|                     |                       |
			//	|----------- Command Message(connect) ------->|
			//	|                                             |
			//	|<------- Window Acknowledgement Size --------|
			//	|                                             |
			//	|<----------- Set Peer Bandwidth -------------|
			//	|                                             |
			//	|-------- Window Acknowledgement Size ------->|
			//	|                                             |
			//	|<------ User Control Message(StreamBegin) ---|
			//	|                                             |
			//	|<------------ Command Message ---------------|
			//	|       (_result- connect response)           |
			//	|                                             |

			//		The command structure from server to client is as follows:
			//	+--------------+----------+----------------------------------------+
			//	| Field Name   |   Type   |             Description                |
			//		+--------------+----------+----------------------------------------+
			//	| Command Name |  String  | _result or _error; indicates whether   |
			//	|              |          | the response is result or error.       |
			//	+--------------+----------+----------------------------------------+
			//	| Transaction  |  Number  | Transaction ID is 1 for connect        |
			//	| ID           |          | responses                              |
			//	|              |          |                                        |
			//	+--------------+----------+----------------------------------------+
			//	| Properties   |  Object  | Name-value pairs that describe the     |
			//	|              |          | properties(fmsver etc.) of the         |
			//	|              |          | connection.                            |
			//	+--------------+----------+----------------------------------------+
			//	| Information  |  Object  | Name-value pairs that describe the     |
			//	|              |          | response from|the server. ’code’,      |
			//	|              |          | ’level’, ’description’ are names of few|
			//	|              |          | among such information.                |
			//	+--------------+----------+----------------------------------------+

			//	window Size
			//	chunk Size
			//	bandwidth
			acknow := Chunk{
				Type:           ChunkType0,
				ChunkStreamID_: ChunkStreamIdNetwork,
				Timestamp:      0,
				TypeID:         MessageTypeIDWindowAcknowledgementSize,
				StreamID:       0,
				Length:         len(windowSize),

				Body: windowSize,
				Size: len(windowSize),
			}

			bandwidth := Chunk{
				Type:           ChunkType0,
				ChunkStreamID_: ChunkStreamIdNetwork,
				Timestamp:      0,
				TypeID:         MessageTypeIDSetPeerBandWith,
				StreamID:       0,
				Length:         len(peerBandwidth),

				Body: peerBandwidth,
				Size: len(peerBandwidth),
			}

			setChunkSize := Chunk{
				Type:           ChunkType0,
				ChunkStreamID_: ChunkStreamIdNetwork,
				Timestamp:      0,
				TypeID:         MessageTypeIDSetChunkSize,
				StreamID:       0,
				Length:         len(chunkSize),

				Body: chunkSize,
				Size: len(chunkSize),
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

			result := Chunk{
				Type:           ChunkType0,
				ChunkStreamID_: ChunkStreamIdSystem,
				Timestamp:      0,
				TypeID:         MessageTypeIDCommandAMF0,
				StreamID:       0,
				Length:         n,

				Body: tmp[:n],
				Size: n,
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

			err = s.SendChunks(acknow, bandwidth, result, setChunkSize)
			s.parser.localChunkMaxSize = ChunkSize
			return err
		} else if MessageFcPublish == command {
			// 使用Adobe Flash Media Live Encoder工具推流会发送该消息, 测试发现, 不应答也可以推流. 如果要应答, 按照Net Command格式.
		} else if MessageReleaseStream == command {
			// 使用Adobe Flash Media Live Encoder工具推流会发送该消息, 测试发现, 不应答也可以推流. 如果要应答, 按照Net Command格式.
		} else if MessageCreateStream == command {

			//		The command structure from server to client is as follows:
			//	+--------------+----------+----------------------------------------+
			//	| Field Name   |   Type   |             Description                |
			//		+--------------+----------+----------------------------------------+
			//	| Command Name |  String  | _result or _error; indicates whether   |
			//	|              |          | the response is result or error.       |
			//	+--------------+----------+----------------------------------------+
			//	| Transaction  |  Number  | ID of the command that response belongs|
			//	| ID           |          | to.                                    |
			//	+--------------+----------+----------------------------------------+
			//	| Command      |  Object  | If there exists any command info this  |
			//	| Object       |          | is set, else this is set to null type. |
			//	+--------------+----------+----------------------------------------+
			//	| Stream       |  Number  | The return value is either a stream ID |
			//	| ID           |          | or an error information object.        |
			//	+--------------+----------+----------------------------------------+

			writer := libflv.NewAMF0Writer()
			writer.AddString(MessageResult)
			writer.AddNumber(transactionId)
			writer.AddNull()
			writer.AddNumber(1) // 应答0, 某些推流端可能会失败

			var tmp [128]byte
			n := writer.ToBytes(tmp[:])

			response := Chunk{
				Type:           ChunkType0,
				ChunkStreamID_: ChunkStreamIdSystem,
				Timestamp:      0,
				TypeID:         MessageTypeIDCommandAMF0,
				StreamID:       0,
				Length:         n,

				Body: tmp[:n],
				Size: n,
			}

			return s.SendChunks(response)
		} else if MessagePublish == command {
			// stream
			for i := 2; i < len(amf0); i++ {
				if str, ok := amf0[i].(string); ok {
					s.stream = str
					break
				}
			}

			state := s.handler.OnPublish(s.app, s.stream)

			if utils.HookStateOK == state {
				return s.sendStatus(transactionId, "status", "NetStream.Publish.Start", "Start publishing")
			} else if utils.HookStateOccupy == state {
				return s.sendStatus(transactionId, "error", "NetStream.Publish.BadName", "Already publishing")
			} else {
				s.conn.Close()
			}
		} else if MessagePlay == command {
			for i := 2; i < len(amf0); i++ {
				str, ok := amf0[i].(string)
				if ok {
					s.stream = str
					break
				}
			}

			s.playStreamId = chunk.StreamID
			state := s.handler.OnPlay(s.app, s.stream)

			if utils.HookStateOK == state {
				return s.sendStatus(transactionId, "status", "NetStream.Play.Start", "Start live")
			} else {
				s.conn.Close()
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

func (s *Stack) sendStatus(transactionId float64, level, code, description string) error {

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
		Type:           ChunkType0,
		ChunkStreamID_: ChunkStreamIdSystem,
		Timestamp:      0,
		TypeID:         MessageTypeIDCommandAMF0,
		StreamID:       1,
		Length:         n,
		Body:           tmp[:n],
		Size:           n,
	}

	return s.SendChunks(response)
}

func (s *Stack) Close() {
	s.handler = nil
	s.publisherHandler = nil
	s.parser.onPartPacketCB = nil
	s.conn = nil
}

func (s *Stack) MetaData() map[string]interface{} {
	return s.metaData
}

func (s *Stack) SendStreamBeginChunk() error {
	bytes := make([]byte, 6)
	binary.BigEndian.PutUint16(bytes, 0)
	binary.BigEndian.PutUint32(bytes[2:], s.playStreamId)

	streamBegin := Chunk{
		Type:           ChunkType0,
		ChunkStreamID_: ChunkStreamIdNetwork,
		Timestamp:      0,
		TypeID:         MessageTypeIDUserControlMessage,
		StreamID:       0,
		Length:         6,
		Body:           bytes,
		Size:           len(windowSize),
	}

	return s.SendChunks(streamBegin)
}

func (s *Stack) SendStreamEOFChunk() error {
	bytes := make([]byte, 6)
	binary.BigEndian.PutUint16(bytes, 1)
	binary.BigEndian.PutUint32(bytes[2:], s.playStreamId)

	streamEof := Chunk{
		Type:           ChunkType0,
		ChunkStreamID_: ChunkStreamIdNetwork,
		Timestamp:      0,
		TypeID:         MessageTypeIDUserControlMessage,
		StreamID:       0,
		Length:         6,
		Body:           bytes,
		Size:           6,
	}

	return s.SendChunks(streamEof)
}

func NewStack(conn net.Conn, handler OnEventHandler) *Stack {
	utils.Assert(handler != nil)
	return &Stack{parser: NewParser(), handler: handler,
		audioStreamIndex: -1,
		videoStreamIndex: -1,
		conn:             conn,
	}
}
