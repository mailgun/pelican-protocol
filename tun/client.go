package pelicantun

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PelicanSocksProxyConfig struct {
	Listen           Addr
	Dest             Addr
	tickIntervalMsec int
	ChaserCfg        ChaserConfig
}

type PelicanSocksProxy struct {
	Cfg     PelicanSocksProxyConfig
	Ready   chan bool
	reqStop chan bool
	Done    chan bool

	Up            *TcpUpstreamReceiver
	chasers       map[*Chaser]bool
	LastRemoteReq chan net.Addr
	OpenClientReq chan int
	lastRemote    net.Addr
	ChaserDoneCh  chan *Chaser

	testonly_dont_contact_downstream bool

	chaser *Chaser // manages our http connections to the reverse proxy; see alphabeta.go
	mut    sync.Mutex
}

func NewPelicanSocksProxy(cfg PelicanSocksProxyConfig) *PelicanSocksProxy {

	p := &PelicanSocksProxy{
		Cfg:           cfg,
		Ready:         make(chan bool),
		reqStop:       make(chan bool),
		Done:          make(chan bool),
		chasers:       make(map[*Chaser]bool),
		LastRemoteReq: make(chan net.Addr),
		OpenClientReq: make(chan int),
		ChaserDoneCh:  make(chan *Chaser),
	}
	p.SetDefault()
	p.Up = NewTcpUpstreamReceiver(p.Cfg.Listen)

	return p
}

func (f *PelicanSocksProxy) SetDefault() {
	if f.Cfg.Listen.Port == 0 {
		f.Cfg.Listen.Port = GetAvailPort()
	}
	if f.Cfg.Listen.Ip == "" {
		f.Cfg.Listen.Ip = "0.0.0.0"
	}
	f.Cfg.Listen.SetIpPort()

	if f.Cfg.Dest.Port == 0 {
		f.Cfg.Dest.Port = 80
	}
	if f.Cfg.Dest.Ip == "" {
		f.Cfg.Dest.Ip = "127.0.0.1"
	}
	f.Cfg.Dest.SetIpPort()
	if f.Cfg.tickIntervalMsec == 0 {
		f.Cfg.tickIntervalMsec = 250
	}

	chaserDefault := DefaultChaserConfig()
	if f.Cfg.ChaserCfg.ConnectTimeout == 0 {
		f.Cfg.ChaserCfg.ConnectTimeout = chaserDefault.ConnectTimeout
	}
	if f.Cfg.ChaserCfg.TransportTimeout == 0 {
		f.Cfg.ChaserCfg.TransportTimeout = chaserDefault.TransportTimeout
	}
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *PelicanSocksProxy) RequestStop() bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	select {
	case <-s.reqStop:
		return false
	default:
		close(s.reqStop)
		return true
	}
}

func (f *PelicanSocksProxy) Stop() {
	f.RequestStop()
	<-f.Done // 010 is hanging here, waiting for client to finish shutdown.
	WaitUntilServerDown(f.Cfg.Listen.IpPort)
	if f.chaser != nil {
		// chaser can be nil
		f.chaser.Stop()
	}
}

func (f *PelicanSocksProxy) LastRemote() (net.Addr, error) {
	select {
	case lastRemote := <-f.LastRemoteReq:
		return lastRemote, nil
	case <-f.reqStop:
		return nil, fmt.Errorf("PelicanSocksProxy shutting down.")
	case <-f.Done:
		return nil, fmt.Errorf("PelicanSocksProxy shutting down.")
	}
}

