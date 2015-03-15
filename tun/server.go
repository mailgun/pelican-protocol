package pelicantun

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
	createQueue chan *LongPoller
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
		createQueue: make(chan *LongPoller),
	}
}

// only callable from same goroutine as Start(); and
// only callled by Start() on shutting down.
func (s *ReverseProxy) finish(tunnelMap *map[string]*LongPoller) {
	s.web.Stop()
	po("rev: s.web.Stop() has returned.  s.web = %p <<<<<<<<\n", s.web)

	// close all our downstream connections
	for _, t := range *tunnelMap {
		t.Stop()
	}

	close(s.Done)
}

// dispatch to tunnel based on key
func (s *ReverseProxy) Start() {

	s.startExternalHttpListener()

	// start processing loop
	go func() {
		tunnelMap := make(map[string]*LongPoller)
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

				select {
				case tunnel.ClientPacketRecvd <- pp:
				case <-s.ReqStop:
					// don't deadlock
					return
				}
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
		//po("top level handler(): in '/' and '/ping' packetHandler, packet len without key: %d: making new tunnelPacket, url = '%s', http.Request r = '%#v'. r.Body = '%s'\n", len(body)-KeyLen, r.URL, *r, string(body))

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

		tunnel := NewLongPoller(s.Cfg.Dest)

		err = tunnel.Start()
		if err != nil {
			po("Server::createHandler: Could not connect to destination: '%s'.\n", err)
			http.Error(respW, fmt.Sprintf("Could not connect to destination: '%s'", err),
				http.StatusInternalServerError)
			return
		}

		key := tunnel.key
		s.createQueue <- tunnel

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
