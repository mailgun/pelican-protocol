package pelicantun

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// simple counting server for testing
//
// ============================================

type CountingTestServer struct {
	Listen Addr

	Ready   chan bool
	reqStop chan bool
	Done    chan bool

	lsn net.Listener

	FirstClient             chan bool
	FirstCountingClientSeen chan bool
	mut                     sync.Mutex

	lastCountRecv int
	lastCountSent int

	tmLastSend []time.Time
	tmLastRecv []time.Time
}

func NewCountingTestServer(a Addr) *CountingTestServer {
	if a.Port == 0 {
		a.Port = GetAvailPort()
	}
	if a.Ip == "" {
		a.Ip = "0.0.0.0"
	}
	a.SetIpPort()

	r := &CountingTestServer{
		Listen:                  a,
		Ready:                   make(chan bool),
		reqStop:                 make(chan bool),
		Done:                    make(chan bool),
		FirstClient:             make(chan bool),
		FirstCountingClientSeen: make(chan bool),
		tmLastSend:              make([]time.Time, 0),
		tmLastRecv:              make([]time.Time, 0),
	}
	return r
}

// format of the expected client messages
var countClientRegex = regexp.MustCompile(`^from CountingTestClient: client_packet_number=(\d+)`)
var countServerRegex = regexp.MustCompile(`^from CountingTestServer: server_packet_number=(\d+)`)

func (r *CountingTestServer) Start() error {

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
			po("CountingTestServer::Start(): top of for{} loop.\n")
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
					panic(fmt.Sprintf("server CountingTestServer::Start(): error duing listener.Accept(): '%s'\n", err))
				}
				continue // accept again
			}

			// close FirstClient only once
			select {
			case <-r.FirstClient:
			default:
				close(r.FirstClient)
			}

			po("server CountingTestServer::Start(): accepted '%v' -> '%v' local.\n", conn.RemoteAddr(), conn.LocalAddr())

			// read from the connections to service clients

			go func(netconn net.Conn) {
				defer func() {
					po("done with CountingTestServer netconn handling goroutine for %v -> %v", netconn.RemoteAddr(), netconn.LocalAddr())
				}()

				buf := make([]byte, 100)
				for {
					if r.IsStopRequested() {
						return
					}

					err = netconn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
					panicOn(err)

					n, err := netconn.Read(buf)
					if err != nil && err.Error() != "i/o timeout" {
						continue
					}
					if err != nil && err.Error() != "EOF" {
						panic(err)
					}
					if n > 0 {
						sbuf := string(buf[:n])
						po("CountingTestServer: reader service routine read buf '%s'\n", sbuf)

						const first_cli_msg = "from CountingTestClient: client_packet_number"
						if n > len(first_cli_msg) && string(buf[:len(first_cli_msg)]) == first_cli_msg {

							po("CountingTestServer: I see a client with proper preamble!")

							r.NoteTmRecv()

							select {
							case <-r.FirstCountingClientSeen:
								po("CountingTestServer: already past first close of r.FirstHelloClient")
							default:
								close(r.FirstCountingClientSeen)
								po("CountingTestServer: closed r.FirstCountingClientSeen")
							}

							match := countClientRegex.FindStringSubmatch(sbuf)
							po("CountingTestServer: countClientRegex match = %#v\n", match)
							if match == nil || len(match) != 2 {
								po("CountingTestServer: bad client message/no match to regex!: '%s", sbuf)
								continue
							}
							pktCliCount := match[1]

							cliCount, err := strconv.Atoi(pktCliCount)
							panicOn(err)

							if r.lastCountRecv > 0 {
								// check the sequence
								if cliCount != r.lastCountRecv+2 {
									panic(fmt.Sprintf("CountingTestServer: cliCount out of order! cliCount = %d, but r.lastCountSeen = %d, should hit the odd numbers, each in turn!", cliCount, r.lastCountRecv))
								}
							}

							serverMsg := fmt.Sprintf("from CountingTestServer: server_packet_number=%d", cliCount+1)
							r.lastCountRecv = cliCount

							// write back to these clients

							err = netconn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
							panicOn(err)

							nw, _ := netconn.Write([]byte(serverMsg))
							if nw > 0 {
								r.NoteTmSent()

								po("CountingTestServer saw %d, and wrote reply '%s'\n", cliCount, serverMsg)
								r.lastCountSent = cliCount + 1
							}

						}

					}

				}
			}(conn)

		}

	}()
	<-r.Ready
	WaitUntilServerUp(r.Listen.IpPort)
	return nil
}

func (r *CountingTestServer) Nonecho(msg string) {
	// tell next to arrive socket msg instead of echoing what they gave us.

	po("\n\n  CountingTestServer::Nonecho() called with msg = '%s'\n\n", msg)

}

