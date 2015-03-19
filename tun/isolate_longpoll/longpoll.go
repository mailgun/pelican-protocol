package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// A LongPoller (aka tunnel) is the server-side implementation
// of long-polling. We connect the http client (our pelican socks proxy)
// with the downstream target, typically an http server or sshd.
// For the client side implementation of long polling, see the
// file alphabeta.go and the Chaser structure and methods.
//
// Inside the reverse proxy, the LongPoller represents a 1:1, one
// client to one (downstream target) server connection,
// if you ignore the socks-proxy and reverse-proxy in the middle.
// A ReverseProxy can have many LongPollers, mirroring the number of
// connections on the client side to the socks proxy. The key
// distinguishes them. The LongerPoller is where we implement the
// server side of the long polling.
//
// http request flow (client initiating direction), http replies
// flow in the opposite direction of the arrows below.
//
//        "upstream"                               "downstream"
//           V                                         ^
//     e.g. web-browser                          e.g. web-server
//           |                                         ^
//           v                                         |
// -----------------------             -------------------------
// | TcpUpstreamReceiver |             |  net.Conn TCP connect |
// |    |                |             |               ^       |
// |    v                |             |           ServerRW    |
// | ClientRW            |             |               ^       |
// |    v                |    http     |               |       |
// | Chaser->alpha/beta->|------------>|WebServer--> LongPoller|
// -----------------------             -------------------------
//   pelican-socks-proxy                 pelican-reverse-proxy
//
//
type LongPoller struct {
	reqStop           chan bool
	Done              chan bool
	ClientPacketRecvd chan *tunnelPacket

	rw        *ServerRW // manage the goroutines that read and write dnConn
	recvCount int
	conn      net.Conn

	// server issues a unique key for the connection, which allows multiplexing
	// of multiple client connections from this one ip if need be.
	// The ssh integrity checks inside the tunnel prevent malicious tampering.
	key     string
	pollDur time.Duration

	Dest Addr

	mut          sync.Mutex
	CloseKeyChan chan string
}

// Make a new LongPoller as a part of the server (ReverseProxy is the server;
// PelicanSocksProxy is the client).
//
// If a CloseKeyChan receives a key, we return any associated client -> server
// http request immediately for that key, to facilitate quick shutdown.
//
func NewLongPoller(dest Addr, pollDur time.Duration) *LongPoller {
	key := GenPelicanKey()
	if dest.Port == 0 {
		dest.Port = GetAvailPort()
	}
	if dest.Ip == "" {
		dest.Ip = "0.0.0.0"
	}
	dest.SetIpPort()

	s := &LongPoller{
		reqStop:           make(chan bool),
		Done:              make(chan bool),
		ClientPacketRecvd: make(chan *tunnelPacket),
		key:               string(key),
		Dest:              dest,
		CloseKeyChan:      make(chan string),
		pollDur:           pollDur,
	}

	return s
}

