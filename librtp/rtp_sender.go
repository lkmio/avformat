package librtp

import (
	"github.com/yangjiechina/avformat/transport"
	"net"
)

type RtpSender struct {
	Rtp  transport.ITransport
	Rtcp transport.ITransport

	RtpConn  net.Conn
	RtcpConn net.Conn

	//rtcp
	PktCount   int
	SSRC       uint32
	OctetCount int
}

func (s *RtpSender) OnRTPPacket(conn net.Conn, data []byte) {
	if s.RtpConn == nil {
		s.RtpConn = conn
	}
}

func (s *RtpSender) OnRTCPPacket(conn net.Conn, data []byte) {
	if s.RtcpConn == nil {
		s.RtcpConn = conn
	}

	//packs, err := rtcp.Unmarshal(data)
	//if err != nil {
	//	log.Sugar.Warnf("解析rtcp包失败 err:%s conn:%s pkt:%s", err.Error(), conn.RemoteAddr().String(), hex.EncodeToString(data))
	//	return
	//}
	//
	//for _, pkt := range packs {
	//	if _, ok := pkt.(*rtcp.ReceiverReport); ok {
	//	} else if _, ok := pkt.(*rtcp.SourceDescription); ok {
	//	} else if _, ok := pkt.(*rtcp.Goodbye); ok {
	//	}
	//}
}
