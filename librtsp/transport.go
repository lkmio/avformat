package librtsp

import (
	"fmt"
	"github.com/lkmio/avformat/utils"
)

var (
	startPort = 2000
)

type Server struct {
	rtp        utils.Transport
	rtcp       utils.Transport
	mediaType  utils.AVMediaType
	clientPort string //sample:20000-20001
	serverAddr string
	serverPort [2]int
}

func CreateServer() (*Server, error) {
	var err error
	var transport1 utils.Transport
	var transport2 utils.Transport
	if startPort+1 >= utils.PortMaximum {
		startPort = 20000
	}
	port1, port2, b := utils.AllocPairPort(startPort, false)
	if !b {
		return nil, fmt.Errorf("failed to allocate port")

	}
	startPort += 2

	defer func() {
		if err != nil {
			if transport1 != nil {
				transport1.Close()
			}
		}
	}()
	transport1, err = utils.NewUDPTransport(port1)
	if err != nil {
		return nil, err
	}
	transport2, err = utils.NewUDPTransport(port2)
	if err != nil {
		return nil, err
	}

	return &Server{rtp: transport1, rtcp: transport2, clientPort: fmt.Sprintf("%d-%d", port1, port2)}, err
}

func (s *Server) traversal() {
	bytes := make([]byte, 12)
	bytes[0] = 0x80
	s.rtp.(*utils.UDPTransport).WriteTo(bytes, s.serverAddr, s.serverPort[0])
	go s.rtp.Read()
}
