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

	down *Downstream

	ab2lp chan []byte
	lp2ab chan []byte

	recvCount int

	tmLastSend []time.Time
	tmLastRecv []time.Time

	name string
}

func NewLittlePoll(pollDur time.Duration, dn *Downstream, ab2lp chan []byte, lp2ab chan []byte) *LittlePoll {

	s := &LittlePoll{
		reqStop:    make(chan bool),
		Done:       make(chan bool),
		pollDur:    pollDur,
		ab2lp:      ab2lp,
		lp2ab:      lp2ab,
		down:       dn,
		tmLastSend: make([]time.Time, 0),
		tmLastRecv: make([]time.Time, 0),
		name:       "LittlePoll",
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

		var pack []byte

		// in cliReq and bytesFromServer, the client is upstream and the
		// server is downstream. In LittlePoll, we read from the server
		// and write those bytes in Replies to the client. In LittlePoll, we read
		// from the client Requests and write those bytes to the server.

		// keep at most 2 cliRequests on hand, cycle them in FIFO order.
		// they are: oldestReqPack, and waitingCliReqs[0], in that order.

		waitingCliReqs := make([][]byte, 0, 2)
		var oldestReqPack []byte
		var countForUpstream int64

		curReply := make([]byte, 0, 4096)

		// abort if returns false
		sendReplyUpstream := func() bool {
			if len(oldestReqPack) == 0 {

				select {
				case s.lp2ab <- curReply: // !send
					s.NoteTmSent()

				case <-s.reqStop:
					po("lp sendReplyUpstream got reqStop, returning false")
					return false
				}
				countForUpstream = 0
				if len(waitingCliReqs) > 0 {
					oldestReqPack = waitingCliReqs[0]
					waitingCliReqs = waitingCliReqs[1:]
				} else {
					oldestReqPack = oldestReqPack[:0]
					longPollTimeUp.Stop()
				}
			}
			return true
		}

		for {
			po("%p longpoller: at top of LittlePoll loop, inside Start(). len(wait)=%d", s, len(waitingCliReqs))

			if oldestReqPack != nil {
				po("%p  longpoller: at top of LittlePoll loop, inside Start(). string(oldestReqPack.body='%s'", s, string(oldestReqPack))
			} else {
				po("%p  longpoller: oldestReqPack = nil", s)
			}
			select {

			case <-longPollTimeUp.C:
				po("%p  longPollTimeUp!!", s)
				if !sendReplyUpstream() {
					return
				}

			// Only receive if we have a waiting packet body to write to.
			// Otherwise let the RecvFromDownCh() do the fixed size buffering.
			case b500 := <-s.down.Generate:
				po("%p  LittlePoll got data from downstream <-s.rw.RecvFromDownCh() got b500='%s'\n", s, string(b500))

				curReply = append(curReply, b500...)
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

				po("%p  LittlePoll, just received ClientPacket with pack.body = '%s'\n", s, string(pack))

				// have to both send and receive

				po("%p  just before s.rw.SendToDownCh()", s)

				// we got data from the client for server!
				// read from the request body and write to the ResponseWriter
				select {
				// s.rw.SendToDownCh() is a 1000 buffered channel so okay to not use a timeout;
				// in fact we do want the back pressure to keep us from
				// writing too much too fast.
				case s.down.Absorb <- pack:
					po("%p  sent data on s.downAbsorb <- pack", s)
					//po("%p  sent data on s.rw.SendToDownCh()", s)
				case <-s.reqStop:
					po("%p  got reqStop, *not* returning", s)
					// avoid deadlock on shutdown, but do
					// finish processing this packet, don't return yet
				}
				po("%p  just after s.down.Absorb <- pack", s)
				//po("%p  just after s.rw.SendToDownCh()", s)

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
				case b500 := <-s.down.Generate:
					po("%p  longpoller  <-s.rw.RecvFromDownCh() got b500='%s'\n", s, string(b500))
					curReply = append(curReply, b500...)

				case <-time.After(10 * time.Millisecond):
					po("%p  after 10msec of extra s.rw.RecvFromDownCh() reads", s)

					// stop trying to read from server downstream, and send what
					// we got upstream to client.
				}

				if countForUpstream > 0 || len(waitingCliReqs) > 0 {
					if !sendReplyUpstream() {
						return
					}
				} else {
					po("%p  LongPoll countForUpstream(%d); len(waitingCliReqs)==%d  ...response so far: '%s'", s, countForUpstream, len(waitingCliReqs), string(oldestReqPack))
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
	nr := len(r.tmLastRecv)
	ns := len(r.tmLastSend)
	min := nr
	if ns < min {
		min = ns
	}
	if min == 0 {
		fmt.Printf("%s history: none.\n", r.name)
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
