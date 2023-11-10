package librtsp

import (
	"bufio"
	bytes2 "bytes"
	"fmt"
	"github.com/yangjiechina/avformat/librtsp/sdp"
	"github.com/yangjiechina/avformat/utils"
	"net"
	"net/http"
	"net/textproto"
	url2 "net/url"
	"strconv"
	"strings"
	"sync"
)

type Setup string

const (
	SetupOptions  = "1"
	SetupDescribe = "2"
	SetupSetup    = "3"
	SetupPlay     = "4"
	SetupTeardown = "5"
	SetupTearAuth = "6"
)

type OnRTPPacketHandler func(mediaType utils.AVMediaType, data []byte)

type Puller struct {
	url       string
	buffer    []byte
	version   string
	host      string
	port      int
	transport utils.Transport
	username  string
	password  string
	session   string

	medias    []*Server
	state     Setup
	setupLock sync.Mutex
	handler   OnRTPPacketHandler
}

func NewPuller(h OnRTPPacketHandler) *Puller {
	return &Puller{buffer: make([]byte, 1024), handler: h}
}

func parseTransportHeader(header string) map[string]string {
	split := strings.Split(header, ";")
	params := make(map[string]string, 10)
	if len(split) > 2 {
		params["mediaProtocol"] = split[0]
		params["transportProtocol"] = split[1]
		for _, s := range split[2:] {
			space := strings.TrimSpace(s)
			index := strings.Index(strings.TrimSpace(space), "=")
			if index == -1 {
				params["space"] = ""
			} else {
				params[space[:index]] = space[index+1:]
			}
		}
	}

	return params
}

func (p *Puller) OnPacketHandler(_ net.Conn, data []byte) {
	p.setupLock.Lock()
	defer p.setupLock.Unlock()

	reader := bufio.NewReader(bytes2.NewReader(data))
	tp := textproto.NewReader(reader)
	line, err := tp.ReadLine()
	split := strings.Split(line, " ")
	if len(split) < 3 {
		panic(fmt.Errorf("unknow response line of response:%s", line))
	}
	protocol := split[0]
	code := split[1]
	reason := split[2]

	println(protocol)
	println(code)
	println(reason)

	header, err := tp.ReadMIMEHeader()
	if err != nil {
		panic(err)
	}

	if p.state == SetupOptions {
		p.describe("")
	} else if p.state == SetupDescribe {
		if code == strconv.Itoa(http.StatusUnauthorized) {
			if params, err := parseWWWAuthenticateHeader(header.Get("www-Authenticate")); err != nil {
				panic(err)
			} else {
				params["uri"] = p.url
				params["username"] = p.username
				credentials, err := generateCredentials(params, p.password)
				if err != nil {
					panic(err)
				}
				p.describe(credentials)
			}

		} else {
			p.setup(data[len(data)-reader.Buffered():])
		}
	} else if p.state == SetupSetup {
		transport := header.Get("Transport")
		session := header.Get("Session")
		//if session == "" {
		//	panic("session must be contained in the setup response")
		//}
		params := parseTransportHeader(transport)
		if strings.Contains(strings.ToUpper(params["mediaProtocol"]), "TCP") {

		}
		_ = params["transportProtocol"]
		_ = params["destination"]
		source := params["source"]
		serverPort := params["server_port"]
		clientPort := params["client_port"]
		if params["transportProtocol"] != "unicast" || serverPort == "" {
			p.teardown(session)
		} else {
			p.session = session
			if err := p.play(source, serverPort, clientPort, session); err != nil {
				println(err)
				p.teardown(session)
			}
		}
	}
}

func (p *Puller) OnDisconnectedHandler(conn net.Conn, err error) {

}

func (p *Puller) Open(url string) error {
	parse, err := url2.Parse(url)
	if err != nil {
		return err
	}
	p.version = "1.0"
	//rtsp 2.0
	if parse.User != nil {
		password, b := parse.User.Password()
		username := parse.User.Username()
		if b {
			url = strings.ReplaceAll(url, fmt.Sprintf("%s:%s@", username, password), "")
		} else {
			url = strings.ReplaceAll(url, fmt.Sprintf("%s@", username), "")
		}
		p.password = password
		p.username = username
	} else {
		p.version = "1.0"
	}

	port := parse.Port()
	if port == "" {
		p.port = DefaultPort
	} else {
		i, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		p.port = i
	}
	p.host = parse.Hostname()
	p.url = url
	client, err := utils.NewTCPClient(nil, p.host, p.port)
	if err != nil {
		return err
	}

	client.SetOnPacketHandler(p.OnPacketHandler)
	client.SetOnDisconnectedHandler(p.OnDisconnectedHandler)
	go client.Read()
	p.transport = client
	return p.options()
}

