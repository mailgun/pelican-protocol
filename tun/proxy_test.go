package main

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestFullRoundtripSocksProxyTalksToReverseProxy002(t *testing.T) {

	web, rev, fwd, err := StartTestSystemWithPing()
	panicOn(err)
	defer web.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	cv.Convey("Given a ForwardProxy and a ReverseProxy, they should communicate over http", t, func() {

		po("\n fetching url from %v\n", fwd.Cfg.Listen.IpPort)

		by, err := FetchUrl("http://" + fwd.Cfg.Listen.IpPort + "/ping")
		cv.So(err, cv.ShouldEqual, nil)
		//fmt.Printf("by:'%s'\n", string(by))
		cv.So(string(by), cv.ShouldEqual, "pong")
	})
	fmt.Printf("\n done with TestSocksProxyTalksToReverseProxy002()\n")
}
