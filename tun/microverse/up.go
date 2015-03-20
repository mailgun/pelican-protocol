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

	generateHistory []*Log
	absorbHistory   []*Log
}

func (s *Upstream) recordGen(what []byte) {
	cp := make([]byte, len(what))
	copy(cp, what)
	s.generateHistory = append(s.generateHistory, &Log{when: time.Now(), what: cp})
	s.absorbHistory = append(s.absorbHistory, &Log{}) // make spacing apparent
}

func (s *Upstream) recordAbs(what []byte) {
	cp := make([]byte, len(what))
	copy(cp, what)
	s.absorbHistory = append(s.absorbHistory, &Log{when: time.Now(), what: cp})
	s.generateHistory = append(s.generateHistory, &Log{})
}

func (s *Upstream) showHistory() {
	fmt.Printf("Upstream history:\n")
	for i := 0; i < len(s.absorbHistory); i++ {
		if s.absorbHistory[i].when.IsZero() {

		} else {
			fmt.Printf("Abs @ %v: '%s'\n",
				s.absorbHistory[i].when,
				string(s.absorbHistory[i].what))
		}

		if s.generateHistory[i].when.IsZero() {

		} else {
			fmt.Printf("Gen @ %v:                  '%s'\n",
				s.generateHistory[i].when,
				string(s.generateHistory[i].what))
		}
	}
}

func NewUpstream() *Upstream {
	s := &Upstream{
		reqStop:         make(chan bool),
		Done:            make(chan bool),
		Absorb:          make(chan []byte),
		Generate:        make(chan []byte),
		generateHistory: make([]*Log, 0),
		absorbHistory:   make([]*Log, 0),
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
				s.recordAbs(by)
				po("upstream absorb sees '%s'", string(by))
				seen = append(seen, by...)
			case <-time.After(genDelay):
				select {
				case s.Generate <- ng:
					s.recordGen(ng)
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
