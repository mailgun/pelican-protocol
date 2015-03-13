package pelicantun

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

type PelicanSocksProxyConfig struct {
	Listen           Addr
	Dest             Addr
	tickIntervalMsec int
}

// for requesting a doneAlarm and preventing races
// during testing
type ReaderDoneAlarmTicket struct {
	// the chan bool received on Reply can be waited on to know
	// when the reader-done alarm fires. It is the most
	// recent f.doneAlarm from the Start() goroutine.
	Reply chan chan bool

	// don't read TopLoopCount until Reply is received. Race otherwise.
	TopLoopCount int64
}

func NewReaderDoneAlarmTicket() *ReaderDoneAlarmTicket {
	return &ReaderDoneAlarmTicket{
		Reply: make(chan chan bool),
	}
}

type PelicanSocksProxy struct {
	Cfg     PelicanSocksProxyConfig
	Ready   chan bool
	ReqStop chan bool
	Done    chan bool

	Up                   *TcpUpstreamReceiver
	readers              map[*ConnReader]bool
	LastRemoteReq        chan net.Addr
	OpenClientReq        chan int
	lastRemote           net.Addr
	ConnReaderDoneCh     chan *ConnReader
	ReaderDoneAlarmReqCh chan *ReaderDoneAlarmTicket

	testonly_dont_contact_downstream bool
	doneAlarm                        chan bool
}

