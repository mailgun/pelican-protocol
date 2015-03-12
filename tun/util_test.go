package pelicantun

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
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
	cli := NewBcastClient(Addr{Port: fwd.Cfg.Listen.Port})
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

func NewBcastClient(dest Addr) *BcastClient {

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

func (cli *BcastClient) WaitForMsg() string {
	<-cli.MsgRecvd
	return cli.lastMsg
}

func (cli *BcastClient) Start() {

	go func() {
		close(cli.Ready)
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
		//cli.MsgRecvd = make(chan bool)

		conn.Close()
		close(cli.Done)
	}()
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

			select {
			case <-time.After(time.Millisecond * serverReadTimeoutMsec):
			}

		}

	}()
	return nil
}

func (r *BcastServer) Bcast(msg string) {
	// tell all waiting sockets about msg

	po("\n\n  BcastServer::Bcast() called with msg = '%s'\n\n", msg)

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

//////////// test our simple client and server can talk

func TestTcpClientAndServerCanTalkDirectly012(t *testing.T) {

	// start broadcast server
	srv := NewBcastServer(Addr{})
	srv.Start()
	po("\n done with srv.Start()\n")

	// start broadcast client
	cli := NewBcastClient(Addr{Port: srv.Listen.Port})
	cli.Start()

	po("\n done with cli.Start()\n")

	cv.Convey("The broadcast client and server should be able to speak directly without proxies, sending and receiving", t, func() {

		msg := "BREAKING NEWS"
		po("\n about to srv.Bcast()\n")
		srv.Bcast(msg)

		found := cli.Expect(msg)

		cv.So(found, cv.ShouldEqual, true)
	})

	fmt.Printf("Given a Forward and Reverse proxy, in order to avoid creating new sockets too often (expensive), we should re-use the existing sockets for up to 5 round trips in 30 seconds.")
}

func (cli *BcastClient) Expect(msg string) bool {
	tries := 40
	sleep := time.Millisecond * 40
	found := false
	for i := 0; i < tries; i++ {
		if cli.LastMsgReceived() == msg {
			found = true
			break
		}
		time.Sleep(sleep)
	}
	if !found {
		panic(fmt.Errorf("could not find expected LastMsgReceived() == '%s' in %d tries of %v each", msg, tries, sleep))
	}
	return found
}
