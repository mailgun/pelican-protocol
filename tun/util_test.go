package pelicantun

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// test utilities

var specialFastTestReadTimeout time.Duration = 500 * time.Millisecond

/*
example use of StartTestSystemWithPing() to setup a test:

	web, rev, fwd, err := StartTestSystemWithPing()
	panicOn(err)
	defer web.Stop()
	defer rev.Stop()
	defer fwd.Stop()

*/
func StartTestSystemWithPing() (*WebServer, *ReverseProxy, *PelicanSocksProxy, error) {
	// setup a mock web server that replies to ping with pong.
	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	web, err := NewWebServer(WebServerConfig{}, mux, specialFastTestReadTimeout)
	panicOn(err)
	web.Start()

	if !PortIsBound(web.Cfg.Listen.IpPort) {
		panic("web server did not come up")
	}

	// start a reverse proxy
	rev := NewReverseProxy(ReverseProxyConfig{Dest: web.Cfg.Listen})
	rev.Start()

	if !PortIsBound(rev.Cfg.Listen.IpPort) {
		panic("rev proxy not up")
	}

	// start the forward proxy, talks to the reverse proxy.
	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: rev.Cfg.Listen,
	})
	fwd.Start()
	if !PortIsBound(fwd.Cfg.Listen.IpPort) {
		panic("fwd proxy not up")
	}

	return web, rev, fwd, nil
}

func StartTestSystemWithBcast() (*BcastClient, *BcastServer, *ReverseProxy, *PelicanSocksProxy, error) {

	// start broadcast server (to test long-poll functionality/server initiated message)
	srv := NewBcastServer(Addr{})
	srv.Start()

	// start a reverse proxy
	rev := NewReverseProxy(ReverseProxyConfig{Dest: srv.Listen})
	rev.Start()

	if !PortIsBound(rev.Cfg.Listen.IpPort) {
		panic("rev proxy not up")
	}

	// start the forward proxy, talks to the reverse proxy.
	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: rev.Cfg.Listen,
	})
	fwd.Start()
	if !PortIsBound(fwd.Cfg.Listen.IpPort) {
		panic("fwd proxy not up")
	}

	// start broadcast client (to test receipt of long-polled data from server)
	cli := NewBroadcastClient(Addr{Port: srv.Listen.Port})
	cli.Start()

	return cli, srv, rev, fwd, nil
}

type BcastClient struct {
	Dest Addr

	Ready   chan bool
	ReqStop chan bool
	Done    chan bool

	MsgRecvd chan bool
	lastMsg  string
}

func NewBroadcastClient(dest Addr) *BcastClient {

	if dest.Port == 0 {
		panic("client's dest Addr setting must specify port to contact")
	}
	if dest.Ip == "" {
		dest.Ip = "0.0.0.0"
	}
	dest.SetIpPort()

	r := &BcastClient{
		MsgRecvd: make(chan bool),
		Dest:     dest,
		Ready:    make(chan bool),
		ReqStop:  make(chan bool),
		Done:     make(chan bool),
	}

	return r
}

func (cli *BcastClient) LastMsgReceived() string {
	return cli.lastMsg
}

func (cli *BcastClient) Start() {

	conn, err := net.Dial("tcp", cli.Dest.IpPort)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write([]byte("hello"))
	panicOn(err)

	buf := make([]byte, 100)
	_, err = conn.Read(buf)
	panicOn(err)
	cli.lastMsg = string(buf)
	close(cli.MsgRecvd)
	cli.MsgRecvd = make(chan bool)

	conn.Close()

}

func (r *BcastClient) IsStopRequested() bool {
	select {
	case <-r.ReqStop:
		return true
	default:
		return false
	}
}

func (r *BcastClient) Stop() {
	if r.IsStopRequested() {
		return
	}
	close(r.ReqStop)
	<-r.Done
}

type BcastServer struct {
	Listen Addr

	Ready   chan bool
	ReqStop chan bool
	Done    chan bool

	lsn     net.Listener
	waiting []net.Conn
}

func NewBcastServer(a Addr) *BcastServer {
	if a.Port == 0 {
		a.Port = GetAvailPort()
	}
	if a.Ip == "" {
		a.Ip = "0.0.0.0"
	}
	a.SetIpPort()

	r := &BcastServer{
		Listen:  a,
		Ready:   make(chan bool),
		ReqStop: make(chan bool),
		Done:    make(chan bool),
	}
	return r
}

func (r *BcastServer) Start() error {

	var err error
	r.lsn, err = net.Listen("tcp", r.Listen.IpPort)
	if err != nil {
		return err
	}
	go func() {
		// Insure proper close down on all exit paths.
		defer func() {
			r.lsn.Close()
			close(r.Done)
		}()

		close(r.Ready)

		// the Accept loop
		for {
			po("client BcastServer::Start(): top of for{} loop.\n")
			if r.IsStopRequested() {
				return
			}

			const serverReadTimeoutMsec = 100
			err := r.lsn.(*net.TCPListener).SetDeadline(time.Now().Add(time.Millisecond * serverReadTimeoutMsec))
			panicOn(err)

			conn, err := r.lsn.Accept()
			if err != nil {
				if r.IsStopRequested() {
					return
				}

				if strings.HasSuffix(err.Error(), "i/o timeout") {
					// okay, ignore
				} else {
					panic(fmt.Sprintf("server BcastServer::Start(): error duing listener.Accept(): '%s'\n", err))
				}
				continue // accept again
			}

			r.waiting = append(r.waiting, conn)

			po("server BcastServer::Start(): accepted '%v' -> '%v' local. len(r.waiting) = %d now.\n", conn.RemoteAddr(), conn.LocalAddr(), len(r.waiting))

		}
	}()
	return nil
}

func (r *BcastServer) Broadcast(msg string) {
	// tell all waiting sockets about msg

	by := []byte(msg)
	for _, conn := range r.waiting {
		n, err := conn.Write(by)
		if n != len(by) {
			panic(fmt.Errorf("could not write everything to conn '%#v'; only %d out of %d", conn, n, len(by)))
		}
		panicOn(err)
	}

}

func (r *BcastServer) IsStopRequested() bool {
	select {
	case <-r.ReqStop:
		return true
	default:
		return false
	}
}

func (r *BcastServer) Stop() {
	if r.IsStopRequested() {
		return
	}
	close(r.ReqStop)
	r.lsn.Close()
	<-r.Done
}
