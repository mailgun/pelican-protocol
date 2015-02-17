package main

import (
	"os"
	"os/signal"
)

// The PelicanClient is local to the web browser and
// speaks pelican-protocol over to the pelican server
// that runs on the web server host. The PelicanClient
// acts as a Socks-proxy and as a key manager. The
// Pelican Server acts as a reverse proxy.
//
// PelicanClient shuts down in response to ctrl-c or SIGINT.
type PelicanClient struct {
	ReqStop chan bool
	Done    chan bool
	Ready   chan bool
	CtrlC   chan os.Signal
}

func NewPelicanClient() *PelicanClient {
	p := &PelicanClient{
		CtrlC:   make(chan os.Signal, 1),
		Done:    make(chan bool),
		ReqStop: make(chan bool),
		Ready:   make(chan bool),
	}
	signal.Notify(p.CtrlC, os.Interrupt)
	return p
}

func (s *PelicanClient) IsStopped() bool {
	select {
	case <-s.Done:
		return true
	default:
		return false
	}
	return false
}

func (s *PelicanClient) Stop() {
	// avoid double closing ReqStop here
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	<-s.Done
}

func (s *PelicanClient) Start() {
	go func() {
		close(s.Ready)
		for {
			select {
			case <-s.CtrlC:
				close(s.Done)
				return
			case <-s.ReqStop:
				close(s.Done)
				return
			}
		}
	}()
}
