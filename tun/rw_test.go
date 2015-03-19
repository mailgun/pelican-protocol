package main

import (
	"fmt"
	"net"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
	"github.com/glycerine/rbuf"
)

func TestRW017(t *testing.T) {

	cv.Convey("When we start a RW that turns a net.Conn connection into a pair of channels, then reads and writes to/from the netConn should succeed and notice if the connection is dropped.", t, func() {

		echo := NewEchoServer(Addr{Ip: "127.0.0.1"})
		echo.Start()
		defer echo.Stop()

		conn, err := net.Dial("tcp", echo.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		// use a small buffer to simulate needing multiple reads
		rw := NewServerRW(conn, 2, nil, nil)
		rw.Start()
		defer rw.Stop()

		m1 := []byte("yippeeee! yes!")
		rw.SendToDownCh() <- m1

		// accumulate reads into here
		circ := rbuf.NewFixedSizeRingBuf(200)

	for10msec:
		for {
			select {
			case m2 := <-rw.RecvFromDownCh(): //dnReadToUpWrite:
				circ.Write(m2)

				// 100 msec might not be long enough for all test circumstances.
			case <-time.After(1000 * time.Millisecond):
				break for10msec
			}
		}

		s1 := string(m1)
		m2 := circ.Bytes()
		s2 := string(m2)
		if s1 != s2 {
			po("m1 = '%s' while m2 = '%s'\n", s1, s2)
			panic("echo server not echoing")
		}

		po("m1 m2 compare looks good\n")

		circ.Reset()

		a1 := "more 0123"
		rw.SendToDownCh() <- []byte(a1)

	forb:
		for {
			select {
			case b := <-rw.RecvFromDownCh():
				circ.Write(b)

				// 100 msec might not be long enough for all test circumstances.
			case <-time.After(1000 * time.Millisecond):
				break forb
			}
		}

		b1 := string(circ.Bytes())

		cv.So(s1, cv.ShouldResemble, s2)
		cv.So(a1, cv.ShouldResemble, b1)

		// when the the server stops we should
		// get nil channels back that can't be read or sent on.
		rw.Stop()

		select {
		case b := <-rw.RecvFromDownCh():
			panic(fmt.Errorf("bad: even with echo stopped, we got read on dnReadToUpWrite!: '%s'", string(b)))
		case <-time.After(10 * time.Millisecond):
			cv.So(true, cv.ShouldEqual, true) // i.e. looks good, we should timeout.
		}

		// check that write stops
		doNotSend := []byte("do not send!")
		select {
		case rw.SendToDownCh() <- doNotSend:
			panic(fmt.Errorf("bad: even with echo stopped, we got write to upReadToDnWrite: '%s'", string(doNotSend)))
		case <-time.After(10 * time.Millisecond):
			cv.So(true, cv.ShouldEqual, true) // i.e. looks good, we should timeout.
		}

	})
}
