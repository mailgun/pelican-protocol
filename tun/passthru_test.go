package main

/* not sure if we want to do passthrough at this level or ealier?

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPassthroughToNonPelican011(t *testing.T) {

	web, rev, fwd, err := StartTestSystemWithPing()
	panicOn(err)
	defer web.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	cv.Convey("Given a ReverseProxy listening on http port, in order to serve clients that aren't speaking Pelican protocol, we need to fall back to serving from the regular webserver if its not Pelican protocol being spoken. This will be indicated by checking that the key represents a Pelican Protocol Key (which is signed in a recognizable format; but appears random)", t, func() {

		po("\n fetching url from %v\n", fwd.Cfg.Listen.IpPort)

		by, err := FetchUrl("http://" + fwd.Cfg.Listen.IpPort + "/ping")
		cv.So(err, cv.ShouldEqual, nil)
		//fmt.Printf("by:'%s'\n", string(by))
		cv.So(string(by), cv.ShouldEqual, "pong")

		po("\n fetching url from rev directly, without supplying a key: %s\n", rev.Cfg.Listen.Ip)

		by, err = FetchUrl("http://" + rev.Cfg.Listen.IpPort + "/ping")
		cv.So(err, cv.ShouldEqual, nil)
		//fmt.Printf("by:'%s'\n", string(by))
		cv.So(string(by), cv.ShouldEqual, "pong")
	})

}
*/
