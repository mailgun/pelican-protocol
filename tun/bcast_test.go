package main

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

//////////// test our simple client and server can talk

func TestTcpClientAndServerCanTalkDirectly012(t *testing.T) {

	// start broadcast server
	srv := NewBcastServer(Addr{})
	srv.Start()
	po("\n done with srv.Start()\n")

	// start broadcast client
	cli := NewBcastClient(Addr{Port: srv.Listen.Port})
	cli.Start()

	// let the client hook up to the server first, or else we'll race.
	<-srv.FirstClient
	po("\n done with cli.Start()\n")

	cv.Convey("The broadcast client and server should be able to speak directly without proxies, sending and receiving", t, func() {

		msg := "BREAKING NEWS"
		po("\n about to srv.Bcast()\n")
		srv.Bcast(msg)

		found := cli.Expect(msg)

		cv.So(found, cv.ShouldEqual, true)
	})

	fmt.Printf("Given a Forward and Reverse proxy, in order to avoid creating new sockets too often (expensive), we should re-use the existing sockets for up to 5 round trips in 30 seconds.")
}

// not a real test, just answering a question about what happens on socket close:
// the go code sees an "EOF" error.
/*
// when the client shutsdown the server gets EOF. But does EOF always mean
// the socket is closed??
func TestTcpClientAndServerGetWhatErrorOnDisconnectVsEndOfSending999(t *testing.T) {

	// start broadcast server
	srv := NewBcastServer(Addr{})
	srv.Start()
	po("\n done with srv.Start()\n")

	// start broadcast client
	cli := NewBcastClient(Addr{Port: srv.Listen.Port})
	cli.Start()

	// let the client hook up to the server first, or else we'll race.
	<-srv.FirstClient
	po("\n done with cli.Start()\n")

	cv.Convey("When the broadcast client closes the connection, the server should see EOF", t, func() {

		// client sees err.Error == "EOF" when server closes connection.
		//srv.CloseClientConnections()

		cli.Stop()

		msg := "BREAKING NEWS"
		po("\n about to srv.Bcast()\n")
		srv.Bcast(msg)
		po("\n done with srv.Bcast()\n")

		found := cli.Expect(msg)

		cv.So(found, cv.ShouldEqual, true)
	})

	cv.Convey("When the broadcast server closes the connection, the client should see EOF", t, func() {

	})
}
*/
