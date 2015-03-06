package pelicantun

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type ReverseProxyConfig struct {
	Listen addr
	Dest   addr
}

// one ReverseProxy can contain many tunnels
type ReverseProxy struct {
	Cfg     ReverseProxyConfig
	Done    chan bool
	ReqStop chan bool
	web     *WebServer

	packetQueue chan tunnelPacket
	createQueue chan *tunnel
}

func (p *ReverseProxy) Stop() {
	p.ReqStop <- true
	<-p.Done
}

func NewReverseProxy(cfg ReverseProxyConfig) *ReverseProxy {

	// get an available port
	if cfg.Listen.Port == 0 {
		cfg.Listen.Port = GetAvailPort()
	}
	if cfg.Listen.Ip == "" {
		cfg.Listen.Ip = "127.0.0.1"
	}

	return &ReverseProxy{
		Cfg:         cfg,
		Done:        make(chan bool),
		ReqStop:     make(chan bool),
		packetQueue: make(chan tunnelPacket),
		createQueue: make(chan *tunnel),
	}
}

func (s *ReverseProxy) Start() {

	s.startExternalHttpListener()

	// start processing loop
	go func() {
		po("ReverseProxy::Start(), aka tunnelMuxer started\n")
		tunnelMap := make(map[string]*tunnel)
		for {
			select {
			case pp := <-s.packetQueue:
				key := make([]byte, KeyLen)
				// read key
				//n, err := pp.req.Body.Read(key)
				if len(pp.body) < KeyLen {
					log.Printf("Couldn't read key, not enough bytes in body. len(body) = %d\n", len(pp.body))
					continue
				}
				copy(key, pp.body)

				po("tunnelMuxer: from pp <- packetQueue, we read key '%x'\n", key)
				// find tunnel
				p, ok := tunnelMap[string(key)]
				if !ok {
					log.Printf("Couldn't find tunnel for key = '%x'", key)
					continue
				}
				// handle
				po("tunnelMuxer found tunnel for key '%x'\n", key)
				p.receiveOnePacket(pp)

			case p := <-s.createQueue:
				po("tunnelMuxer: got p=%p on <-createQueue\n", p)
				tunnelMap[p.key] = p
				po("tunnelMuxer: after adding key '%x', tunnelMap is now: '%#v'\n", p.key, tunnelMap)

			case <-s.ReqStop:
				s.web.Stop()
				close(s.Done)
				return
			}
		}
		po("tunnelMuxer done\n")
	}()
}

func (s *ReverseProxy) startExternalHttpListener() {

	// packetHandler
	packetHandler := func(c http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		panicOn(err)
		po("top level handler(): in '/' and '/ping' handler, packet len without key: %d: making new tunnelPacket, http.Request r = '%#v'. r.Body = '%s'\n", len(body)-KeyLen, *r, string(body))

		pp := tunnelPacket{
			resp:    c,
			request: r,
			body:    body, // includes key of KeyLen in prefix
			done:    make(chan bool),
		}
		s.packetQueue <- pp
		<-pp.done // wait until done before returning, as this will return anything written to c to the client.
	}

	// createHandler
	createHandler := func(c http.ResponseWriter, r *http.Request) {

		_, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(c, "Could not read r.Body",
				http.StatusInternalServerError)
			return
		}

		key := genKey()
		po("in createhandler(): Server::createHandler generated key '%s'\n", key)

		p, err := NewTunnel(key, s.Cfg.Dest.IpPort)
		if err != nil {
			http.Error(c, "Could not connect",
				http.StatusInternalServerError)
			return
		}
		po("Server::createHandler about to send createQueue <- p, where p = %p\n", p)
		s.createQueue <- p
		po("Server::createHandler(): sent createQueue <- p.\n")

		c.Write([]byte(key))
		po("Server::createHandler done.\n")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", packetHandler)
	mux.HandleFunc("/create", createHandler)

	webcfg := WebServerConfig{IP: s.Cfg.Listen.Ip, Port: s.Cfg.Listen.Port}
	s.web = NewWebServer(webcfg, mux)
	fmt.Printf("\n about to w.web.Start() with webcfg = '%#v'\n", webcfg)
	s.web.Start()

}

const (
	readTimeoutMsec = 1000
)

// a tunnel represents a 1:1, one client to one server connection.
// a ReverseProxy can have many of these.
type tunnel struct {
	C chan tunnelPacket

	// server issues a unique key for the connection, which allows multiplexing
	// of multiple client connections from this one ip if need be.
	// The ssh integrity checks inside the tunnel prevent malicious tampering.
	key       string
	conn      net.Conn
	recvCount int
}

type tunnelPacket struct {
	resp    http.ResponseWriter
	request *http.Request
	body    []byte
	done    chan bool
}

// print out shortcut
var po = VPrintf

func NewTunnel(key, destAddr string) (p *tunnel, err error) {
	po("starting with NewTunnel\n")
	p = &tunnel{
		C:         make(chan tunnelPacket),
		key:       key,
		recvCount: 0,
	}
	log.Println("Attempting connect", destAddr)
	p.conn, err = net.Dial("tcp", destAddr)
	panicOn(err)

	err = p.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	panicOn(err)

	log.Println("ResponseWriter directed to ", destAddr)
	po("done with NewTunnel\n")
	return
}

func (p *tunnel) receiveOnePacket(pp tunnelPacket) {
	p.recvCount++
	po("\n ====================\n server tunnel.recvCount = %d    len(pp.body)= %d\n ================\n", p.recvCount, len(pp.body))

	po("in tunnel::handle(pp) with pp = '%#v'\n", pp)
	// read from the request body and write to the ResponseWriter
	writeMe := pp.body[KeyLen:]
	n, err := p.conn.Write(writeMe)
	if n != len(writeMe) {
		log.Printf("tunnel::handle(pp): could only write %d of the %d bytes to the connection. err = '%v'", n, len(pp.body), err)
	} else {
		po("tunnel::handle(pp): wrote all %d bytes of writeMe to the final (sshd server) connection: '%s'.", len(writeMe), string(writeMe))
	}
	pp.request.Body.Close()
	if err == io.EOF {
		p.conn = nil
		log.Printf("tunnel::handle(pp): EOF for key '%x'", p.key)
		return
	}
	// read out of the buffer and write it to conn
	pp.resp.Header().Set("Content-type", "application/octet-stream")
	// temp for debug: n64, err := io.Copy(pp.resp, p.conn)

	b500 := make([]byte, 1<<17)

	err = p.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	panicOn(err)

	n64, err := p.conn.Read(b500)
	if err != nil {
		// i/o timeout expected
	}
	po("\n\n server got reply from p.conn of len %d: '%s'\n", n64, string(b500[:n64]))
	_, err = pp.resp.Write(b500[:n64])
	if err != nil {
		panic(err)
	}

	// don't panicOn(err)
	log.Printf("tunnel::handle(pp): io.Copy into pp.resp from p.conn moved %d bytes", n64)
	pp.done <- true
	po("tunnel::handle(pp) done.\n")
}

func genKey() string {
	key := make([]byte, KeyLen)
	for i := 0; i < KeyLen; i++ {
		key[i] = byte(rand.Int())
	}
	return string(key)
}
