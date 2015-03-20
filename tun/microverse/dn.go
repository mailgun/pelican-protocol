package main

import (
	"fmt"
	mathrand "math/rand"
	"sync"
	"time"
)

type Downstream struct {
	reqStop chan bool
	Done    chan bool
	mut     sync.Mutex

	Absorb   chan []byte
	Generate chan []byte

	hist *HistoryLog
}

func NewDownstream() *Downstream {
	s := &Downstream{
		reqStop:  make(chan bool),
		Done:     make(chan bool),
		Absorb:   make(chan []byte),
		Generate: make(chan []byte),
		hist:     NewHistoryLog("downstream"),
	}
	return s
}

func (s *Downstream) Start() {
	genDelay := 5 * time.Second

	nextGen := 0
	ng := []byte(fmt.Sprintf("%d", nextGen))
	seen := []byte{}
	go func() {
		for {
			select {
			case <-s.reqStop:
				close(s.Done)
				return
			case by := <-s.Absorb:
				s.hist.RecordAbs(by)
				po("upstream absorb sees '%s'", string(by))
				seen = append(seen, by...)
			case <-time.After(genDelay):
				select {
				case s.Generate <- ng:
					s.hist.RecordGen(ng)
				case <-s.reqStop:
					close(s.Done)
					return
				}
				po("upstream Generate returned '%s'", ng)
				nextGen++
				ng = []byte(fmt.Sprintf("%d", nextGen))
			}
		}
	}()
}

func (s *Downstream) Stop() {
	s.RequestStop()
	<-s.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *Downstream) RequestStop() bool {
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
