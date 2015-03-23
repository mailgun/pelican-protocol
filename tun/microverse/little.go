package main

import (
	"fmt"
	"sync"
	"time"
)

type LittlePoll struct {
	reqStop chan bool
	Done    chan bool

	pollDur time.Duration

	mut sync.Mutex

	down *Boundary

	ab2lp chan *tunnelPacket
	lp2ab chan *tunnelPacket

	recvCount int

	tmLastSend []time.Time
	tmLastRecv []time.Time

	name string

	key       string // keep rw happy
	lastUseTm time.Time

	nextReplySerial             int64
	lastRequestSerialNumberSeen int64
}

func NewLittlePoll(pollDur time.Duration, dn *Boundary, ab2lp chan *tunnelPacket, lp2ab chan *tunnelPacket) *LittlePoll {

	s := &LittlePoll{
		reqStop:         make(chan bool),
		Done:            make(chan bool),
		pollDur:         pollDur,
		ab2lp:           ab2lp, // receive from "socks-proxy" (Chaser)
		lp2ab:           lp2ab, // send to "socks-proxy" (Chaser)
		down:            dn,    // the "web-server", downstream most boundary target.
		tmLastSend:      make([]time.Time, 0),
		tmLastRecv:      make([]time.Time, 0),
		name:            "LittlePoll",
		nextReplySerial: 1,
	}

	return s
}

func (s *LittlePoll) Stop() {
	po("%p LittlePoll stop received", s)
	s.RequestStop()
	<-s.Done
	po("%p LittlePoll stop done", s)
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *LittlePoll) RequestStop() bool {
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

func (s *LittlePoll) finish() {
	close(s.Done)
}

// LittlePoll::Start() implements the long-polling logic.
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

func (s *LittlePoll) Start() error {

	go func() {
		defer func() { s.finish() }()

		// duration of the long poll

		longPollTimeUp := time.NewTimer(s.pollDur)

		var pack *tunnelPacket

		// in cliReq and bytesFromServer, the client is upstream and the
		// server is downstream. In LittlePoll, we read from the server
		// and write those bytes in Replies to the client. In LittlePoll, we read
		// from the client Requests and write those bytes to the server.

		// keep at most 2 cliRequests on hand, cycle them in FIFO order.
		// they are: oldestReqPack, and waitingCliReqs[0], in that order.

		waiters := NewRequestFifo(2)

		var countForUpstream int64

		curReply := make([]byte, 0, 4096)

		// tries to send, and does if we have
		// a waiting request to send on.
		//
		// returns false iff we got s.reqStop
		// while trying to send.
		sendReplyUpstream := func() bool {

			if waiters.Empty() {
				return true
			}

			oldest := waiters.PopRight()

			po("%p LittlePoll sendReplyUpstream() is sending along oldest ClientRequest with response, countForUpstream(%d) >0 || len(waitingCliReqs)==%d was > 0", s, countForUpstream, waiters.Len())

			if countForUpstream != int64(len(oldest.respdup.Bytes())) {
				panic(fmt.Sprintf("should never get here: countForUpstream is out of sync with oldest.respdup.Bytes(): %d == countForUpstream != len(oldest.respdup.Bytes()) == %d", countForUpstream, len(oldest.respdup.Bytes())))
			}

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

				if countForUpstream != int64(len(oldest.respdup.Bytes())) {
					panic(fmt.Sprintf("should never get here: countForUpstream is out of sync with oldest.respdup.Bytes(): %d == countForUpstream != len(oldest.respdup.Bytes()) == %d", countForUpstream, len(oldest.respdup.Bytes())))
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

			// little only -- this actually does the send reply in the microverse.
			select {
			case s.lp2ab <- oldest:
				//okay
			case <-s.reqStop:
				// shutting down
				return false
			}

			countForUpstream = 0

			if waiters.Empty() {
				longPollTimeUp.Stop()
			}
			return true

			/* new
			select {
			// in real code: send curReply on r, where
			// r:= waiters.PeekRight()

			case s.lp2ab <- curReply: // send
				waiters.PopRight()
				s.NoteTmSent()
				countForUpstream = 0
				curReply = curReply[:0]

			case <-s.reqStop:
				po("lp sendReplyUpstream got reqStop, returning false")
				return false
			}

			if waiters.Empty() {
				longPollTimeUp.Stop()
			}
			return true
			*/
		}

		for {
			po("%p longpoller: at top of LittlePoll loop, inside Start(). len(wait)=%d", s, waiters.Len())

			select {

			case <-longPollTimeUp.C:
				po("%p  longPollTimeUp!!", s)
				if !sendReplyUpstream() {
					return
				}

			// Only receive if we have a waiting packet body to write to.
			// Otherwise let the RecvFromDownCh() do the fixed size buffering.
			case b500 := <-func() chan []byte {
				if !waiters.Empty() {
					return s.down.Generate // compare longpoller.go: return s.rw.RecvFromDownCh()
				} else {
					return nil
				}
			}():
				if len(b500) > 0 {
					s.lastUseTm = time.Now()
				}

				oldestReqPack := waiters.PeekRight()
				po("%p  LittlePoll got data from downstream <-s.rw.RecvFromDownCh() got b500='%s'. oldestReqPack.respdup.Bytes() = '%s'\n", s, string(b500), string(oldestReqPack.respdup.Bytes()))

				_, err := oldestReqPack.resp.Write(b500)
				if err != nil {
					panic(err)
				}
				countForUpstream += int64(len(b500))

				_, err = oldestReqPack.respdup.Write(b500)
				if err != nil {
					panic(err)
				}

				if !sendReplyUpstream() {
					return
				}

			case pack = <-s.ab2lp:
				s.recvCount++
				s.NoteTmRecv()
				po("%p  longPoller got client packet! recvCount now: %d", s, s.recvCount)

				// reset timer. only hold this packet open for at most 'dur' time.
				// since we will be replying to oldestReqPack (if any) immediately,
				// we can reset the timer to reflect pack's arrival.
				longPollTimeUp.Reset(s.pollDur)

				po("%p  LittlePoll, just received ClientPacket with pack.reqBody = '%s'\n", s, string(pack.reqBody))

				// have to both send and receive

				po("%p  just before s.rw.SendToDownCh()", s)

				if len(pack.reqBody) > 0 {
					// we got data from the client for server!
					// read from the request body and write to the ResponseWriter
					select {
					// s.rw.SendToDownCh() is a 1000 buffered channel so okay to not use a timeout;
					// in fact we do want the back pressure to keep us from
					// writing too much too fast.
					case s.down.Absorb <- pack.reqBody:
						po("%p  sent data on s.downAbsorb <- pack", s)
						//po("%p  sent data on s.rw.SendToDownCh()", s)
					case <-s.reqStop:
						po("%p  got reqStop, *not* returning", s)
						// avoid deadlock on shutdown, but do
						// finish processing this packet, don't return yet
					}
				} // end if len(pack) > 0

				po("%p  just after s.down.Absorb <- pack", s)
				//po("%p  just after s.rw.SendToDownCh()", s)

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
				select {
				case b500 := <-s.down.Generate:
					po("%p  longpoller  <-s.rw.RecvFromDownCh() got b500='%s'\n", s, string(b500))

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
					po("%p  after 10msec of extra s.rw.RecvFromDownCh() reads", s)

					// stop trying to read from server downstream, and send what
					// we got upstream to client.
				}

				// key piece of logic for the long-poll is here:
				// reply immediately under two conditions: there
				// are bytes to send back upstream, or we have
				// more than one of the alpha/beta parked here.
				if countForUpstream > 0 || waiters.Len() > 1 {
					if !sendReplyUpstream() {
						return
					}
				} else {
					po("%p  LongPoll len(curReply) == %d; waiters.Len()==%d", s, len(curReply), waiters.Len())
				}

				// end case pack = <-s.ClientPacketRecvd:
			case <-s.reqStop:
				po("%p  got reqStop, returning", s)
				return

			} //end select
		} // end for

	}()

	return nil
}

// logging

func (r *LittlePoll) NoteTmRecv() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastRecv = append(r.tmLastRecv, time.Now())
}

func (r *LittlePoll) NoteTmSent() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastSend = append(r.tmLastSend, time.Now())
}

