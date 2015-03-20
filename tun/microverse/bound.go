package main

import (
	"fmt"
	mathrand "math/rand"
	"sync"
	"time"
)

// Boundary represents the most downstream or most upstream
// party, contains a name to distinguish who this is, and
// provides SetEcho() and SendEvery() methods to configure
// the sending behavior. It records its traffic for history
// analysis.
type Boundary struct {
	reqStop chan bool
	Done    chan bool
	mut     sync.Mutex

	Absorb   chan []byte
	Generate chan []byte

	hist *HistoryLog

	doEcho    bool
	sendEvery time.Duration
	name      string
}

func NewBoundary(name string) *Boundary {
	s := &Boundary{
		reqStop:  make(chan bool),
		Done:     make(chan bool),
		Absorb:   make(chan []byte),
		Generate: make(chan []byte),
		hist:     NewHistoryLog(name),
		name:     name,
	}
	return s
}

func (s *Boundary) SendEvery(dur time.Duration) {
	s.sendEvery = dur
	po("%s SendEvery called(): s.sendEvery = %v", s.name, dur)
}

func (s *Boundary) SetEcho(echo bool) {
	s.doEcho = echo
	po("%s SetEcho called(): s.doEcho = %v", s.name, s.doEcho)

}

func (s *Boundary) Start() {

	// make a stopped timer, the default, the 24 hours is
	// just to be sure we get to call genTimer.Stop() before
	// the timer goes off.
	genTimer := time.NewTimer(24 * time.Hour)
	genTimer.Stop()

	if s.sendEvery == 0 {
		po("******* %s::Start() has no sendEvery set! I'll never send anything unless you SetEcho(true).", s.name)
	} else {
		// start the timer
		genTimer.Reset(s.sendEvery)
	}

	var err error
	nextGen := 0
	ng := []byte(fmt.Sprintf("..origin:%s:%d..", s.name, nextGen))
	seen := []byte{}
	go func() {
		for {
			select {
			case <-s.reqStop:
				close(s.Done)
				return
			case by := <-s.Absorb:
				s.hist.RecordAbs(by)
				po("%s absorb sees '%s'", s.name, string(by))
				seen = append(seen, by...)

				if s.doEcho {
					po("%s is echoing '%s'", s.name, string(by))
					echo := []byte(fmt.Sprintf("..%s echo of ('%s')..", s.name, string(by)))
					err = s.Gen(echo)
					if err != nil {
						if err.Error() == "shutdown" {
							return
						}
						panic(err)
					}
				}

			case <-genTimer.C:
				po("%s <-genTimer.C fired, generating: '%s'", s.name, string(ng))
				genTimer.Reset(s.sendEvery)

				err = s.Gen(ng)
				if err != nil {
					if err.Error() == "shutdown" {
						return
					}
					panic(err)
				}
				po("%s Generate returned '%s'", s.name, ng)
				nextGen++
				ng = []byte(fmt.Sprintf("%d", nextGen))

			}
		}
	}()
}

func (s *Boundary) Gen(by []byte) error {
	po("%s Boundary Gen('%s') called.", s.name, string(by))
	select {
	case s.Generate <- by:
		s.hist.RecordGen(by)
	case <-s.reqStop:
		close(s.Done)
		return fmt.Errorf("shutdown")
	}

	return nil
}

func (s *Boundary) Stop() {
	s.RequestStop()
	<-s.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *Boundary) RequestStop() bool {
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

func RandSlice(nbytes int) []byte {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	b := make([]byte, nbytes)
	for i := 0; i < nbytes; i++ {
		b[i] = byte(r.Uint32() % 256)
	}
	return b
}
