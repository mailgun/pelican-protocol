package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/glycerine/go-tigertonic"

	_ "expvar"       // Imported for side-effect of handling /debug/vars.
	"net/http/pprof" // Imported for side-effect of handling /debug/pprof.
)

// a simple web server that serves a static page, for testing of forwards.

// =================

func mainExample() {
	fmt.Printf("webserv main running.\n")
	w := NewWebServer("127.0.0.1:7708", nil)
	w.Start()
	select {}
	// ...
	w.Stop()
}

// =============

type WebServer struct {
	Addr        string
	ServerReady chan bool      // closed once server is listening on Addr
	RequestStop chan bool      // close this to tell server to shutdown
	Done        chan bool      // recv on this to know that server is indeed shutdown
	StopSigCh   chan os.Signal // signals will send on this to request stop
	LastReqBody string
	Tts         *tigertonic.Server
	mux         *tigertonic.TrieServeMux
	cfg         WebConfig
	host        string
	pid         int
}

type WebConfig struct{}

func (s *WebServer) Stop() error {
	close(s.RequestStop)
	s.Tts.Close()
	err := WaitUntilServerDown(s.Addr, 0)
	<-s.Done
	return err
}

func (s *WebServer) IsStopRequested() bool {
	select {
	case <-s.RequestStop:
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
		time.Sleep(50 * time.Millisecond)
		attempt++
		if attempt > 40 {
			panic(fmt.Sprintf("could not connect to server at '%s' after 40 tries of 50msec", addr))
		}
	}
}

func WaitUntilServerDown(addr string, ntries int) error {
	attempt := 1
	tries := 80
	if ntries > 0 {
		tries = ntries
	}
	durBetweenTries := 50 * time.Millisecond
	for {
		if !PortIsBound(addr) {
			return nil
		}
		//fmt.Printf("WaitUntilServerUp: on attempt %d, sleep then try again\n", attempt)
		time.Sleep(durBetweenTries)
		attempt++
		if attempt > tries {
			return fmt.Errorf("could always connect to server at '%s' after %d tries of %v", addr, tries, durBetweenTries)
		}
	}
	return nil
}

func PortIsBound(addr string) bool {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func NewWebServer(addr string, cfg *WebConfig) *WebServer {

	// the global state in metrics::DefaultRegistry is a problem: unregister everything first.
	/*
		metrics.Unregister("myCutlassHandler")
	*/

	VPrintf("NewWebServer called for addr = %v\n", addr)

	host, _ := os.Hostname()

	s := &WebServer{
		Addr:        addr,
		ServerReady: make(chan bool),
		RequestStop: make(chan bool),
		Done:        make(chan bool),
		StopSigCh:   make(chan os.Signal),
		host:        host,
		pid:         os.Getpid(),
	}
	if cfg != nil {
		s.cfg = *cfg
	}

	//	frontBody, err := ioutil.ReadFile("web/html/anon_or_ident.html")
	//	panicOn(err)

	FrontHandler := func(w http.ResponseWriter, r *http.Request) {

		frontBody, err := ioutil.ReadFile("web/html/anon_or_ident.html")
		panicOn(err)

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		//browserCacheSeconds := 0
		//w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, must-revalidate", browserCacheSeconds))
		// or just simply better for development:
		w.Header().Set("Cache-Control", "no-cache")

		fmt.Fprintf(w, "<html>")
		title := `Pelican-protocol on guard: New server-host-key detected`
		fmt.Fprintf(w, `<head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"><title>%s</title>`, title)
		s.AddScriptIncludes(w)
		fmt.Fprintf(w, `</head>`)
		io.Copy(w, bytes.NewBuffer(frontBody))
		fmt.Fprintf(w, "\n<html>")

	}

	s.mux = tigertonic.NewTrieServeMux()

	// initial webpage dispensing
	s.mux.HandleFunc("GET", "/", FrontHandler)
	/*
		s.mux.HandleNamespace("/css", makeVerbDirHandler("web/css"))
		s.mux.HandleNamespace("/script", makeVerbDirHandler("web/script"))
		s.mux.HandleNamespace("/media", makeVerbDirHandler("web/media"))
		s.mux.HandleNamespace("/images", makeVerbDirHandler("web/media"))
		s.mux.HandleFunc("GET", "/favicon.ico", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "web/media/favicon.ico") })
	*/
	//s.RetainDebugRoutes()

	// logged:
	//s.Tts = tigertonic.NewServer(addr, tigertonic.Logged(s.mux, nil))

	// non-logged, faster.
	s.Tts = tigertonic.NewServer(addr, s.mux)

	return s
}

func (s *WebServer) Start() *WebServer {

	go func() {
		err := s.Tts.ListenAndServe()
		if nil != err {
			//log.Println(err) // accept tcp 127.0.0.1:3000: use of closed network connection
		}
		close(s.Done)
	}()

	WaitUntilServerUp(s.Addr)
	close(s.ServerReady)
	return s
}

// =====

type KeepDebug struct{}

func (k *KeepDebug) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// repair the stripped path that HandleNamespace gives us so
	// that pprof.Index knows what to do.
	r.URL.Path = "/debug/pprof" + r.URL.Path
	pprof.Index(w, r)
}

func (s *WebServer) RetainDebugRoutes() {
	s.mux.HandleNamespace("/debug/pprof", &KeepDebug{})
	s.mux.HandleFunc("GET", "/debug/pprof/cmdline", pprof.Cmdline)
	s.mux.HandleFunc("GET", "/debug/pprof/symbol", pprof.Symbol)
	s.mux.Handle("GET", "/debug/vars", http.DefaultServeMux)
}

func (s *WebServer) AddScriptIncludes(w http.ResponseWriter) {
	fmt.Fprintf(w, `<script type="text/javascript" language="javascript" src="/script/jquery.js"></script>`)
	fmt.Fprintf(w, `<script type="text/javascript" language="javascript" src="/script/jquery.dataTables.min.js"></script>`)

	fmt.Fprintf(w, `<link rel="stylesheet" type="text/css" href="/css/jquery.dataTables.min.css"><style type="text/css" class="init"></style>`)

	fmt.Fprintf(w, `<style>
      table {
        text-align: center;
      }
      </style>
     <link rel="stylesheet" type="text/css" href="https://eafdbc63c97ce6bec9ef-b0a668e5876bef6fe25684caf71db405.ssl.cf1.rackcdn.com/v1-latest/canon.min.css">
     `)
}