func NewPelicanSocksProxy(cfg PelicanSocksProxyConfig) *PelicanSocksProxy {

	p := &PelicanSocksProxy{
		Cfg:                  cfg,
		Ready:                make(chan bool),
		ReqStop:              make(chan bool),
		Done:                 make(chan bool),
		readers:              make(map[*ConnReader]bool),
		LastRemoteReq:        make(chan net.Addr),
		OpenClientReq:        make(chan int),
		ConnReaderDoneCh:     make(chan *ConnReader),
		doneAlarm:            nil, // nil on purpose to start with
		ReaderDoneAlarmReqCh: make(chan *ReaderDoneAlarmTicket),
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
}

func (f *PelicanSocksProxy) Stop() {
	close(f.ReqStop)
	<-f.Done
	WaitUntilServerDown(f.Cfg.Listen.IpPort)
}

func (f *PelicanSocksProxy) GetDoneReaderIndicator() chan bool {
	ticket := NewReaderDoneAlarmTicket()

	select {
	case f.ReaderDoneAlarmReqCh <- ticket:
	case <-f.ReqStop:
		return nil //, fmt.Errorf("PelicanSocksProxy shutting down.")
	case <-f.Done:
		return nil //, fmt.Errorf("PelicanSocksProxy shutting down.")
	}

	// sent, so we are sure to get a reply, based on the implementation/
	// handling of f.ReaderDoneAlarmReqCh in Start()
	doneAlarm := <-ticket.Reply

	return doneAlarm
}

func (f *PelicanSocksProxy) GetTopLoopCount() int64 {
	ticket := NewReaderDoneAlarmTicket()

	select {
	case f.ReaderDoneAlarmReqCh <- ticket:
	case <-f.ReqStop:
		return -1 //, fmt.Errorf("PelicanSocksProxy shutting down.")
	case <-f.Done:
		return -1 //, fmt.Errorf("PelicanSocksProxy shutting down.")
	}

	// sent, so we are sure to get a reply, based on the implementation/
	// handling of f.ReaderDoneAlarmReqCh in Start()
	<-ticket.Reply
	return ticket.TopLoopCount
}

func (f *PelicanSocksProxy) LastRemote() (net.Addr, error) {
	select {
	case lastRemote := <-f.LastRemoteReq:
		return lastRemote, nil
	case <-f.ReqStop:
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
		c = f.OpenClientCount()
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

// OpenClientCount returns -1 if PSP is shutting down.
func (f *PelicanSocksProxy) OpenClientCount() int {
	select {
	case count := <-f.OpenClientReq:
		return count
	case <-f.ReqStop:
		return -1
	case <-f.Done:
		return -1
	}
}

// There is one TcpUpstreamReceiver per port that
// the PelicanSocksProxy listens on. It blocks on the
// socket Accept() call so that the main goroutine of
// the PSP doesn't have to block.
//
type TcpUpstreamReceiver struct {
	Listen              Addr
	UpstreamTcpConnChan chan net.Conn
	Ready               chan bool
	ReqStop             chan bool
	Done                chan bool

	lsn net.Listener
}

func NewTcpUpstreamReceiver(a Addr) *TcpUpstreamReceiver {
	a.SetIpPort()
	r := &TcpUpstreamReceiver{
		Listen:              a,
		UpstreamTcpConnChan: make(chan net.Conn),
		Ready:               make(chan bool),
		ReqStop:             make(chan bool),
		Done:                make(chan bool),
	}
	return r
}

func (r *TcpUpstreamReceiver) IsStopRequested() bool {
	select {
	case <-r.ReqStop:
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
			case <-r.ReqStop:
				po("client TcpUpstreamReceiver::Start(): r.ReqStop received.\n")
				return
			}
		}
	}()
	return nil
}

func (r *TcpUpstreamReceiver) Stop() {
	if r.IsStopRequested() {
		return
	}
	close(r.ReqStop)
	r.lsn.Close()
	<-r.Done
}

func (f *PelicanSocksProxy) ConnectDownstreamHttp() (string, error) {

	// initiate new session and read key
	url := fmt.Sprintf("http://%s/create", f.Cfg.Dest.IpPort)
	log.Printf("client/PSP: ConnectDownstreamHttp: attempting POST to '%s'\n", url)
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

func (f *PelicanSocksProxy) redoAlarm() {
	// allow tests to wait until a doneReader has been received, to avoid racing.
	if f.doneAlarm != nil {
		//po("closing active done alarm\n")
		close(f.doneAlarm)
		f.doneAlarm = nil
	}
}

func (f *PelicanSocksProxy) Start() error {
	err := f.Up.Start()
	if err != nil {
		panic(fmt.Errorf("could not start f.Up, our TcpUpstreamReceiver; error: '%s'", err))
	}

	// will this bother our server to get a hang up right away? no, but it means
	// our open and close handling logic will get exercised immediately.
	// Also, very important to do this to prevent races on startup. Test correctness
	// will non-deterministically be impacted if we don't wait with WaitUntilServerUp()
	// here.
	f.doneAlarm = make(chan bool)
	WaitUntilServerUp(f.Cfg.Listen.IpPort)

	// ticker to set a rate at which to hit the server
	//tick := time.NewTicker(time.Duration(int64(f.Cfg.tickIntervalMsec)) * time.Millisecond)

	buf := new(bytes.Buffer)
	buf.Reset()

	//sendCount := 0

	var upConn net.Conn
	go func() {
		var topLoopCount int64
		for {
			topLoopCount++
			select {
			case f.OpenClientReq <- len(f.readers):
				// nothing more, just send when requested

			case f.LastRemoteReq <- f.lastRemote:
				// nothing more, just send when requested

			case upConn = <-f.Up.UpstreamTcpConnChan:
				po("client/PSP: Start(): handling upConn = <-f.Up.UpstreamTcpConnChan.\n")
				f.lastRemote = upConn.RemoteAddr()

				var err error
				var key string
				if f.testonly_dont_contact_downstream {
					po("client/PSP: Start(): handling upConn = <-f.Up.UpstreamTcpConnChan: " +
						"f.testonly_dont_contact_downstream is true.\n")
					key = "testkey"
				} else {
					key, err = f.ConnectDownstreamHttp()
					if err != nil {
						log.Printf("client/PSP: Start() loop: got connection upstream from upConn.RemoteAddr('%s'), but could not connect downstream: '%s'\n", upConn.RemoteAddr(), err)
						fmt.Fprintf(upConn, "PSP/PelicanSocksProxy error: could not connect to downstream PelicanReverseProxy server at address '%s': error was '%s'", f.Cfg.Dest.IpPort, err)
						upConn.Close()

						// don't block startup just because we failed here.
						f.redoAlarm()
						continue
					}
					if key == "" {
						panic("internal error in error handling logic: empty key back but no error from f.ConnectDownstreamHttp()")
					}
				}

				connReader := NewConnReader(upConn, bufSize, key, f.ConnReaderDoneCh, f.Cfg.Dest)
				connReader.Start()
				f.readers[connReader] = true
				//po("after add, len(readers) = %d\n", len(f.readers))

			case alarmReq := <-f.ReaderDoneAlarmReqCh:
				//po("got request for done alarm\n")

				if f.doneAlarm == nil {
					f.doneAlarm = make(chan bool)
				}
				alarmReq.TopLoopCount = topLoopCount
				alarmReq.Reply <- f.doneAlarm
				//po("replied with done alarm\n")

			case doneReader := <-f.ConnReaderDoneCh:
				//po("doneReader received on channel, len(readers) = %d\n", len(f.readers))
				if !f.readers[doneReader] {
					panic(fmt.Sprintf("doneReader %p not found in f.readers = '%#v'", doneReader, f.readers))
				}
				delete(f.readers, doneReader)
				//po("after delete, len(readers) = %d\n", len(f.readers))

				f.redoAlarm()

			case <-f.ReqStop:
				po("client: in <-f.ReqStop, len(readers) = %d\n", len(f.readers))
				f.Up.Stop()

				// the reader.Stop() will call back in on f.ConnReaderDoneCh
				// to report finishing. Therefore we use StopWithoutReporting()
				// here to avoid a deadlock situation.
				for reader, _ := range f.readers {
					reader.StopWithoutReporting()
					delete(f.readers, reader)
				}

				close(f.Done)
				return
			}
		}
	}()

	// wait until the WaitUntilServerUp(f.Cfg.Listen.IpPort) completes processing.
	// the go-routine above has to start and get the close message before doneAlarm
	// will be closed in turn.
	<-f.doneAlarm

	return nil
}