func (r *LittlePoll) ShowTmHistory() {
	r.mut.Lock()
	defer r.mut.Unlock()
	po("LittlePoll.ShowTmHistory() called.")
	nr := len(r.tmLastRecv)
	ns := len(r.tmLastSend)
	min := nr
	if ns < min {
		min = ns
	}
	fmt.Printf("%s history: ns=%d.  nr=%d.  min=%d.\n", r.name, ns, nr, min)

	for i := 0; i < ns; i++ {
		fmt.Printf("%s history of Send from LP to AB '%v'  \n",
			r.name,
			r.tmLastSend[i])
	}

	for i := 0; i < nr; i++ {
		fmt.Printf("%s history of Recv from AB at LP '%v'  \n",
			r.name,
			r.tmLastRecv[i])
	}

	for i := 0; i < min; i++ {
		fmt.Printf("%s history: elap: '%s'    Recv '%v'   Send '%v'  \n",
			r.name,
			r.tmLastSend[i].Sub(r.tmLastRecv[i]),
			r.tmLastRecv[i], r.tmLastSend[i])
	}

	for i := 0; i < min-1; i++ {
		fmt.Printf("%s history: send-to-send elap: '%s'\n", r.name, r.tmLastSend[i+1].Sub(r.tmLastSend[i]))
	}
	for i := 0; i < min-1; i++ {
		fmt.Printf("%s history: recv-to-recv elap: '%s'\n", r.name, r.tmLastRecv[i+1].Sub(r.tmLastRecv[i]))
	}

}

func (s *LittlePoll) getReplySerial() int64 {
	s.mut.Lock()
	defer s.mut.Unlock()
	v := s.nextReplySerial
	s.nextReplySerial++
	return v
}
