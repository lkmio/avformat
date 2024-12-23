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

	metaData *libflv.AMF0Object

	audioStreamIndex int
	videoStreamIndex int
	audioTimestamp   uint32
	videoTimestamp   uint32
	playStreamId     uint32

	receiveDataSize      uint32
	totalReceiveDataSize uint32
	conn                 net.Conn
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
	s.totalReceiveDataSize += uint32(length)
	if s.receiveDataSize > WindowSize/2 {
		bytes := [4]byte{}
		binary.BigEndian.PutUint32(bytes[:], s.totalReceiveDataSize)
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
		amf0 := libflv.AMF0{}
		if err := amf0.Unmarshal(chunk.Body[:chunk.Size]); err != nil {
			return err
		} else if amf0.Size() < 2 {
			break
		}

		element := amf0.Get(1)
		if libflv.AMF0DataTypeString != element.Type() {
			break
		} else if "onMetaData" != string(element.(libflv.AMF0String)) {
			break
		}

		if libflv.AMF0DataTypeObject == amf0.Get(2).Type() {
			s.metaData = amf0.Get(2).(*libflv.AMF0Object)
		} else if libflv.AMF0DataTypeECMAArray == amf0.Get(2).Type() {
			s.metaData = amf0.Get(2).(*libflv.AMF0ECMAArray).AMF0Object
		}
		break
	case MessageTypeIDDataAMF3:
		break
	case MessageTypeIDCommandAMF0, MessageTypeIDSharedObjectAMF0:
		amf0 := libflv.AMF0{}
		err := amf0.Unmarshal(chunk.Body[:chunk.Size])
		if err != nil {
			return err
		} else if amf0.Size() < 2 {
			break
		} else if libflv.AMF0DataTypeString != amf0.Get(0).Type() {
			return fmt.Errorf("the first element of amf0 must be of type command: %s", hex.EncodeToString(chunk.Body[:chunk.Size]))
		} else if libflv.AMF0DataTypeNumber != amf0.Get(1).Type() {
			return fmt.Errorf("the second element of amf0 must be id of transaction: %s", hex.EncodeToString(chunk.Body[:chunk.Size]))
		}

		command := string(amf0.Get(0).(libflv.AMF0String))
		transactionId := float64(amf0.Get(1).(libflv.AMF0Number))

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
			writer := libflv.AMF0{}
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

			writer.Add(properties)
			writer.Add(information)
			n, _ := writer.Marshal(tmp[:])

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

			if amf0.Size() > 2 && libflv.AMF0DataTypeObject == amf0.Get(2).Type() {
				object := amf0.Get(2).(*libflv.AMF0Object)
				if property := object.FindProperty("app"); property != nil && libflv.AMF0DataTypeString == property.Value.Type() {
					s.app = string(property.Value.(libflv.AMF0String))
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

			writer := libflv.AMF0{}
			writer.AddString(MessageResult)
			writer.AddNumber(transactionId)
			writer.Add(libflv.AMF0Null{})
			writer.AddNumber(1) // 应答0, 某些推流端可能会失败

			var tmp [128]byte
			n, _ := writer.Marshal(tmp[:])

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

			//  The command structure from the client to the server is as follows:
			// +--------------+----------+----------------------------------------+
			// | Field Name   |   Type   |             Description                |
			// 	+--------------+----------+----------------------------------------+
			// | Command Name |  String  | Name of the command, set to "publish". |
			// +--------------+----------+----------------------------------------+
			// | Transaction  |  Number  | Transaction ID set to 0.               |
			// | ID           |          |                                        |
			// +--------------+----------+----------------------------------------+
			// | Command      |  Null    | Command information object does not    |
			// | Object       |          | exist. Set to null type.               |
			// +--------------+----------+----------------------------------------+
			// | Publishing   |  String  | Name with which the stream is          |
			// | Name         |          | published.                             |
			// +--------------+----------+----------------------------------------+
			// | Publishing   |  String  | Type of publishing. Set to "live",     |
			// | Type         |          | "record", or "append".                 |
			// |              |          | record: The stream is published and the|
			// |              |          | data is recorded to a new file.The file|
			// |              |          | is stored on the server in a           |
			// |              |          | subdirectory within the directory that |
			// |              |          | contains the server application. If the|
			// |              |          | file already exists, it is overwritten.|
			// |              |          | append: The stream is published and the|
			// |              |          | data is appended to a file. If no file |
			// |              |          | is found, it is created.               |
			// |              |          | live: Live data is published without   |
			// |              |          | recording it in a file.                |
			// +--------------+----------+----------------------------------------+
			// stream
			if amf0.Size() > 3 {
				if stream, ok := amf0.Get(3).(libflv.AMF0String); ok {
					s.stream = string(stream)
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

			// The command structure from the client to the server is as follows:
			// +--------------+----------+-----------------------------------------+
			// | Field Name   |   Type   |             Description                 |
			// 	+--------------+----------+-----------------------------------------+
			// | Command Name |  String  | Name of the command. Set to "play".     |
			// +--------------+----------+-----------------------------------------+
			// | Transaction  |  Number  | Transaction ID set to 0.                |
			// | ID           |          |                                         |
			// +--------------+----------+-----------------------------------------+
			// | Command      |   Null   | Command information does not exist.     |
			// | Object       |          | Set to null type.                       |
			// +--------------+----------+-----------------------------------------+
			// | Stream Name  |  String  | Name of the stream to play.             |
			// |              |          | To play video (FLV) files, specify the  |
			// |              |          | name of the stream without a file       |
			// |              |          | extension (for example, "sample"). To   |
			// |              |          | play back MP3 or ID3 tags, you must     |
			// |              |          | precede the stream name with mp3:       |
			// |              |          | (for example, "mp3:sample". To play     |
			// |              |          | H.264/AAC files, you must precede the   |
			// |              |          | stream name with mp4: and specify the   |
			// |              |          | file extension. For example, to play the|
			// |              |          | file sample.m4v,specify "mp4:sample.m4v"|
			// |              |          |                                         |
			// +--------------+----------+-----------------------------------------+
			// | Start        |  Number  | An optional parameter that specifies    |
			// |              |          | the start time in seconds. The default  |
			// |              |          | value is -2, which means the subscriber |
			// |              |          | first tries to play the live stream     |
			// |              |          | specified in the Stream Name field. If a|
			// |              |          | live stream of that name is not found,it|
			// |              |          | plays the recorded stream of the same   |
			// |              |          | name. If there is no recorded stream    |
			// |              |          | with that name, the subscriber waits for|
			// |              |          | a new live stream with that name and    |
			// |              |          | plays it when available. If you pass -1 |
			// |              |          | in the Start field, only the live stream|
			// |              |          | specified in the Stream Name field is   |
			// |              |          | played. If you pass 0 or a positive     |
			// |              |          | number in the Start field, a recorded   |
			// |              |          | stream specified in the Stream Name     |
			// |              |          | field is played beginning from the time |
			// |              |          | specified in the Start field. If no     |
			// |              |          | recorded stream is found, the next item |
			// |              |          | in the playlist is played.              |
			// +--------------+----------+-----------------------------------------+
			// | Duration     |  Number  | An optional parameter that specifies the|
			// |              |          | duration of playback in seconds. The    |
			// |              |          | default value is -1. The -1 value means |
			// |              |          | a live stream is played until it is no  |
			// |              |          | longer available or a recorded stream is|
			// |              |          | played until it ends. If you pass 0, it |
			// |              |          | plays the single frame since the time   |
			// |              |          | specified in the Start field from the   |
			// |              |          | beginning of a recorded stream. It is   |
			// |              |          | assumed that the value specified in     |
			// |              |          | the Start field is equal to or greater  |
			// |              |          | than 0. If you pass a positive number,  |
			// |              |          | it plays a live stream for              |
			// |              |          | the time period specified in the        |
			// |              |          | Duration field. After that it becomes   |
			// |              |          | available or plays a recorded stream    |
			// |              |          | for the time specified in the Duration  |
			// |              |          | field. (If a stream ends before the     |
			// |              |          | time specified in the Duration field,   |
			// |              |          | playback ends when the stream ends.)    |
			// |              |          | If you pass a negative number other     |
			// |              |          | than -1 in the Duration field, it       |
			// |              |          | interprets the value as if it were -1.  |
			// +--------------+----------+-----------------------------------------+
			// | Reset        | Boolean  | An optional Boolean value or number     |
			// |              |          | that specifies whether to flush any     |
			// |              |          | previous playlist.                      |
			// +--------------+----------+-----------------------------------------+

			if amf0.Size() > 3 {
				if stream, ok := amf0.Get(3).(libflv.AMF0String); ok {
					s.stream = string(stream)
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

	amf0Writer := libflv.AMF0{}
	amf0Writer.AddString(MessageOnStatus)
	amf0Writer.AddNumber(transactionId)
	amf0Writer.Add(libflv.AMF0Null{})

	object := &libflv.AMF0Object{}
	object.AddStringProperty("level", level)
	object.AddStringProperty("code", code)
	object.AddStringProperty("description", description)

	amf0Writer.Add(object)

	var tmp [128]byte
	n, _ := amf0Writer.Marshal(tmp[:])

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

func (s *Stack) MetaData() *libflv.AMF0Object {
	return s.metaData
}

func (s *Stack) SendStreamBeginChunk() error {
	bytes := make([]byte, 6)
	binary.BigEndian.PutUint16(bytes, 0)
	binary.BigEndian.PutUint32(bytes[2:], 1)

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
	binary.BigEndian.PutUint32(bytes[2:], 1)

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