func (p *Puller) options() error {
	p.state = SetupOptions
	request := Request{
		method:  "OPTIONS",
		url:     p.url,
		version: p.version,
		header:  make(map[string]string, 2),
	}
	request.header["CSeq"] = SetupOptions
	request.header["User-Agent"] = "avformat/librtsp"

	bytes := request.toBytes(p.buffer)
	_, err := p.transport.Write(p.buffer[:bytes])
	return err
}

func (p *Puller) describe(authHeader string) error {
	p.state = SetupDescribe
	request := Request{
		method:  "DESCRIBE",
		url:     p.url,
		version: p.version,
		header:  make(map[string]string, 2),
	}
	request.header["CSeq"] = SetupDescribe
	request.header["Accept"] = "application/sdp"
	if authHeader != "" {
		request.header["Authorization"] = authHeader
	}

	bytes := request.toBytes(p.buffer)
	_, err := p.transport.Write(p.buffer[:bytes])
	return err
}

func (p *Puller) setup(body []byte) error {
	var sd sdp.SessionDescription
	err := sd.Unmarshal(body)
	if err != nil {
		return err
	}

	var tracks []struct {
		url       string
		mediaType utils.AVMediaType
	}
	if len(sd.MediaDescriptions) == 0 {
		tracks = append(tracks, struct {
			url       string
			mediaType utils.AVMediaType
		}{url: p.url, mediaType: utils.AVMediaTypeVideo})
	} else {
		for _, description := range sd.MediaDescriptions {
			for _, attribute := range description.Attributes {
				if "control" == attribute.Key {
					var mediaType utils.AVMediaType
					if "video" == strings.ToLower(description.MediaName.Media) {
						mediaType = utils.AVMediaTypeVideo
					} else {
						mediaType = utils.AVMediaTypeAudio
					}
					tracks = append(tracks, struct {
						url       string
						mediaType utils.AVMediaType
					}{url: attribute.Value, mediaType: mediaType})
					break
				}
			}
		}
	}

	for _, track := range tracks {
		p.state = SetupSetup
		request := Request{
			method:  "SETUP",
			url:     track.url,
			version: p.version,
			header:  make(map[string]string, 2),
		}
		request.header["CSeq"] = SetupSetup
		request.header["Accept"] = "application/sdp"

		server, err := CreateServer()
		if err != nil {
			return err
		}
		transport := fmt.Sprintf("%s;%s;client_port=%d-%d", "RTP/AVP", "unicast", server.rtp.ListenPort(), server.rtcp.ListenPort())
		request.header["Transport"] = transport
		bytes := request.toBytes(p.buffer)
		if _, err = p.transport.Write(p.buffer[:bytes]); err != nil {
			return err
		}

		server.mediaType = track.mediaType
		p.medias = append(p.medias, server)
		server.rtp.SetOnPacketHandler(func(conn net.Conn, data []byte) {
			if p.handler != nil {
				p.handler(server.mediaType, data)
			}
		})
		return err
	}
	return nil
}

func (p *Puller) play(serverAddr, serverPort, clientPort, session string) error {
	split := strings.Split(serverPort, "-")
	if len(split) != 2 {
		return fmt.Errorf("invalid format of the server port:%s", serverPort)
	}

	var count int
	for j := 0; j < len(p.medias); j++ {
		media := p.medias[j]
		if media.serverPort[0] != 0 {
			count++
		}
		if media.clientPort != clientPort {
			continue
		}
		for i := 0; i < 2; i++ {
			atoi, err := strconv.Atoi(split[i])
			if err != nil {
				return err
			}
			media.serverPort[i] = atoi
		}

		media.serverAddr = serverAddr
		media.traversal()
		count++
	}

	if count == len(p.medias) {
		p.state = SetupPlay
		request := Request{
			method:  "PLAY",
			url:     p.url,
			version: p.version,
			header:  make(map[string]string, 2),
		}
		request.header["CSeq"] = SetupPlay
		request.header["Session"] = session
		bytes := request.toBytes(p.buffer)
		if _, err := p.transport.Write(p.buffer[:bytes]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Puller) teardown(session string) error {
	p.state = SetupTeardown
	request := Request{
		method:  "TEARDOWN",
		url:     p.url,
		version: p.version,
		header:  make(map[string]string, 2),
	}
	request.header["CSeq"] = SetupTeardown
	request.header["Session"] = session

	bytes := request.toBytes(p.buffer)
	_, err := p.transport.Write(p.buffer[:bytes])
	return err
}
