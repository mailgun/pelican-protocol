package pelicantun

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
	// _ "net/http/pprof" // side-effect: installs handlers for /debug/pprof

	"github.com/glycerine/go-tigertonic"
)

type WebServer struct {
	ServerReady chan bool // closed once server is listening on Addr
	Done        chan bool // closed when server shutdown.

	requestStop chan bool // private. Users should call Stop().

	// we use tigertonic because it actually implements graceful stopping;
	// as opposed to the built-in http library web server.
	tts *tigertonic.Server

	started bool
	stopped bool
	Cfg     WebServerConfig
}

type WebServerConfig struct {
	Listen addr
}

func NewWebServer(cfg WebServerConfig, mux *http.ServeMux) *WebServer {

	if mux == nil {
		mux = http.NewServeMux()
	}

	// get an available port
	if cfg.Listen.Port == 0 {
		cfg.Listen.Port = GetAvailPort()
	}
	if cfg.Listen.Ip == "" {
		cfg.Listen.Ip = "127.0.0.1"
	}
	cfg.Listen.SetIpPort()
	//VPrintf("hey hey: starting webserver on '%s'\n", cfg.Listen.IpPort)

	s := &WebServer{
		Cfg:         cfg,
		ServerReady: make(chan bool),
		Done:        make(chan bool),
		requestStop: make(chan bool),
	}

	s.tts = tigertonic.NewServer(s.Cfg.Listen.IpPort, mux)
	//s.tts = tigertonic.NewServer(s.Cfg.Addr, http.DefaultServeMux) // supply debug/pprof diagnostics

	return s
}

func (s *WebServer) Start() {
	if s.started {
		return
	}
	s.started = true

	go func() {
		err := s.tts.ListenAndServe()
		if nil != err {
			//log.Println(err) // accept tcp 127.0.0.1:3000: use of closed network connection
		}
		s.stopped = true
		close(s.Done)
	}()

	WaitUntilServerUp(s.Cfg.Listen.IpPort)
	close(s.ServerReady)
}

func (s *WebServer) Stop() {
	if s.stopped || s.IsStopRequested() {
		// not stopping races here, just preventing
		// panic on two serial Stops() under web_test.go failure situation.
		return
	}
	close(s.requestStop)
	s.tts.Close()
	//VPrintf("in WebServer::Stop() after s.tts.Close()\n")
	<-s.Done
	//VPrintf("in WebServer::Stop() after <-s.Done(): s.Addr = '%s'\n", s.Cfg.Listen.IpPort)

	WaitUntilServerDown(s.Cfg.Listen.IpPort)
}

func (s *WebServer) IsStopRequested() bool {
	select {
	case <-s.requestStop:
		return true
	default:
		return false
	}
}

func WaitUntilServerUp(addr string) {
	attempt := 1
	for {
		if PortIsBound(addr) {
			return
		}
		time.Sleep(500 * time.Millisecond)
		attempt++
		if attempt > 40 {
			panic(fmt.Sprintf("could not connect to server at '%s' after 40 tries of 500msec", addr))
		}
	}
}

func WaitUntilServerDown(addr string) {
	attempt := 1
	for {
		if !PortIsBound(addr) {
			return
		}
		//fmt.Printf("WaitUntilServerUp: on attempt %d, sleep then try again\n", attempt)
		time.Sleep(500 * time.Millisecond)
		attempt++
		if attempt > 40 {
			panic(fmt.Sprintf("could always connect to server at '%s' after 40 tries of 500msec", addr))
		}
	}
}

func PortIsBound(addr string) bool {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func FetchUrl(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return []byte{}, err
		}
		return contents, nil
	}
}
