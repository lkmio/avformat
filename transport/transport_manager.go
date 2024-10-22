package transport

import (
	"fmt"
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/utils"
	"net"
	"sync"
)

type Manager interface {
	AllocPort(tcp bool, cb func(port uint16) error) error

	AllocPairPort(cb, c2 func(port uint16) error) error

	NewTCPServer(ip string) (*TCPServer, error)

	NewUDPServer(ip string) (*UDPServer, error)

	NewUDPClient(ip string, remoteAddr *net.UDPAddr) (*UDPClient, error)
}

type transportManager struct {
	startPort uint16
	endPort   uint16
	nextPort  uint16
	lock      sync.Mutex
}

func (t *transportManager) NewTCPServer(ip string) (*TCPServer, error) {
	server := TCPServer{}
	err := t.AllocPort(true, func(port uint16) error {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ip, port))
		if err != nil {
			return err
		}

		return server.Bind(addr)
	})

	return &server, err
}

func (t *transportManager) NewUDPServer(ip string) (*UDPServer, error) {
	server := UDPServer{}
	err := t.AllocPort(false, func(port uint16) error {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
		if err != nil {
			return err
		}

		return server.Bind(addr)
	})

	return &server, err
}

func (t *transportManager) NewUDPClient(ip string, remoteAddr *net.UDPAddr) (*UDPClient, error) {
	client := UDPClient{}
	err := t.AllocPort(false, func(port uint16) error {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
		if err != nil {
			return err
		}

		return client.Connect(addr, remoteAddr)
	})

	return &client, err
}

func (t *transportManager) AllocPort(tcp bool, cb func(port uint16) error) error {
	loop := func(start, end uint16, tcp bool) (uint16, error) {
		for i := start; i < end; i++ {
			if used := utils.Used(int(i), tcp); !used {
				return i, cb(i)
			}
		}

		return 0, nil
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	port, err := loop(t.nextPort, t.endPort, tcp)
	if port == 0 {
		port, err = loop(t.startPort, t.nextPort, tcp)
	}

	if port == 0 {
		return fmt.Errorf("no available ports in the [%d-%d] range", t.startPort, t.endPort)
	} else if err != nil {
		return err
	}

	t.nextPort = t.nextPort + 1%t.endPort
	t.nextPort = uint16(libbufio.MaxInt(int(t.nextPort), int(t.startPort)))
	return nil
}

func (t *transportManager) AllocPairPort(cb func(port uint16) error, cb2 func(port uint16) error) error {
	if err := t.AllocPort(false, cb); err != nil {
		return err
	}

	if err := t.AllocPort(false, cb2); err != nil {
		return err
	}
	return nil
}

func NewTransportManager(start, end uint16) Manager {
	utils.Assert(end > start)

	return &transportManager{
		startPort: start,
		endPort:   end,
		nextPort:  start,
	}
}
