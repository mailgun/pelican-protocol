package pelicantun

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type ReverseProxyConfig struct {
	Listen      Addr
	Dest        Addr
	LongPollDur time.Duration
}

// one ReverseProxy can contain many tunnels
type ReverseProxy struct {
	Cfg     ReverseProxyConfig
	Done    chan bool
	reqStop chan bool
	web     *WebServer

	packetQueue chan *tunnelPacket
	createQueue chan *LongPoller
	mut         sync.Mutex
	closeKeyReq chan string
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *ReverseProxy) RequestStop() bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	select {
	case <-s.reqStop:
		return false
	default:
		close(s.reqStop)
		po("ReverseProxy just closed s.reqStop")
		return true
	}
}

func (p *ReverseProxy) Stop() {
	p.RequestStop()
	<-p.Done
}

func NewReverseProxy(cfg ReverseProxyConfig) *ReverseProxy {

	if cfg.LongPollDur == 0 {
		cfg.LongPollDur = time.Second * 30
	}

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
		reqStop:     make(chan bool),
		packetQueue: make(chan *tunnelPacket),
		createQueue: make(chan *LongPoller),
		closeKeyReq: make(chan string),
	}
}

// only callable from same goroutine as Start(); and
// only callled by Start() on shutting down.
func (s *ReverseProxy) finish(tunnelMap *map[string]*LongPoller) {

	// the web stop is hanging: and thus hanging up finishing of the 010 / 01a tests.

	// the tunnels/ServerRW/LongPoller will be holding open
	// web connections, so if try to shutdown the web first,
	// we'll deadlock until those connections timeout after 60 seconds.
	// So, shutdown the LongPollers first.

	// close all our downstream connections
	for _, t := range *tunnelMap {
		t.Stop()
		po("%p rev stopped LongPoller %p", s, t)
	}

	// stop the web from accepting new connections, before we tell
	// all the tunnelMap connections to stop as well.
	s.web.Stop()
	po("rev: s.web.Stop() has returned.  s.web = %p <<<<<<<<\n", s.web)

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
			case closekey := <-s.closeKeyReq:
				tunnel, ok := tunnelMap[closekey]
				if !ok {
					log.Printf("Couldn't find tunnel for key = '%x'", closekey)
					continue
				}
				close(tunnel.CloseKeyChan)
				delete(tunnelMap, closekey)

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
				case <-s.reqStop:
					// don't deadlock
					return
				}
			case p := <-s.createQueue:
				po("ReverseProxy::Start(): got tunnelPacket  p=%p on <-createQueue\n", p)
				tunnelMap[p.key] = p
				po("ReverseProxy::Start(): after adding key '%s' to tunnelMap", string(p.key[:5]))

			case <-s.reqStop:
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

		if len(body) < HeaderLen || !legitPelicanKey {
			// pass through here to the downstream webserver directly, by-passing pelican protocol stuff

			// here we could act simply as a pass through proxy

			// or instead: we'll assume that such multiplexing has already been handled for us up front.
			// e.g.
			http.Error(c, fmt.Sprintf("Pelican Protocol key not found or couldn't read key, not enough bytes in body. len(body) = %d\n",
				len(body)),
				http.StatusBadRequest)
			return
		}
		s.injectPacket(c, r, body[HeaderLen:], string(key), BytesToSerial(body[KeyLen:KeyLen+SerialLen]))
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

		tunnel := NewLongPoller(s.Cfg.Dest, s.Cfg.LongPollDur)
		po("%p '%s' LongPoller NewLongPoller just called, returning me. RemoteAddr: '%s'", tunnel, string(tunnel.key[:5]), r.RemoteAddr)

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
		po("Server::createHandler done for key '%s'...\n", string(key[:5]))
	}

	// closeKeyHandler
	closeKeyHandler := func(respW http.ResponseWriter, r *http.Request) {
		po("Server::closeKeyHandler starting.\n")

		body, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		panicOn(err)
		key := make([]byte, KeyLen)
		copy(key, body)
		legitPelicanKey := IsLegitPelicanKey(key)

		if len(body) < KeyLen || !legitPelicanKey {
			// pass through here to the downstream webserver directly, by-passing pelican protocol stuff

			// here we could act simply as a pass through proxy

			// or instead: we'll assume that such multiplexing has already been handled for us up front.
			// e.g.
			http.Error(respW, fmt.Sprintf("Pelican Protocol key not found or couldn't read key, not enough bytes in body. len(body) = %d\n",
				len(body)),
				http.StatusBadRequest)
			return
		}

		skey := string(key)
		select {
		case s.closeKeyReq <- skey:
			po("rev server's closeKeyHandler passed along close key '%s' to s.closeKeyReq", skey[:5])
		case <-s.reqStop:
			//don't deadlock
		}
		po("Server::closeKeyHandler done for key '%s'...\n", string(key[:5]))
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", packetHandler)
	mux.HandleFunc("/create", createHandler)
	mux.HandleFunc("/closekey", closeKeyHandler)

	webcfg := WebServerConfig{
		Listen:      s.Cfg.Listen,
		ReadTimeout: 1 * time.Second,
	}
	var err error
	s.web, err = NewWebServer(webcfg, mux)
	panicOn(err)
	//VPrintf("\n Server::createHandler(): about to w.web.Start() with webcfg = '%#v'\n", webcfg)
	s.web.Start("ReverseProxy")

}

const (
	readTimeoutMsec = 1000
)

type tunnelPacket struct {
	resp    http.ResponseWriter
	respdup *bytes.Buffer // duplicate resp here, to enable testing

	request *http.Request
	reqBody []byte
	key     string // separate from reqBody
	done    chan bool

	requestSerial int64 // order the sends with content by serial number
	replySerial   int64 // order the replies by serial number. Empty replies get serial number -1.
}

// print out shortcut
func po(format string, a ...interface{}) {
	if Verbose {
		TSPrintf("\n\n"+format+"\n\n", a...)
	}
}

func (s *ReverseProxy) injectPacket(c http.ResponseWriter, r *http.Request, body []byte, key string, reqSerial int64) ([]byte, error) {
	pack := &tunnelPacket{
		resp:          c,
		respdup:       new(bytes.Buffer),
		request:       r,
		reqBody:       body, // body no longer includes key of KeyLen in prefix
		done:          make(chan bool),
		key:           key,
		requestSerial: reqSerial,
	}

	select {
	case s.packetQueue <- pack:

	case <-s.Done:
		// don't deadlock
	case <-s.reqStop:
		// don't deadlock
	}

	select {
	// wait until done before returning, as this will return anything written to c to the client.
	case <-pack.done:
		// okay, writing to c is done.

	case <-s.Done:
		// don't deadlock
	case <-s.reqStop:
		// don't deadlock
	}
	return pack.respdup.Bytes(), nil
}
