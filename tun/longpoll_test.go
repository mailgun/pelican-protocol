package pelicantun

//	cv "github.com/glycerine/goconvey/convey"

/*
func TestLongPollToGetLowLatency010(t *testing.T) {

	cli, srv, rev, fwd, err := StartTestSystemWithBcast()
	panicOn(err)
	defer srv.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	cv.Convey("Given a ForwardProxy and a ReverseProxy communicating over http, in order to acheive low-latency sends from server to client, long-polling with two sockets (one send and one receive) should be used. So a server that has something to send (e.g. broadcast here) should be able to send back to the client immediately.\n", t, func() {
		cli.Start()
		defer cli.Stop()
		<-srv.FirstClient

		po("got past <-srv.FirstClient\n")

		defer func() { po("deferred frunction running...!\n") }()

		msg := "BREAKING NEWS"
		srv.Bcast(msg)

		// wait for message to get from server to client.
		select {
		case <-cli.MsgRecvd:
			// excllent
			po("excellent, cli got a message!\n")
			po("client saw: '%s'", cli.LastMsgReceived())
		case <-time.After(time.Second * 2):
			po("\n\nWe waited 2 seconds, bailing.\n")
			srv.Stop()
			// should have gotten a message from the server at the client by now.

			panic("should have gotten a message from the server at the client by now!")
			cv.So(cli.LastMsgReceived(), cv.ShouldEqual, msg)
		}

		cv.So(cli.LastMsgReceived(), cv.ShouldEqual, msg)

	})

	//	fmt.Printf("Given a Forward and Reverse proxy, in order to avoid creating new sockets too often (expensive), we should re-use the existing sockets for up to 5 round trips in 30 seconds.")
}

/*
func TestLongPollKeepsFifoOrdering016(t *testing.T) {

	cli, srv, rev, fwd, err := StartTestSystemWithBcast()
	panicOn(err)
	defer srv.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	cv.Convey("Test ordering discipline for long-polling: Given a ForwardProxy and a ReverseProxy communicating over http, in order to acheive low-latency sends from server to client, long-polling with two sockets (one send and one receive) should be used. The two long polling sockets should still observe a first-to-send is first-to-get-a-reply order discipline.\n", t, func() {
		cli.Start()
		defer cli.Stop()
		<-srv.FirstClient

		po("got past <-srv.FirstClient\n")

		defer func() { po("deferred frunction running...!\n") }()

		msg := "BREAKING NEWS"
		srv.Bcast(msg)

		// wait for message to get from server to client.
		select {
		case <-cli.MsgRecvd:
			// excllent
			po("excellent, cli got a message!\n")
			po("client saw: '%s'", cli.LastMsgReceived())
		case <-time.After(time.Second * 2):
			po("\n\nWe waited 2 seconds, bailing.\n")
			srv.Stop()
			// should have gotten a message from the server at the client by now.

			panic("should have gotten a message from the server at the client by now!")
			cv.So(cli.LastMsgReceived(), cv.ShouldEqual, msg)
		}

		cv.So(cli.LastMsgReceived(), cv.ShouldEqual, msg)

	})

	//	fmt.Printf("Given a Forward and Reverse proxy, in order to avoid creating new sockets too often (expensive), we should re-use the existing sockets for up to 5 round trips in 30 seconds.")
}

*/
