package main

import (
	"fmt"
	"sync"
	"time"
)

type Upstream struct {
	reqStop chan bool
	Done    chan bool
	mut     sync.Mutex

	Absorb   chan []byte
	Generate chan []byte
}

func NewUpstream() *Upstream {
	s := &Upstream{
		reqStop:  make(chan bool),
		Done:     make(chan bool),
		Absorb:   make(chan []byte),
		Generate: make(chan []byte),
	}
	return s
}

func (s *Upstream) Start() {
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
				po("upstream absorb sees '%s'", string(by))
				seen = append(seen, by...)
			case <-time.After(genDelay):
				select {
				case s.Generate <- ng:
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

func (s *Upstream) Stop() {
	s.RequestStop()
	<-s.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *Upstream) RequestStop() bool {
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
