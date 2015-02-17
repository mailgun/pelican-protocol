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
			fmt.Printf("\n done with shovel goroutine.\n")
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
