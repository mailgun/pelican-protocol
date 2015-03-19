package main

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestReuseSockets011(t *testing.T) {

	web, rev, fwd, err := StartTestSystemWithPing()
	panicOn(err)
	defer web.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	cv.Convey("Given a ForwardProxy and a ReverseProxy communicating over http, in order to acheive low-latency sends, the sockets should be re-used by http-pipelining.", t, func() {

		po("\n fetching url from %v\n", fwd.Cfg.Listen.IpPort)

		by, err := FetchUrl("http://" + fwd.Cfg.Listen.IpPort + "/ping")
		cv.So(err, cv.ShouldEqual, nil)
		//fmt.Printf("by:'%s'\n", string(by))
		cv.So(string(by), cv.ShouldEqual, "pong")
	})

	fmt.Printf("Given a Forward and Reverse proxy, in order to avoid creating new sockets too often (expensive), we should re-use the existing sockets for up to 5 round trips in 30 seconds.")
}
