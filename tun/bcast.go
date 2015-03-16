package pelicantun

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// simple client and server to test direct tcp connections through the proxies.

// ============================================

// BcastClient long polls for a single message.
//
type BcastClient struct {
	Dest Addr

	Ready   chan bool
	reqStop chan bool
	Done    chan bool

	MsgRecvd chan bool
	lastMsg  string

	mut sync.Mutex
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
		reqStop:  make(chan bool),
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

		msg := fmt.Sprintf("hello from bcast_client to '%s'", conn.RemoteAddr())
		_, err = conn.Write([]byte(msg))
		panicOn(err)
		po("\n \n bcast_client:  after cli.Start() got to Write '%s' to conn.\n", msg)

		buf := make([]byte, 100)

		// read loop, check for done request
		isTimeout := false
		for {
			isTimeout = false
			err = conn.SetDeadline(time.Now().Add(time.Millisecond * 100))
			panicOn(err)

			// Read
			n, err := conn.Read(buf)
			if err != nil {
				if strings.HasSuffix(err.Error(), "i/o timeout") {
					// okay, ignore
					isTimeout = true
				} else {
					panic(err)
				}
			}
			po("\n bcast_client: after Read, isTimeout: %v, err: %v\n", isTimeout, err)

			if !isTimeout {
				cli.lastMsg = string(buf[:n])
				close(cli.MsgRecvd)
				po("\n bcast_client: message received!!! after cli.Start() got to Read '%s' from conn. n = %d bytes\n", cli.lastMsg, n)
			}

			// check for stop requests
			select {
			case <-cli.reqStop:
				po("\n bcast_client: shutting down.\n")
				conn.Close()
				close(cli.Done)
				return
			default:
				// loop
			}
		} // end for
	}()
}

func (r *BcastClient) IsStopRequested() bool {
	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *BcastClient) Stop() {
	r.RequestStop()
	<-r.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *BcastClient) RequestStop() bool {
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

func (cli *BcastClient) Expect(msg string) bool {
	tries := 40
	sleep := time.Millisecond * 40
	found := false
	for i := 0; i < tries; i++ {
		last := cli.LastMsgReceived()
		if last == msg {
			found = true
			break
		} else {
			po("\n expect rejecting msg: '%s'\n", last)
		}
		time.Sleep(sleep)
	}
	if !found {
		panic(fmt.Errorf("could not find expected LastMsgReceived() == '%s' in %d tries of %v each", msg, tries, sleep))
	}
	return found
}

// ============================================

// BcastServer accumulates clients and then on queue (Bcast() call)
//  sends the same message to all waiting clients. This
//  simulates long polling.
//
type BcastServer struct {
	Listen Addr

	Ready   chan bool
	reqStop chan bool
	Done    chan bool

	lsn     net.Listener
	waiting []net.Conn

	FirstClient  chan bool
	SecondClient chan bool

	mut sync.Mutex
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
		Listen:       a,
		Ready:        make(chan bool),
		reqStop:      make(chan bool),
		Done:         make(chan bool),
		FirstClient:  make(chan bool),
		SecondClient: make(chan bool),
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
			//po("BcastServer::Start(): top of for{} loop.\n")
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

			// close FirstClient only once: the WaitUntilServerIsUp confirmation client.
			if len(r.waiting) == 1 {
				select {
				case <-r.FirstClient:
				default:
					close(r.FirstClient)
				}
			}

			// close SecondClient only once: the actual test client.
			if len(r.waiting) == 2 {
				select {
				case <-r.SecondClient:
				default:
					close(r.SecondClient)
				}
			}

			po("server BcastServer::Start(): accepted '%v' -> '%v' local. len(r.waiting) = %d now.\n", conn.RemoteAddr(), conn.LocalAddr(), len(r.waiting))

			// read from the connections to service clients
			go func(netconn net.Conn) {
				buf := make([]byte, 100)
				for {
					if r.IsStopRequested() {
						return
					}

					err = netconn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
					panicOn(err)

					n, _ := netconn.Read(buf)
					if n > 0 {
						po("bcast_server: reader service routine read buf '%s'\n", string(buf[:n]))
					}
				}
			}(conn)

			//			select {
			//			case <-time.After(time.Millisecond * serverReadTimeoutMsec):
			//			}

		}

	}()
	return nil
}

func (r *BcastServer) Bcast(msg string) {
	// tell all waiting sockets about msg

	po("\n\n  BcastServer::Bcast() called with msg = '%s'\n\n", msg)

	by := []byte(msg)
	i := 0
	for _, conn := range r.waiting {
		po("\n\n  BcastServer::Bcast() sending to conn %d = '%s'\n\n", i, conn.RemoteAddr())
		i++
		n, err := conn.Write(by)
		if n != len(by) {
			panic(fmt.Errorf("could not write everything to conn '%#v'; only %d out of %d", conn, n, len(by)))
		}
		panicOn(err)
	}

}

func (r *BcastServer) IsStopRequested() bool {
	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *BcastServer) Stop() {
	r.RequestStop()
	r.lsn.Close()
	<-r.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *BcastServer) RequestStop() bool {
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
