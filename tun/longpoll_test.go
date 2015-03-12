package pelicantun

import (
	"fmt"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestLongPollToGetLowLatency010(t *testing.T) {

	cli, srv, rev, fwd, err := StartTestSystemWithBcast()
	panicOn(err)
	defer srv.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	cv.Convey("Given a ForwardProxy and a ReverseProxy communicating over http, in order to acheive low-latency sends from server to client, long-polling with two sockets (one send and one receive) should be used. So a server that has something to send (e.g. broadcast here) should be able to send back to the client immediately.", t, func() {

		cli.Start()
		defer cli.Stop()
		<-srv.FirstClient

		msg := "BREAKING NEWS"
		srv.Bcast(msg)

		// wait for message to get from server to client.
		select {
		case <-cli.MsgRecvd:
			// excllent
		case <-time.After(time.Second * 2):
			panic("should have gotten a message from the server at the client by now!")
		}

		cv.So(cli.LastMsgReceived(), cv.ShouldEqual, msg)
	})

	fmt.Printf("Given a Forward and Reverse proxy, in order to avoid creating new sockets too often (expensive), we should re-use the existing sockets for up to 5 round trips in 30 seconds.")
}