// returns nil if the open client count hits target within the maxElap time.
// otherwise a non-nil error is returned. Sleeps in 10 msec increments.
func (f *PelicanSocksProxy) WaitForClientCount(target int, maxElap time.Duration) error {
	trycount := 0
	c := 0
	t0 := time.Now()

	for {
		c = f.CountOfOpenClients()
		trycount++
		if c == target {
			return nil
		} else {
			if time.Since(t0) > maxElap {
				return fmt.Errorf("WaitForClientCount did not hit target %d after %d tries in %v time. Last observed count: %d", target, trycount, maxElap, c)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// CountOfOpenClients returns -1 if PSP is shutting down.
func (f *PelicanSocksProxy) CountOfOpenClients() int {
	select {
	case count := <-f.OpenClientReq:
		return count
	case <-f.reqStop:
		return -1
	case <-f.Done:
		return -1
	}
}

// There is one TcpUpstreamReceiver per port that
// the PelicanSocksProxy listens on. It blocks on the
// socket Accept() call so that the main goroutine of
// the PelicanSocksProxy doesn't have to block.
//
type TcpUpstreamReceiver struct {
	Listen              Addr
	UpstreamTcpConnChan chan net.Conn
	Ready               chan bool
	reqStop             chan bool
	Done                chan bool

	lsn net.Listener
	mut sync.Mutex
}

func NewTcpUpstreamReceiver(a Addr) *TcpUpstreamReceiver {
	a.SetIpPort()
	r := &TcpUpstreamReceiver{
		Listen:              a,
		UpstreamTcpConnChan: make(chan net.Conn),
		Ready:               make(chan bool),
		reqStop:             make(chan bool),
		Done:                make(chan bool),
	}
	return r
}

func (r *TcpUpstreamReceiver) IsStopRequested() bool {
	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *TcpUpstreamReceiver) Start() error {
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

		// the Accept loop
		for {
			po("client TcpUpstreamReceiver::Start(): top of for{} loop.\n")
			if r.IsStopRequested() {
				return
			}

			po("client TcpUpstreamReceiver::Start(): GetTCPConnection listening on '%v'\n", r.Listen.IpPort)

			conn, err := r.lsn.Accept()
			if err != nil {
				if r.IsStopRequested() {
					return
				}

				po("client TcpUpstreamReceiver::Start(): error duing listener.Accept(): '%s'\n", err)
				continue
			}

			po("client TcpUpstreamReceiver::Start(): accepted '%v' -> '%v' local\n", conn.RemoteAddr(), conn.LocalAddr())

			// avoid deadlock on shutdown, use select around the send.
			select {
			case r.UpstreamTcpConnChan <- conn:
				po("client TcpUpstreamReceiver::Start(): sent on r.UpstreamTcpConnChan\n")
			case <-r.reqStop:
				po("client TcpUpstreamReceiver::Start(): r.reqStop received.\n")
				return
			}
		}
	}()
	return nil
}

func (r *TcpUpstreamReceiver) Stop() {
	r.RequestStop()
	r.lsn.Close()
	<-r.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *TcpUpstreamReceiver) RequestStop() bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	select {
	case <-s.reqStop:
		return false
	default:
		close(s.reqStop)
		return true
	}
}

func (f *PelicanSocksProxy) ConnectDownstreamHttp() (string, error) {

	// initiate new session and read key
	url := fmt.Sprintf("http://%s/create", f.Cfg.Dest.IpPort)
	po("client/PSP: ConnectDownstreamHttp: attempting POST to '%s'\n", url)
	resp, err := http.Post(
		url,
		"text/plain",
		&bytes.Buffer{})

	defer func() {
		if resp != nil && resp.Body != nil {

			// next two statements allow re-use of connection:
			// https://groups.google.com/forum/#!topic/golang-nuts/ehZdZ7Wmr-c
			// bradfitz, 8/9/13: "you have to read the entire res.Body for
			// it to re-use the connection."
			ioutil.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}()

	if err != nil {
		return "", fmt.Errorf("ConnectDownstreamHttp: error during Post to '%s': '%s'", url, err)
	}

	key, err2 := ioutil.ReadAll(resp.Body)
	panicOn(err2)

	po("Post got back: resp.Status = '%s' resp.StatusCode = %d\n", resp.Status, resp.StatusCode)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ConnectDownstreamHttp: error during Post to '%s': '%s'; body: '%s'", url, resp.Status, string(key))
	}

	log.Printf("client/PSP: ConnectDownstreamHttp: after Post('/create') we got ResponseWriter with key = '%x'... (string: '%s'...) of len %d\n\n", key[:5], string(key[:5]), len(key))

	return string(key), nil
}

func (f *PelicanSocksProxy) Start() error {
	err := f.Up.Start()
	if err != nil {
		panic(fmt.Errorf("could not start f.Up, our TcpUpstreamReceiver; error: '%s'", err))
	}

	buf := new(bytes.Buffer)
	buf.Reset()

	//sendCount := 0

	var upConn net.Conn
	go func() {
		var topLoopCount int64
		for {
			topLoopCount++
			select {
			case f.OpenClientReq <- len(f.chasers):
				// nothing more, just send when requested

			case f.LastRemoteReq <- f.lastRemote:
				// nothing more, just send when requested

			case upConn = <-f.Up.UpstreamTcpConnChan:
				po("client/PSP: Start(): handling upConn = <-f.Up.UpstreamTcpConnChan.  len(f.chasers) == openClientReq = %d\n", len(f.chasers))
				f.lastRemote = upConn.RemoteAddr()

				var err error
				var key string
				if f.testonly_dont_contact_downstream {
					po("client/PSP: Start(): handling upConn = <-f.Up.UpstreamTcpConnChan: " +
						"f.testonly_dont_contact_downstream is true.\n")
					key = "testkey" + strings.Repeat("0", KeyLen-len("testkey"))
				} else {
					key, err = f.ConnectDownstreamHttp()
					if err != nil {
						log.Printf("client/PSP: Start() loop: got connection upstream from upConn.RemoteAddr('%s'), but could not connect downstream: '%s'\n", upConn.RemoteAddr(), err)
						fmt.Fprintf(upConn, "PSP/PelicanSocksProxy error: could not connect to downstream PelicanReverseProxy server at address '%s': error was '%s'", f.Cfg.Dest.IpPort, err)
						upConn.Close()

						continue
					}
					if key == "" {
						panic("internal error in error handling logic: empty key back but no error from f.ConnectDownstreamHttp()")
					}
				}

				VPrintf("in case upConn = <-f.Up.UpstreamTcpConnChan,  %v -> %v\n", upConn.RemoteAddr(), upConn.LocalAddr())

				chaser := NewChaser(f.Cfg.ChaserCfg, upConn, bufSize, key, f.ChaserDoneCh, f.Cfg.Dest)
				chaser.Start()
				f.chasers[chaser] = true
				po("after add new chaser %p, len(chasers) = %d\n", chaser, len(f.chasers))
				i := 0
				for ch := range f.chasers {
					po("%d-th chaser is %p", i, ch)
					i++
				}
			case thisChaserIsDone := <-f.ChaserDoneCh:
				po("thisChaserIsDone %p received on channel, len(chasers) = %d", thisChaserIsDone, len(f.chasers))
				if !f.chasers[thisChaserIsDone] {
					panic(fmt.Sprintf("thisChaserIsDone %p not found in f.chasers = '%#v'", thisChaserIsDone, f.chasers))
				}
				delete(f.chasers, thisChaserIsDone)
				po("chaser stopped and deleted: %p. after delete, len(chasers) = %d", thisChaserIsDone, len(f.chasers))

				//f.redoAlarm()

			case <-f.reqStop:
				po("client: in <-f.reqStop, len(chasers) = %d\n", len(f.chasers))
				f.Up.Stop()

				// the reader.Stop() will call back in on f.ChaserDoneCh
				// to report finishing. Therefore we use StopWithoutReporting()
				// here to avoid a deadlock situation.
				for reader, _ := range f.chasers {
					reader.StopWithoutReporting()
					delete(f.chasers, reader)
				}

				close(f.Done)
				return
			}
		}
	}()

	po("\n end of PelicanSocksProxy::ConnectDownstreamHttp(): about to call WaitUntilServerUp(f.Cfg.Listen.IpPort) with Listen = %s\n", f.Cfg.Listen.IpPort)
	WaitUntilServerUp(f.Cfg.Listen.IpPort)
	po("\n end of PelicanSocksProxy::ConnectDownstreamHttp(): after call to WaitUntilServerUp(f.Cfg.Listen.IpPort) with Listen = %s    len(f.chasers)=%d\n", f.Cfg.Listen.IpPort, len(f.chasers))
	return nil
}
