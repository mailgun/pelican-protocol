package pelicantun

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type ReverseProxyConfig struct {
	Listen Addr
	Dest   Addr
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
		cfg.Listen.Ip = "0.0.0.0"
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

// only callable from same goroutine as Start(); and
// only callled by Start() on shutting down.
func (s *ReverseProxy) finish(tunnelMap *map[string]*tunnel) {
	s.web.Stop()
	po("rev: s.web.Stop() has returned.  s.web = %p <<<<<<<<\n", s.web)

	// close all our downstream connections
	for _, t := range *tunnelMap {
		t.rw.Close()
	}

	close(s.Done)
}

func (s *ReverseProxy) Start() {

	s.startExternalHttpListener()

	// start processing loop
	go func() {
		tunnelMap := make(map[string]*tunnel)
		defer func() { s.finish(&tunnelMap) }()
		po("ReverseProxy::Start(), aka tunnelMuxer started\n")
		for {
			select {
			case pp := <-s.packetQueue:

				//po("tunnelMuxer: from pp <- packetQueue, we read key '%x'...\n", pp.key)
				// find tunnel
				tunnel, ok := tunnelMap[string(pp.key)]
				if !ok {
					log.Printf("Couldn't find tunnel for key = '%x'", pp.key)
					continue
				}
				// handle
				//po("tunnelMuxer found tunnel for key '%x'\n", pp.key)
				tunnel.receiveOnePacket(pp)

			case p := <-s.createQueue:
				po("tunnelMuxer: got p=%p on <-createQueue\n", p)
				tunnelMap[p.key] = p
				//po("tunnelMuxer: after adding key '%x'..., tunnelMap is now: '%#v'\n", p.key[:5], tunnelMap)

			case <-s.ReqStop:
				// deferred finish() takes care of the rest
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
		po("top level handler(): in '/' and '/ping' packetHandler, packet len without key: %d: making new tunnelPacket, url = '%s', http.Request r = '%#v'. r.Body = '%s'\n",
			len(body)-KeyLen, r.URL, *r, string(body))

		key := make([]byte, KeyLen)
		copy(key, body)
		legitPelicanKey := IsLegitPelicanKey(key)

		if len(body) < KeyLen || !legitPelicanKey {
			// pass through here to the downstream webserver directly, by-passing pelican protocol stuff

			// here we could act simply as a pass through proxy

			// or instead: we'll assume that such multiplexing has already been handled for us up front.
			// e.g.
			http.Error(c, fmt.Sprintf("Pelican Protocol key not found or couldn't read key, not enough bytes in body. len(body) = %d\n",
				len(body)),
				http.StatusBadRequest)
			return
		}

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

		respW.Write([]byte(key))
		po("Server::createHandler done for key '%x'...\n", key[:5])
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", packetHandler)
	mux.HandleFunc("/create", createHandler)

	webcfg := WebServerConfig{Listen: s.Cfg.Listen}
	var err error
	s.web, err = NewWebServer(webcfg, mux, DefaultWebReadTimeout)
	panicOn(err)
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

	// server issues a unique key for the connection, which allows multiplexing
	// of multiple client connections from this one ip if need be.
	// The ssh integrity checks inside the tunnel prevent malicious tampering.
	key       string
	dnConn    net.Conn // downstream, e.g. to tcp speaking sshd server
	recvCount int
	rw        *RW // manage the goroutines that read and write dnConn
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
	key := GenPelicanKey()

	po("ReverseProxy::NewTunnel() top. key = '%x'...\n", key[:5])
	t = &tunnel{
		key:       string(key),
		recvCount: 0,
	}
	po("ReverseProxy::NewTunnel: Attempting connect to our target '%s'\n", destAddr)
	dialer := net.Dialer{
		Timeout:   1000 * time.Millisecond,
		KeepAlive: 30 * time.Second,
	}

	t.dnConn, err = dialer.Dial("tcp", destAddr)
	switch err.(type) {
	case *net.OpError:
		if strings.HasSuffix(err.Error(), "connection refused") {
			// could not reach destination
			return nil, err
		}
	default:
		panicOn(err)
	}

	t.rw = NewRW(t.dnConn, 0)
	t.rw.Start()

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
//   writing pack.body to t.dnConn;
//   reading from t.dnConn some bytes if available;
//   and writing those bytes to pack.resp
//
//  t.dnConn is the downstream ultimate webserver
//  destination.
//
func (t *tunnel) receiveOnePacket(pack *tunnelPacket) {
	t.recvCount++
	po("\n ====================\n server tunnel.recvCount = %d    len(pack.body)= %d\n ================\n", t.recvCount, len(pack.body))

	po("in tunnel::handle(pack) with pack = '%#v'\n", pack)
	// read from the request body and write to the ResponseWriter
	n, err := t.dnConn.Write(pack.body)
	if n != len(pack.body) {
		log.Printf("tunnel::handle(pack): could only write %d of the %d bytes to the connection. err = '%v'", n, len(pack.body), err)
	} else {
		po("tunnel::handle(pack): wrote all %d bytes of pack.body to the final (sshd server) connection: '%s'.", len(pack.body), string(pack.body))
	}
	// done in packetHandler now: pack.request.Body.Close()
	if err == io.EOF {
		t.dnConn.Close() // let the server shutdown sooner rather than holding open the connection.
		t.dnConn = nil
		log.Printf("tunnel::handle(pack): EOF for key '%x'", t.key)
		return
	}
	// read out of the buffer and write it to dnConn
	pack.resp.Header().Set("Content-type", "application/octet-stream")
	// temp for debug: n64, err := io.Copy(pack.resp, t.dnConn)

	b500 := make([]byte, 1<<17) // 128KB

	err = t.dnConn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	panicOn(err)

	n64, err := t.dnConn.Read(b500)
	if err != nil {
		// i/o timeout expected
	}
	po("\n\n server got reply from t.dnConn of len %d: '%s'\n", n64, string(b500[:n64]))
	_, err = pack.resp.Write(b500[:n64])
	if err != nil {
		panic(err)
	}

	_, err = pack.respdup.Write(b500[:n64])
	if err != nil {
		panic(err)
	}

	// don't panicOn(err)
	log.Printf("tunnel::handle(pack): io.Copy into pack.resp from t.dnConn moved %d bytes.\n", n64)
	close(pack.done)
	po("tunnel::handle(pack) done.\n")
}
