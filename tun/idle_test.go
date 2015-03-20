package pelicantun

import (
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

// next test to write, after 016:
// + growing number of bytes in messages get returned.
// + client sends 2, server sends 3, client sends 4, server sends 5 msgs in a row.
//
// + if there are no bytes to send, then the client and the server should remain idle
//   until the long-poll timeout (30 sec by default, can be smaller for testing).
//
func TestIdleLongPollerShouldNotBeChatty050(t *testing.T) {

	cli, srv, rev, fwd, err := StartTestSystemWithCountingServer()
	panicOn(err)

	cv.Convey("Test ordering discipline for long-polling: Given a ForwardProxy and a ReverseProxy communicating over http, in order to acheive low-latency sends from server to client, long-polling with two sockets (one send and one receive) should be used. The two long polling sockets should still observe a first-to-send is first-to-get-a-reply order discipline.\n", t, func() {
		cli.Start()
		<-srv.FirstCountingClientSeen

		po("got past <-srv.FirstCountingClientClient\n")

		// fetch 1 message, then expect quiet on the network
		N := 1
		for i := 0; i < N; i++ {
			select {
			case msg := <-cli.MsgRecvd:
				// excellent
				po("CountingTestClient got a message! client saw: '%s'", msg)
			case <-time.After(time.Second * 2):
				po("CountingTestClient: We waited 2 seconds, bailing.")
				// should have gotten a message from the server at the client by now.

				panic("should have gotten a message from the server at the client by now!")
			}
		}

		po("TestLongPollKeepsFifoOrdering016: done with %d roundtrips", N)

		po("client history:")
		cli.ShowTmHistory()

		po("server history:")
		srv.ShowTmHistory()

		time.Sleep(10 * time.Second)

		po("client history after 10 sec sleep:")
		cli.ShowTmHistory()

		po("server history after 10 sec sleep:")
		srv.ShowTmHistory()

	})

	srv.Stop()
	rev.Stop()
	fwd.Stop()
	cli.Stop()
}
