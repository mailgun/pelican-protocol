package pelicantun

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// simple client and server to test direct tcp connections through the proxies.

// ============================================

// BcastClient long polls for a single message.
//
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
		po("\n after cli.Start() got to Write to conn.\n")

		buf := make([]byte, 100)

		// read loop, check for done request
		isTimeout := false
		for {
			isTimeout = false
			err = conn.SetDeadline(time.Now().Add(time.Millisecond * 100))
			panicOn(err)
			n, err := conn.Read(buf)
			if err != nil {
				if strings.HasSuffix(err.Error(), "i/o timeout") {
					// okay, ignore
					isTimeout = true
				} else {
					panic(err)
				}
			}
			if !isTimeout {
				cli.lastMsg = string(buf[:n])
				close(cli.MsgRecvd)
				po("\n after cli.Start() got to Read from conn. n = %d bytes\n", n)
			}

			// check for stop requests
			select {
			case <-cli.ReqStop:
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
	ReqStop chan bool
	Done    chan bool

	lsn     net.Listener
	waiting []net.Conn

	FirstClient chan bool
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
		Listen:      a,
		Ready:       make(chan bool),
		ReqStop:     make(chan bool),
		Done:        make(chan bool),
		FirstClient: make(chan bool),
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

			// close FirstClient only once
			select {
			case <-r.FirstClient:
			default:
				close(r.FirstClient)
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

					netconn.Read(buf)
					//n, _ := netconn.Read(buf)
					//po("reader service routine read buf '%s'\n", string(buf[:n]))
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
