package pelicantun

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

const bufSize = 1024

type ConnReader struct {
	reader io.Reader
	readCh chan []byte
	bufsz  int

	Ready   chan bool
	ReqStop chan bool
	Done    chan bool
	key     string
}

func NewConnReader(r io.Reader, bufsz int, key string) *ConnReader {
	re := &ConnReader{
		reader:  r,
		readCh:  make(chan []byte, bufsz),
		bufsz:   bufsz,
		Ready:   make(chan bool),
		ReqStop: make(chan bool),
		Done:    make(chan bool),
		key:     key,
	}
	return re
}

func (r *ConnReader) IsStopRequested() bool {
	select {
	case <-r.ReqStop:
		return true
	default:
		return false
	}
}

func (r *ConnReader) Stop() {
	if r.IsStopRequested() {
		return
	}
	close(r.ReqStop)
	<-r.Done
}

func (r *ConnReader) Start() {
	go func() {
		close(r.Ready)
		for {
			b := make([]byte, r.bufsz)
			n, err := r.reader.Read(b)
			if err != nil {
				if !r.IsStopRequested() {
					close(r.ReqStop)
				}
				close(r.Done)
				return
			}

			select {
			case r.readCh <- b[0:n]:
			case <-r.ReqStop:
				close(r.Done)
				return
			}

			if r.IsStopRequested() {
				close(r.Done)
				return
			}
		}
	}()
}

type PelicanSocksProxyConfig struct {
	Listen           addr
	Dest             addr
	tickIntervalMsec int
}

type PelicanSocksProxy struct {
	Cfg     PelicanSocksProxyConfig
	Ready   chan bool
	ReqStop chan bool
	Done    chan bool

	Up            *TcpUpstreamReceiver
	readers       []*ConnReader
	LastRemoteReq chan net.Addr
	lastRemote    net.Addr
}

