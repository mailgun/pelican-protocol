package pelicantun

import (
	cv "github.com/glycerine/goconvey/convey"
	"testing"
	"time"
)

func TestDroppedConnectionsCloseBothEnds022(t *testing.T) {

	srv := NewBcastServer(Addr{})
	srv.Start()
	defer srv.Stop()

	rev := NewReverseProxy(ReverseProxyConfig{Dest: srv.Listen})
	rev.Start()
	defer rev.Stop()

	// whichever end does the active close on the tcp socket is the end that ends up in TIME_WAIT.
	// Hence: on the fwd side, Chaser.rw that manages the connection upstream should never initiate active close.
	// And: on the rev side, the LongPoller

	cv.Convey("Given a reverse proxy or a forward proxy, if we are hit with a IsPortBound()/WaitUntilServerUp() client that opens and closes the connection immedidately, we should waste no resources after the IsPortBound() closes without sending any data. PelicanSocksProxy::Start() does this to itself.", t, func() {

	})
}

// Drops are similar to the close_test.go, actually.
func TestDroppedConnectionsToForwardDontLeak023(t *testing.T) {

	srv := NewBcastServer(Addr{})
	srv.Start()
	defer srv.Stop()

	rev := NewReverseProxy(ReverseProxyConfig{
		Dest: srv.Listen,
	})
	rev.Start()
	defer rev.Stop()

	// rev test, not fwd !!
	//	if !PortIsBound(rev.Cfg.Listen.IpPort) {
	//		panic("rev proxy not up")
	//	}

	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: rev.Cfg.Listen,
	})
	fwd.Start()
	defer fwd.Stop()

	// this check causes things to hang!!! on the tunnel.go until longPollTimeUp!! (30 seconds)
	//
	if !PortIsBound(fwd.Cfg.Listen.IpPort) {
		panic("fwd proxy not up")
	}

	// start broadcast client (to test receipt of long-polled data from server)
	cli := NewBcastClient(Addr{Port: fwd.Cfg.Listen.Port})

	// doubles as a pause for system to come into equilibrium

	PollExpecting("fwd.CountOfOpenClients() == 0", func() bool { return fwd.CountOfOpenClients() == 0 }, time.Second*2)
	/*
		// same as:
		err := fwd.WaitForClientCount(0, time.Second*2)
		panicOn(err)
	*/

	cv.Convey("Given a forward proxy, doing PortIsBound() --a client that opens and closes the connection immedidately, we should waste no resources after the IsPortBound()", t, func() {

		cv.So(PollExpecting("fwd.CountOfOpenClients() == 0", func() bool { return fwd.CountOfOpenClients() == 0 }, time.Second*2), cv.ShouldEqual, true)
		/*
			if !PortIsBound(fwd.Cfg.Listen.IpPort) {
				panic("fwd proxy not up")
			}
			cv.So(PollExpecting("fwd.CountOfOpenClients() == 0", func() bool { return fwd.CountOfOpenClients() == 0 }, time.Second*2), cv.ShouldEqual, true)

			if !PortIsBound(fwd.Cfg.Listen.IpPort) {
				panic("fwd proxy not up")
			}

			cv.So(PollExpecting("fwd.CountOfOpenClients() == 0", func() bool { return fwd.CountOfOpenClients() == 0 }, time.Second*2), cv.ShouldEqual, true)
		*/

		cli.Start()
		defer cli.Stop()

		<-srv.SecondClient
		po("got past <-srv.SecondClient\n")

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

}