func (r *CountingTestServer) IsStopRequested() bool {
	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *CountingTestServer) Stop() {
	r.RequestStop()
	r.lsn.Close()
	<-r.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *CountingTestServer) RequestStop() bool {
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

// client

type CountingTestClient struct {
	Dest Addr

	Ready   chan bool
	reqStop chan bool
	Done    chan bool

	MsgRecvd chan string
	lastMsg  string

	mut sync.Mutex

	lastCountSent   int
	nextCountToSend int

	tmLastSend []time.Time
	tmLastRecv []time.Time
}

func NewCountingTestClient(dest Addr) *CountingTestClient {

	if dest.Port == 0 {
		panic("client's dest Addr setting must specify port to contact")
	}
	if dest.Ip == "" {
		dest.Ip = "0.0.0.0"
	}
	dest.SetIpPort()

	r := &CountingTestClient{
		MsgRecvd:        make(chan string),
		Dest:            dest,
		Ready:           make(chan bool),
		reqStop:         make(chan bool),
		Done:            make(chan bool),
		nextCountToSend: 1,
	}

	return r
}

func (cli *CountingTestClient) LastMsgReceived() string {
	return cli.lastMsg
}

func (cli *CountingTestClient) WaitForMsg() string {
	return <-cli.MsgRecvd
}

func (cli *CountingTestClient) Start() {

	go func() {
		close(cli.Ready)
		conn, err := net.Dial("tcp", cli.Dest.IpPort)
		if err != nil {
			panic(err)
		}

		buf := make([]byte, 100)

		// write-then-read loop, check for done request
		isTimeout := false
		isEOF := false
		for {

			// write
			err = conn.SetDeadline(time.Now().Add(time.Millisecond * 2000))
			panicOn(err)

			msg := fmt.Sprintf(`from CountingTestClient: client_packet_number=%d from '%s'`, cli.nextCountToSend, conn.RemoteAddr())
			nw, err := conn.Write([]byte(msg))
			panicOn(err)
			po("CountingTestClient:  after cli.Start() got to Write '%s' to conn %v -> %v.\n", msg, conn.LocalAddr(), conn.RemoteAddr())

			if nw > 0 {
				cli.NoteTmSent()

				cli.lastCountSent = cli.nextCountToSend
				cli.nextCountToSend += 2
			}

			isTimeout = false
			isEOF = false
			err = conn.SetDeadline(time.Now().Add(time.Millisecond * 2000))
			panicOn(err)

			// Read
			n, err := conn.Read(buf)
			switch {
			case err == nil:
			case strings.HasSuffix(err.Error(), "i/o timeout"):
				// okay, ignore
				isTimeout = true
			case err.Error() == "EOF":
				// when connection is shutdown, we get EOF
				isEOF = true
			default:
				panic(err)
			}
			//po("CountingTestClient: after Read, isTimeout: %v, isEOF: %v, err: %v\n", isTimeout, isEOF, err)

			if !isTimeout && !isEOF {
				sbuf := string(buf[:n])

				match := countServerRegex.FindStringSubmatch(sbuf)
				po("CountingTestClient: countClientRegex match = %#v   len(match)='%d'\n", match, len(match))
				if match == nil || len(match) != 2 {
					po("CountingTestClient: bad client message/no match to regex!: '%s", sbuf)
					continue
				}
				cli.NoteTmRecv()

				pktServerCount := match[1]

				srvCount, err := strconv.Atoi(pktServerCount)
				panicOn(err)

				if cli.lastCountSent > 0 {
					// check the sequence
					if srvCount != cli.lastCountSent+1 {
						panic(fmt.Sprintf("CountingTestClient: srvCount out of order! srvCount = %d, but cli.lastCountSeen = %d, should be incrementing one past what we sent!", srvCount, cli.lastCountSent))
					}
				}

				cli.lastMsg = string(buf[:n])
				cli.MsgRecvd <- cli.lastMsg

				po("\n CountingTestClient: message received!!! after cli.Start() got to Read '%s' from conn. n = %d bytes\n", cli.lastMsg, n)
			}

			if isEOF {
				po("CountingTestClient: got EOF, server shutting down. n=%d bytes recvd; buf = '%s'", n, string(buf[:n]))
				return
			}

			// check for stop requests
			if cli.IsStopRequested() {
				po("CountingTestClient: shutting down.")
				conn.Close()
				close(cli.Done)
				return
			}
		} // end for
	}()
}

func (r *CountingTestClient) IsStopRequested() bool {
	r.mut.Lock()
	defer r.mut.Unlock()

	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *CountingTestClient) Stop() {
	r.RequestStop()
	<-r.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *CountingTestClient) RequestStop() bool {
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

func (cli *CountingTestClient) Expect(msg string) bool {
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

func (r *CountingTestServer) NoteTmRecv() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastRecv = append(r.tmLastRecv, time.Now())
}
func (r *CountingTestServer) NoteTmSent() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastSend = append(r.tmLastSend, time.Now())
}

func (r *CountingTestClient) NoteTmRecv() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastRecv = append(r.tmLastRecv, time.Now())
}
func (r *CountingTestClient) NoteTmSent() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastSend = append(r.tmLastSend, time.Now())
}