func NewPelicanSocksProxy(cfg PelicanSocksProxyConfig) *PelicanSocksProxy {

	p := &PelicanSocksProxy{
		Cfg:           cfg,
		Ready:         make(chan bool),
		ReqStop:       make(chan bool),
		Done:          make(chan bool),
		readers:       make([]*ConnReader, 0),
		LastRemoteReq: make(chan net.Addr),
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
		f.Cfg.Listen.Ip = "127.0.0.1"
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

func (f *PelicanSocksProxy) LastRemote() (net.Addr, error) {
	select {
	case lastRemote := <-f.LastRemoteReq:
		return lastRemote, nil
	case <-f.Done:
		return nil, fmt.Errorf("PelicanSocksProxy shutting down.")
	}
}

func (f *PelicanSocksProxy) OpenClientCount() int {
	return 0
}

// There is one TcpUpstreamReceiver per port that
// the PelicanSocksProxy listens on. It blocks on the
// socket Accept() call so that the main goroutine of
// the PSP doesn't have to block.
//
type TcpUpstreamReceiver struct {
	Listen              addr
	UpstreamTcpConnChan chan net.Conn
	Ready               chan bool
	ReqStop             chan bool
	Done                chan bool

	lsn net.Listener
}

func NewTcpUpstreamReceiver(a addr) *TcpUpstreamReceiver {
	r := &TcpUpstreamReceiver{
		Listen:              a,
		UpstreamTcpConnChan: make(chan net.Conn),
		Ready:               make(chan bool),
		ReqStop:             make(chan bool),
		Done:                make(chan bool),
	}
	a.SetIpPort()
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
		for {
			po("client TcpUpstreamReceiver::Start(): top of for{} loop.\n")
			if r.IsStopRequested() {
				r.lsn.Close()
				close(r.Done)
				return
			}

			po("client TcpUpstreamReceiver::Start(): GetTCPConnection listening on '%v'\n", r.Listen.IpPort)

			conn, err := r.lsn.Accept()
			if err != nil {
				if r.IsStopRequested() {
					r.lsn.Close()
					close(r.Done)
					return
				}

				log.Printf("client TcpUpstreamReceiver::Start(): error duing listener.Accept(): '%s'\n", err)
				continue
			}

			log.Printf("client TcpUpstreamReceiver::Start(): accepted '%v' -> '%v' local\n", conn.RemoteAddr(), conn.LocalAddr())

			// avoid deadlock on shutdown, use select around the send.
			select {
			case r.UpstreamTcpConnChan <- conn:
			case <-r.ReqStop:
				r.lsn.Close()
				close(r.Done)
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

	if err != nil {
		return "", fmt.Errorf("ConnectDownstreamHttp: error during Post to '%s': '%s'", url, err)
	}
	key, err := ioutil.ReadAll(resp.Body)
	panicOn(err)
	resp.Body.Close()
	log.Printf("client/PSP: ConnectDownstreamHttp: after Post('/create') we got ResponseWriter with key = '%x' of len %d\n\n", key, len(key))

	return string(key), nil
}

func (f *PelicanSocksProxy) Start() error {
	err := f.Up.Start()
	if err != nil {
		panic(fmt.Errorf("could not start f.Up, our TcpUpstreamReceiver; error: '%s'", err))
	}

	// will this bother our server to get a hang up right away? it shouldn't.
	WaitUntilServerUp(f.Cfg.Listen.IpPort)

	// ticker to set a rate at which to hit the server
	//tick := time.NewTicker(time.Duration(int64(f.Cfg.tickIntervalMsec)) * time.Millisecond)

	buf := new(bytes.Buffer)
	buf.Reset()

	//sendCount := 0

	var upConn net.Conn
	go func() {
		for {
			select {
			case f.LastRemoteReq <- f.lastRemote:
				// nothing more, just send when requested

			case upConn = <-f.Up.UpstreamTcpConnChan:
				f.lastRemote = upConn.RemoteAddr()

				key, err := f.ConnectDownstreamHttp()
				if err != nil {
					log.Printf("client/PSP: Start() loop: got connection upstream from upConn.RemoteAddr('%s'), but could not connect downstream: '%s'\n", upConn.RemoteAddr(), err)
					upConn.Close()
					continue
				}
				if key == "" {
					panic("internal error in error handling logic: empty key back but no error from f.ConnectDownstreamHttp()")
				}

				connReader := NewConnReader(upConn, bufSize, key)
				connReader.Start()
				f.readers = append(f.readers, connReader)

			case <-f.ReqStop:
				f.Up.Stop()
				for _, reader := range f.readers {
					reader.Stop()
				}
				close(f.Done)
				return
				/*
							case b := <-read:
								// fill buf here
								po("client: <-read of '%s'; hex:'%x' of length %d added to buffer\n", string(b), b, len(b))
								buf.Write(b)
								po("client: after write to buf of len(b)=%d, buf is now length %d\n", len(b), buf.Len())
					case <-tick.C:
						sendCount++
						po("\n ====================\n client sendCount = %d\n ====================\n", sendCount)
						po("client: sendCount %d, got tick.C. key as always(?) = '%x'. buf is now size %d\n", sendCount, key, buf.Len())
						// write buf to new http request, starting with key
						req := bytes.NewBuffer(key)
						buf.WriteTo(req)
						resp, err := http.Post(
							"http://"+f.Cfg.Dest.IpPort+"/ping",
							"application/octet-stream",
							req)
						if err != nil && err != io.EOF {
							log.Println(err.Error())
							continue
						}

						// write http response response to conn

						// we take apart the io.Copy to print out the response for debugging.
						//_, err = io.Copy(conn, resp.Body)

						body, err := ioutil.ReadAll(resp.Body)
						panicOn(err)
						po("client: resp.Body = '%s'\n", string(body))
						_, err = conn.Write(body)
						panicOn(err)
						resp.Body.Close()
				*/

			}
		}
	}()
	return nil
}
