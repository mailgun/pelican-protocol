package pelicantun

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestLongPollToGetLowLatency010(t *testing.T) {

	cli, bcast, rev, fwd, err := StartTestSystemWithBcast()
	panicOn(err)
	defer bcast.Stop()
	defer rev.Stop()
	defer fwd.Stop()
	defer cli.Stop()

	cv.Convey("Given a ForwardProxy and a ReverseProxy communicating over http, in order to acheive low-latency sends from server to client, long-polling with two sockets (one send and one receive) should be used. So a server that has something to send (e.g. broadcast here) should be able to send back to the client immediately.", t, func() {

		msg := "BREAKING NEWS"
		bcast.Broadcast(msg)
		cv.So(cli.LastMsgReceived(), cv.ShouldEqual, msg)
	})

	fmt.Printf("Given a Forward and Reverse proxy, in order to avoid creating new sockets too often (expensive), we should re-use the existing sockets for up to 5 round trips in 30 seconds.")
}
