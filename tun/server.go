package pelicantun

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
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

	packetQueue chan *tunnelPacket
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
	cfg.Listen.SetIpPort()

	if cfg.Dest.Port == 0 {
		cfg.Dest = NewAddr2("127.0.0.1", 80)
	}
	cfg.Dest.SetIpPort()

	fmt.Printf("in NewReverseProxy, cfg = '%#v'\n", cfg)

	return &ReverseProxy{
		Cfg:         cfg,
		Done:        make(chan bool),
		ReqStop:     make(chan bool),
		packetQueue: make(chan *tunnelPacket),
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

				po("tunnelMuxer: from pp <- packetQueue, we read key '%x'\n", pp.key)
				// find tunnel
				tunnel, ok := tunnelMap[string(pp.key)]
				if !ok {
					log.Printf("Couldn't find tunnel for key = '%x'", pp.key)
					continue
				}
				// handle
				po("tunnelMuxer found tunnel for key '%x'\n", pp.key)
				tunnel.receiveOnePacket(pp)

			case p := <-s.createQueue:
				po("tunnelMuxer: got p=%p on <-createQueue\n", p)
				tunnelMap[p.key] = p
				po("tunnelMuxer: after adding key '%x', tunnelMap is now: '%#v'\n", p.key, tunnelMap)

			case <-s.ReqStop:
				s.web.Stop()
				po("rev: s.web.Stop() has returned.  s.web = %p <<<<<<<<\n", s.web)
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
		r.Body.Close()
		panicOn(err)
		po("top level handler(): in '/' and '/ping' handler, packet len without key: %d: making new tunnelPacket, http.Request r = '%#v'. r.Body = '%s'\n", len(body)-KeyLen, *r, string(body))

		key := make([]byte, KeyLen)
		if len(body) < KeyLen {
			http.Error(c, fmt.Sprintf("Couldn't read key, not enough bytes in body. len(body) = %d\n",
				len(body)),
				http.StatusBadRequest)
			return
		}
		copy(key, body)

		s.injectPacket(c, r, body[KeyLen:], string(key))
	}

	// createHandler
	createHandler := func(respW http.ResponseWriter, r *http.Request) {

		po("Server::createHandler starting.\n")
		_, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(respW, "Could not read r.Body",
				http.StatusInternalServerError)
			return
		}

		tunnel, err := s.NewTunnel(s.Cfg.Dest.IpPort)
		if err != nil {
			po("Server::createHandler: Could not connect to destination: '%s'.\n", err)
			http.Error(respW, fmt.Sprintf("Could not connect to destination: '%s'", err),
				http.StatusInternalServerError)
			return
		}
		key := tunnel.key

		po("Server::createHandler, about to write key '%s'.\n", key)
		respW.Write([]byte(key))
		po("Server::createHandler done for key '%x'.\n", key)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", packetHandler)
	mux.HandleFunc("/create", createHandler)

	webcfg := WebServerConfig{Listen: s.Cfg.Listen}
	s.web = NewWebServer(webcfg, mux)
	//VPrintf("\n Server::createHandler(): about to w.web.Start() with webcfg = '%#v'\n", webcfg)
	s.web.Start()

}

const (
	readTimeoutMsec = 1000
)

// a tunnel represents a 1:1, one client to one server connection,
// if you ignore the socks-proxy and reverse-proxy in the middle.
// A ReverseProxy can have many tunnels, mirroring the number of
// connections on the client side to the socks proxy. The key
// distinguishes them.
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
	respdup *bytes.Buffer // duplicate resp here, to enable testing

	request *http.Request
	body    []byte
	key     string // separate from body
	done    chan bool
}

// print out shortcut
var po = VPrintf

