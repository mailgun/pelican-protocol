package pelicantun

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// simple echo server for testing
//
// ============================================

type EchoServer struct {
	Listen Addr

	Ready   chan bool
	reqStop chan bool
	Done    chan bool

	lsn net.Listener

	FirstClient chan bool
	mut         sync.Mutex
}

func NewEchoServer(a Addr) *EchoServer {
	if a.Port == 0 {
		a.Port = GetAvailPort()
	}
	if a.Ip == "" {
		a.Ip = "0.0.0.0"
	}
	a.SetIpPort()

	r := &EchoServer{
		Listen:      a,
		Ready:       make(chan bool),
		reqStop:     make(chan bool),
		Done:        make(chan bool),
		FirstClient: make(chan bool),
	}
	return r
}

func (r *EchoServer) Start() error {

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
			//po("EchoServer::Start(): top of for{} loop.\n")
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
					panic(fmt.Sprintf("server EchoServer::Start(): error duing listener.Accept(): '%s'\n", err))
				}
				continue // accept again
			}

			// close FirstClient only once
			select {
			case <-r.FirstClient:
			default:
				close(r.FirstClient)
			}

			//po("server EchoServer::Start(): accepted '%v' -> '%v' local.\n", conn.RemoteAddr(), conn.LocalAddr())

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
					//po("echo service routine read buf '%s'\n", string(buf[:n]))

					err = netconn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
					panicOn(err)

					netconn.Write(buf[:n])
					//nw, _ := netconn.Write(buf[:n])
					//po("echo service routine wrote buf '%s'\n", string(buf[:nw]))
				}
			}(conn)

		}

	}()
	<-r.Ready
	WaitUntilServerUp(r.Listen.IpPort)
	return nil
}

func (r *EchoServer) Nonecho(msg string) {
	// tell next to arrive socket msg instead of echoing what they gave us.

	po("\n\n  EchoServer::Nonecho() called with msg = '%s'\n\n", msg)

}

func (r *EchoServer) IsStopRequested() bool {
	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *EchoServer) Stop() {
	r.RequestStop()
	r.lsn.Close()
	<-r.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *EchoServer) RequestStop() bool {
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
