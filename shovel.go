package pelican

import (
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
// channel just before starting the io.Copy. The
// label parameter allows reporting on when a specific shovel
// was shut down.
func (s *Shovel) Start(w io.WriteCloser, r io.ReadCloser, label string) {
	go func() {
		var err error
		var n int64
		defer func() {
			close(s.Done)
			VPrintf("\n shovel %s copied %d bytes before shutting down\n", label, n)
		}()
		close(s.Ready)
		n, err = io.Copy(w, r)
		if err != nil {
			// don't freak out, the network connection got closed most likely.
			// e.g. read tcp 127.0.0.1:33631: use of closed network connection
			//panic(fmt.Sprintf("in Shovel '%s', io.Copy failed: %v\n", label, err))
			return
		}
	}()
	go func() {
		<-s.ReqStop
		r.Close() // causes io.Copy to finish
		w.Close()
	}()
}

// stop the shovel goroutine. returns only once the goroutine is done.
func (s *Shovel) Stop() {
	// avoid double closing ReqStop here
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	<-s.Done
}

// a ShovelPair manages the forwarding of a bidirectional
// channel, such as that in forwarding an ssh connection.
type ShovelPair struct {
	AB      *Shovel
	BA      *Shovel
	Done    chan bool
	ReqStop chan bool
	Ready   chan bool
}

// make a new ShovelPair
func NewShovelPair() *ShovelPair {
	return &ShovelPair{
		AB:      NewShovel(),
		BA:      NewShovel(),
		Done:    make(chan bool),
		ReqStop: make(chan bool),
		Ready:   make(chan bool),
	}
}

// Start the pair of shovels. ab_label will label the a<-b shovel. ba_label will
// label the b<-a shovel.
func (s *ShovelPair) Start(a io.ReadWriteCloser, b io.ReadWriteCloser, ab_label string, ba_label string) {
	s.AB.Start(a, b, ab_label)
	<-s.AB.Ready
	s.BA.Start(b, a, ba_label)
	<-s.BA.Ready
	close(s.Ready)

	// if one stops, shut down the other
	go func() {
		select {
		case <-s.AB.Done:
			s.BA.Stop()
		case <-s.BA.Done:
			s.AB.Stop()
		}
	}()
}

func (s *ShovelPair) Stop() {
	s.AB.Stop()
	s.BA.Stop()
}