func (rev *ReverseProxy) NewTunnel(destAddr string) (t *tunnel, err error) {
	key := genKey()

	po("ReverseProxy::NewTunnel() top. key = '%x'\n", key)
	t = &tunnel{
		C:         make(chan tunnelPacket),
		key:       string(key),
		recvCount: 0,
	}
	po("ReverseProxy::NewTunnel: Attempting connect to our target '%s'\n", destAddr)
	dialer := net.Dialer{
		Timeout:   1000 * time.Millisecond,
		KeepAlive: 30 * time.Second,
	}

	t.conn, err = dialer.Dial("tcp", destAddr)
	switch err.(type) {
	case *net.OpError:
		if strings.HasSuffix(err.Error(), "connection refused") {
			// could not reach destination
			return nil, err
		}
	default:
		panicOn(err)
	}

	err = t.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	panicOn(err)

	po("ReverseProxy::NewTunnel: ResponseWriter directed to '%s'\n", destAddr)

	po("ReverseProxy::NewTunnel about to send createQueue <- t, where t = %p\n", t)
	rev.createQueue <- t
	po("ReverseProxy::NewTunnel: sent createQueue <- t.\n")

	po("ReverseProxy::NewTunnel done.\n")
	return
}

func (s *ReverseProxy) injectPacket(c http.ResponseWriter, r *http.Request, body []byte, key string) ([]byte, error) {
	pack := &tunnelPacket{
		resp:    c,
		respdup: new(bytes.Buffer),
		request: r,
		body:    body, // body no longer includes key of KeyLen in prefix
		done:    make(chan bool),
		key:     key,
	}

	select {
	case s.packetQueue <- pack:

	case <-s.Done:
		// don't deadlock
	case <-s.ReqStop:
		// don't deadlock
	}

	select {
	// wait until done before returning, as this will return anything written to c to the client.
	case <-pack.done:
		// okay, writing to c is done.

	case <-s.Done:
		// don't deadlock
	case <-s.ReqStop:
		// don't deadlock
	}
	return pack.respdup.Bytes(), nil
}

// receiveOnePacket() closes pack.done after:
//   writing pack.body to t.conn;
//   reading from t.conn some bytes if available;
//   and writing those bytes to pack.resp
//
//  t.conn is the downstream ultimate webserver
//  destination.
//
func (t *tunnel) receiveOnePacket(pack *tunnelPacket) {
	t.recvCount++
	po("\n ====================\n server tunnel.recvCount = %d    len(pack.body)= %d\n ================\n", t.recvCount, len(pack.body))

	po("in tunnel::handle(pack) with pack = '%#v'\n", pack)
	// read from the request body and write to the ResponseWriter
	n, err := t.conn.Write(pack.body)
	if n != len(pack.body) {
		log.Printf("tunnel::handle(pack): could only write %d of the %d bytes to the connection. err = '%v'", n, len(pack.body), err)
	} else {
		po("tunnel::handle(pack): wrote all %d bytes of pack.body to the final (sshd server) connection: '%s'.", len(pack.body), string(pack.body))
	}
	// done in packetHandler now: pack.request.Body.Close()
	if err == io.EOF {
		t.conn.Close() // let the server shutdown sooner rather than holding open the connection.
		t.conn = nil
		log.Printf("tunnel::handle(pack): EOF for key '%x'", t.key)
		return
	}
	// read out of the buffer and write it to conn
	pack.resp.Header().Set("Content-type", "application/octet-stream")
	// temp for debug: n64, err := io.Copy(pack.resp, t.conn)

	b500 := make([]byte, 1<<17) // 128KB

	err = t.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	panicOn(err)

	n64, err := t.conn.Read(b500)
	if err != nil {
		// i/o timeout expected
	}
	po("\n\n server got reply from t.conn of len %d: '%s'\n", n64, string(b500[:n64]))
	_, err = pack.resp.Write(b500[:n64])
	if err != nil {
		panic(err)
	}

	_, err = pack.respdup.Write(b500[:n64])
	if err != nil {
		panic(err)
	}

	// don't panicOn(err)
	log.Printf("tunnel::handle(pack): io.Copy into pack.resp from t.conn moved %d bytes", n64)
	close(pack.done)
	po("tunnel::handle(pack) done.\n")
}

func genKey() []byte {
	key := make([]byte, KeyLen)
	for i := 0; i < KeyLen; i++ {
		key[i] = byte(rand.Int())
	}
	return key
}