func (s *LongPoller) Stop() {
	po("%p LongPoller stop received", s)
	s.RequestStop()
	<-s.Done
	po("%p LongPoller stop done", s)
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *LongPoller) RequestStop() bool {
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

func (s *LongPoller) finish() {
	s.rw.Stop()
	close(s.Done)
}

// LongPoller::Start() implements the long-polling logic.
//
// When a new client request comes in (2nd one), we bump any
// already waiting long-poll into replying to its request.
//
//     new reader ------> bumps waiting/long-polling reader & takes its place.
//       ^                      |
//       |                      V
//       ^                      |
//       |                      V
//    client <-- returns to <---/
//
// it's a closed loop track with only one goroutine per tunnel
// actively holding on a long poll.
//
// There are only ever two client (http) requests outstanding
// at any given moment in time.
//
func (s *LongPoller) Start() error {

	skey := string(s.key[:5])

	err := s.dial()
	if err != nil {
		return fmt.Errorf("%s '%s' LongPoller could not dial '%s': '%s'", s, skey, s.Dest.IpPort, err)
	}

	// s.dial() sets s.conn on success.
	s.rw = NewServerRW(s.conn, 0, nil, nil)
	s.rw.Start()

	go func() {
		defer func() { s.finish() }()

		// duration of the long poll
		longPollTimeUp := time.After(s.pollDur)

		var pack *tunnelPacket

		// in cliReq and bytesFromServer, the client is upstream and the
		// server is downstream. In LongPoller, we read from the server
		// and write those bytes in Replies to the client. In LongPoller, we read
		// from the client Requests and write those bytes to the server.

		// keep at most 2 cliRequests on hand, cycle them in FIFO order.
		// they are: oldestReqPack, and waitingCliReqs[0], in that order.
		waitingCliReqs := make([]*tunnelPacket, 0, 2)
		var oldestReqPack *tunnelPacket
		var countForUpstream int64

		// sends replies upsteram
		sendUp := func() {
			if oldestReqPack != nil {
				po("%p '%s' LongPoll::Start(): sendUp() is sending along oldest ClientRequest with response, countForUpstream(%d) >0 || len(waitingCliReqs)==%d was > 0   ...response: '%s'", s, skey, countForUpstream, len(waitingCliReqs), string(oldestReqPack.respdup.Bytes()))
				close(oldestReqPack.done) // send!
				countForUpstream = 0
				if len(waitingCliReqs) > 0 {
					oldestReqPack = waitingCliReqs[0]
					waitingCliReqs = waitingCliReqs[1:]
				} else {
					oldestReqPack = nil
					longPollTimeUp = nil
				}
			}
		}

		for {
			po("%p '%s' longpoller: at top of LongPoller loop, inside Start(). len(wait)=%d", s, skey, len(waitingCliReqs))

			if oldestReqPack != nil {
				po("%p '%s' longpoller: at top of LongPoller loop, inside Start(). string(oldestReqPack.body='%s'", s, skey, string(oldestReqPack.body))
			} else {
				po("%p '%s' longpoller: oldestReqPack = nil", s, skey)
			}
			select {

			case <-longPollTimeUp:
				po("longPollTimeUp!!")
				// SEND reply! (by closing oldestReq.done)
				sendUp()

			// Only receive if we have a waiting packet body to write to.
			// Otherwise let the RecvFromDownCh() do the fixed size buffering.
			case b500 := <-func() chan []byte {
				if oldestReqPack != nil {
					return s.rw.RecvFromDownCh()
				} else {
					return nil
				}
			}():
				po("%p '%s' LongPoller got data from downstream <-s.rw.RecvFromDownCh() got b500='%s'\n", s, skey, string(b500))

				countForUpstream += int64(len(b500))
				_, err := oldestReqPack.resp.Write(b500)
				if err != nil {
					panic(err)
				}

				_, err = oldestReqPack.respdup.Write(b500)
				if err != nil {
					panic(err)
				}
				sendUp()

			case pack = <-s.ClientPacketRecvd:
				s.recvCount++
				po("%p '%s' longPoller got client packet! recvCount now: %d", s, skey, s.recvCount)

				// reset timer. only hold this packet open for at most 'dur' time.
				// since we will be replying to oldestReqPack (if any) immediately,
				// we can replace this timer.
				// TODO: is their a simpler reset instead of replace the timer?
				longPollTimeUp = time.After(s.pollDur)

				po("%p '%s' LongPoller, just received ClientPacket with pack.body = '%s'\n", s, skey, string(pack.body))

				// have to both send and receive

				pack.resp.Header().Set("Content-type", "application/octet-stream")

				// we got data from the client for server!
				// read from the request body and write to the ResponseWriter
				select {
				// s.rw.SendToDownCh() is a 1000 buffered channel so okay to not use a timeout;
				// in fact we do want the back pressure to keep us from
				// writing too much too fast.
				case s.rw.SendToDownCh() <- pack.body:

				case <-s.reqStop:
					// avoid deadlock on shutdown, but do
					// finish processing this packet, don't return yet
				}

				// transfer data from server to client

				// get the oldest packet, and reply using that. http requests
				// get serviced mostly FIFO this way, and our long-poll
				// timer reflects the time since the most recent packet
				// arrival.
				waitingCliReqs = append(waitingCliReqs, pack)
				oldestReqPack = waitingCliReqs[0]
				waitingCliReqs = waitingCliReqs[1:]

				// add any data from the next 10 msec to return packet to client
				select {
				case b500 := <-s.rw.RecvFromDownCh():
					po("%p '%s' longpoller  <-s.rw.RecvFromDownCh() got b500='%s'\n", s, skey, string(b500))

					countForUpstream += int64(len(b500))
					_, err := oldestReqPack.resp.Write(b500)
					if err != nil {
						panic(err)
					}

					_, err = oldestReqPack.respdup.Write(b500)
					if err != nil {
						panic(err)
					}

				case <-time.After(10 * time.Millisecond):
					// stop trying to read from server downstream, and send what
					// we got upstream to client.
				}

				if countForUpstream > 0 || len(waitingCliReqs) > 0 {
					sendUp()
				} else {
					po("%p '%s' LongPoll countForUpstream(%d); len(waitingCliReqs)==%d  ...response so far: '%s'", s, skey, countForUpstream, len(waitingCliReqs), string(oldestReqPack.respdup.Bytes()))
				}

				// end case pack = <-s.ClientPacketRecvd:
			case <-s.reqStop:
				return
			case <-s.CloseKeyChan:
				po("%p '%s' LongPoller in nil packet state, got closekeychan. Shutting down.", s, skey)

				// empty out the oldest and wait queue, replying to zero, one, or both requests.
				if oldestReqPack != nil {
					close(oldestReqPack.done)
					for _, p := range waitingCliReqs {
						close(p.done)
					}
					waitingCliReqs = waitingCliReqs[len(waitingCliReqs):]
					oldestReqPack = nil
				}
				return
			} //end select
		} // end for

		/*
				// *** not sure this is correct: where is the 2nd packet held open???

				// wait for a read for a possibly long duration. this is the "long poll" part.
				dur := 30 * time.Second
				// the client will spin up another goroutine/thread/sender if it has
				// an additional send in the meantime.

				po("LongPoll::Start(): tunnel.go starting to wait up to %v", dur)

				var n64 int64
				longPollTimeUp := time.After(dur)

				select {
				case <-s.reqStop:
					close(pack.done)
					pack = nil
					return

				case b500 := <-s.rw.RecvFromDownCh():
					po("tunnel.go: <-s.rw.RecvFromDownCh() got b500='%s'\n", string(b500))

					n64 += int64(len(b500))
					_, err := pack.resp.Write(b500)
					if err != nil {
						panic(err)
					}

					_, err = pack.respdup.Write(b500)
					if err != nil {
						panic(err)
					}
					close(pack.done)
					pack = nil

				case <-longPollTimeUp:
					po("tunnel.go: longPollTimeUp!!\n")
					// send it along its way anyhow
					close(pack.done)
					pack = nil

				case <-s.CloseKeyChan:
					po("tunnel.go: LongPoller with pending packet got closekey. returning packet and then exiting LongPoller")
					close(pack.done)
					pack = nil
					return

				case newpacket := <-s.ClientPacketRecvd:
					po("tunnel.go: <-s.ClientPakcetRecvd!!: %#v\n", newpacket)
					s.recvCount++
					// finish previous packet without data, because client sent another packet
					close(pack.done)
					pack = newpacket
				}

				po("LongPoll::Start(): at end of select/long wait.")
			}
		*/
	}()

	return nil
}

func (s *LongPoller) dial() error {

	po("ReverseProxy::NewTunnel: Attempting connect to our target '%s'\n", s.Dest.IpPort)
	dialer := net.Dialer{
		Timeout:   1000 * time.Millisecond,
		KeepAlive: 30 * time.Second,
	}

	var err error
	s.conn, err = dialer.Dial("tcp", s.Dest.IpPort)
	switch err.(type) {
	case *net.OpError:
		if strings.HasSuffix(err.Error(), "connection refused") {
			// could not reach destination
			return err
		}
	default:
		panicOn(err)
	}

	return err
}
