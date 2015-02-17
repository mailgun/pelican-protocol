package pelican

import (
	"fmt"
	"io"
)

// Shovel shovels data from an io.ReadCloser to an io.WriteCloser
// in an independent go routine started by Shovel::Start().
// You can request that the shovel stop by closing ReqStop,
// and wait until Done is closed to know that it is finished.
type Shovel struct {
	Done    chan bool
	ReqStop chan bool
	Ready   chan bool
}

// make a new Shovel
func NewShovel() *Shovel {
	return &Shovel{
		Done:    make(chan bool),
		ReqStop: make(chan bool),
		Ready:   make(chan bool),
	}
}

// Start starts the shovel doing an io.Copy from r to w. The
// goroutine that is running the copy will close the Ready
// channel just before starting the io.Copy.
func (s *Shovel) Start(w io.WriteCloser, r io.ReadCloser) {
	go func() {
		defer func() {
			r.Close()
			w.Close()
			close(s.Done)
		}()
		close(s.Ready)
		_, err := io.Copy(w, r)
		if err != nil {
			fmt.Printf("shovel, io.Copy failed: %v\n", err)
			return
		}
	}()
	go func() {
		<-s.ReqStop
		fmt.Printf("\n ReqStop detected.\n")
		r.Close() // causes io.Copy to finish
	}()
}

// stop the shovel goroutine. returns only once the goroutine is done.
func (s *Shovel) Stop() {
	close(s.ReqStop)
	<-s.Done
}

// a ShovelPair manages the forwarding of a bidirectional
// channel, such as that in forwarding an ssh connection.
type ShovelPair struct {
	AB      *Shovel
	BA      *Shovel
	Done    chan bool
	ReqStop chan bool
}

// make a new ShovelPair
func NewShovelPair() *ShovelPair {
	return &ShovelPair{
		AB:      NewShovel(),
		BA:      NewShovel(),
		Done:    make(chan bool),
		ReqStop: make(chan bool),
	}
}

// Start the pair of shovels
func (s *ShovelPair) Start(a io.ReadWriteCloser, b io.ReadWriteCloser) {
	s.AB.Start(a, b)
	<-s.AB.Ready
	s.BA.Start(b, a)
	<-s.BA.Ready
}

func (s *ShovelPair) Stop() {
	s.AB.Stop()
	s.BA.Stop()
}
