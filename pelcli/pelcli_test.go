package main

import (
	cv "github.com/glycerine/goconvey/convey"
	"syscall"
	"testing"
)

func TestPelcliShutdown(t *testing.T) {

	cv.Convey("Given a running Pelcli, when it receives SIGINT it should shutdown cleanly.", t, func() {
		p := NewPelicanClient()
		p.Start()
		<-p.Ready

		// should get handled just like close(p.ReqStop)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)

		<-p.Done
		cv.So(p.IsStopped(), cv.ShouldEqual, true)
	})
}
