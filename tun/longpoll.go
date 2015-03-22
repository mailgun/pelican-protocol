package pelicantun

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
	lastUseTm    time.Time

	nextReplySerial             int64
	lastRequestSerialNumberSeen int64
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
		nextReplySerial:   1,
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

func (s *LongPoller) getReplySerial() int64 {
	s.mut.Lock()
	defer s.mut.Unlock()
	v := s.nextReplySerial
	s.nextReplySerial++
	return v
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
	s.rw = NewServerRW("ServerRW/LongPoller", s.conn, 0, nil, nil, s)
	s.rw.Start()

	go func() {
		defer func() { s.finish() }()

		// duration of the long poll

		longPollTimeUp := time.NewTimer(s.pollDur)

		var pack *tunnelPacket

		// in cliReq and bytesFromServer, the client is upstream and the
		// server is downstream. In LongPoller, we read from the server
		// and write those bytes in Replies to the client. In LongPoller, we read
		// from the client Requests and write those bytes to the server.

		// keep at most 2 cliRequests on hand, cycle them in FIFO order.
		// they are: oldestReqPack, and waitingCliReqs[0], in that order.

		waiters := NewRequestFifo(2)

		var countForUpstream int64

		sendReplyUpstream := func() bool {
			if waiters.Empty() {
				return true
			}

			oldest := waiters.PopRight()

			po("%p '%s' longpoller sendReplyUpstream() is sending along oldest ClientRequest with response, countForUpstream(%d) >0 || len(waitingCliReqs)==%d was > 0", s, skey, countForUpstream, waiters.Len())

			if countForUpstream > 0 {
				// last thing before the reply: append reply serial number, to allow
				// correct ordering on the client end. But skip replySerialNumber
				// addition if this is an empty packet, because there will be lots of those.
				//
				rs := s.getReplySerial()
				rser := SerialToBytes(rs)
				nw, err := oldest.resp.Write(rser)
				if err != nil {
					panic(err)
				}
				if nw != len(rser) {
					panic(fmt.Sprintf("short write: tried to write %d, but wrote %d", len(rser), nw))
				}
				nw, err = oldest.respdup.Write(rser)
				if err != nil {
					panic(err)
				}
				if nw != len(rser) {
					panic(fmt.Sprintf("short write: tried to write %d, but wrote %d", len(rser), nw))
				}
				oldest.replySerial = rs

			} else {
				oldest.replySerial = -1
			}

			close(oldest.done) // send reply!
			countForUpstream = 0

			if waiters.Empty() {
				longPollTimeUp.Stop()
			}
			return true
		} // end sendReplyUpstream

		for {
			po("%p '%s' longpoller: at top of LongPoller loop, inside Start(). len(wait)=%d", s, skey, waiters.Len())

			select {

			case <-longPollTimeUp.C:
				po("%p '%s' longPollTimeUp!!", s, skey)
				sendReplyUpstream()

			// Only receive if we have a waiting packet body to write to.
			// Otherwise let the RecvFromDownCh() do the fixed size buffering.
			case b500 := <-func() chan []byte {
				if !waiters.Empty() {
					return s.rw.RecvFromDownCh()
				} else {
					return nil
				}
			}():
				if len(b500) > 0 {
					s.lastUseTm = time.Now()
				}
				po("%p '%s' LongPoller got data from downstream <-s.rw.RecvFromDownCh() got b500='%s'\n", s, skey, string(b500))

				oldestReqPack := waiters.PeekRight()
				_, err := oldestReqPack.resp.Write(b500)
				if err != nil {
					panic(err)
				}
				countForUpstream += int64(len(b500))

				_, err = oldestReqPack.respdup.Write(b500)
				if err != nil {
					panic(err)
				}
				sendReplyUpstream()

			case pack = <-s.ClientPacketRecvd:
				s.recvCount++
				if len(pack.reqBody) > 0 {
					s.lastUseTm = time.Now()
				}

				po("%p '%s' longPoller got client packet! recvCount now: %d", s, skey, s.recvCount)

				if pack.requestSerial >= 0 {
					if pack.requestSerial != s.lastRequestSerialNumberSeen+1 {
						panic(fmt.Sprintf("pack.requestSerial =%d but s.lastRequestSerialNumberSeen = %d which is not one less", pack.requestSerial, s.lastRequestSerialNumberSeen))
					} else {
						s.lastRequestSerialNumberSeen++
					}
				}
				// reset timer. only hold this packet open for at most 'dur' time.
				// since we will be replying to oldestReqPack (if any) immediately,
				// we can reset the timer to reflect pack's arrival.
				longPollTimeUp.Reset(s.pollDur)

				po("%p '%s' LongPoller, just received ClientPacket with pack.reqBody = '%s'\n", s, skey, string(pack.reqBody))

				// have to both send and receive

				pack.resp.Header().Set("Content-type", "application/octet-stream")

				po("%p '%s' just before s.rw.SendToDownCh()", s, skey)

				if len(pack.reqBody) > 0 {
					// we got data from the client for server!
					// read from the request body and write to the ResponseWriter
					select {
					// s.rw.SendToDownCh() is a 1000 buffered channel so okay to not use a timeout;
					// in fact we do want the back pressure to keep us from
					// writing too much too fast.
					case s.rw.SendToDownCh() <- pack.reqBody:
						po("%p '%s' sent data on s.rw.SendToDownCh()", s, skey)
					case <-s.reqStop:
						po("%p '%s' got reqStop, *not* returning", s, skey)
						// avoid deadlock on shutdown, but do
						// finish processing this packet, don't return yet
					}
				} // end if len(pack.reqBody) > 0

				po("%p '%s' just after s.rw.SendToDownCh()", s, skey)

				// transfer data from server to client

				// get the oldest packet, and reply using that. http requests
				// get serviced mostly FIFO this way, and our long-poll
				// timer reflects the time since the most recent packet
				// arrival.
				waiters.PushLeft(pack)

				// TODO: instead of fixed 10msec, this threshold should be
				// 1x the one-way-trip time from the client-to-server, since that is
				// the expected additional alternative wait time if we have to reply
				// with an empty reply now.
				//
				// add any data from the next 10 msec to return packet to client
				// hence if the server replies quickly, we can reply quickly
				// to the client too.
				startFastWaitTm := time.Now()
				select {
				case b500 := <-s.rw.RecvFromDownCh():
					po("%p '%s' longpoller  <-s.rw.RecvFromDownCh() got b500='%s' during fast-reply-wait (after %v)\n", s, skey, string(b500), time.Since(startFastWaitTm))

					oldest := waiters.PeekRight()
					_, err := oldest.resp.Write(b500)
					if err != nil {
						panic(err)
					}
					countForUpstream += int64(len(b500))

					_, err = oldest.respdup.Write(b500)
					if err != nil {
						panic(err)
					}

				case <-time.After(10 * time.Millisecond):
					po("%p '%s' after 10msec of extra s.rw.RecvFromDownCh() reads", s, skey)

					// stop trying to read from server downstream, and send what
					// we got upstream to client.
				}

				// key piece of logic for the long-poll is here:
				// reply immediately under two conditions: there
				// are bytes to send back upstream, or we have
				// more than one of the alpha/beta parked here.
				if countForUpstream > 0 || waiters.Len() > 1 {
					sendReplyUpstream()
				} else {
					po("%p '%s' LongPoll countForUpstream(%d); len(waitingCliReqs)==%d",
						s, skey, countForUpstream, waiters.Len())
				}

				// end case pack = <-s.ClientPacketRecvd:
			case <-s.reqStop:
				po("%p '%s' got reqStop, returning", s, skey)
				return
			case <-s.CloseKeyChan:
				po("%p '%s' LongPoller in nil packet state, got closekeychan. Shutting down.", s, skey)

				// empty out the oldest and wait queue, replying to zero, one, or both requests.
				p := waiters.PopRight()
				for p != nil {
					close(p.done)
					p = waiters.PopRight()
				}
				return
			} //end select
		} // end for

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
