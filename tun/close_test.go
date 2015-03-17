package pelicantun

import (
	"net"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func Test_RW_EOF_on_reader_does_not_close_writer_0061(t *testing.T) {
	/*
		web, rev, fwd, err := StartTestSystemWithPing()
		panicOn(err)
		defer web.Stop()
		defer rev.Stop()
		defer fwd.Stop()

		conn, err := net.Dial("tcp", fwd.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}
	*/
	cv.Convey("given an operating tunnel from cli -> fwd -> rev -> srv, we should close the fwd and rev net.Conn only once we've gotten EOF on both the read from the srv and the read from the cli", t, func() {

	})

}

func TestPelicanSocksProxyHandlesClientConnectionClose006(t *testing.T) {

	web, rev, fwd, err := StartTestSystemWithPing()
	panicOn(err)
	defer web.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	//	psp := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
	//	psp.testonly_dont_contact_downstream = true
	//	psp.Start()
	//	defer psp.Stop()

	cv.Convey("Given a started PelicanSocksProxy, we should handle Close of the socket after Accept gracefully\n", t, func() {

		po("\n\n at the start of the close_test, we should not have any clients registered, right??: %d\n\n", len(fwd.chasers))
		err := fwd.WaitForClientCount(0, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		cv.So(fwd.CountOfOpenClients(), cv.ShouldEqual, 0)

		conn, err := net.Dial("tcp", fwd.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		err = fwd.WaitForClientCount(1, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		// and another
		conn2, err := net.Dial("tcp", fwd.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		err = fwd.WaitForClientCount(2, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		conn.Close()

		err = fwd.WaitForClientCount(1, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		conn2.Close()

		err = fwd.WaitForClientCount(0, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

	})
}

func TestPelicanSocksProxyHandlesClientConnectionCloseSuperSimple020(t *testing.T) {

	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
	fwd.testonly_dont_contact_downstream = true
	fwd.Start()
	defer fwd.Stop()

	cv.Convey("Given a started PelicanSocksProxy, we should handle Close of the socket after Accept gracefully\n", t, func() {

		po("\n\n at the start of the close_test, we should not have any clients registered, right??: %d\n\n", len(fwd.chasers))

		err := fwd.WaitForClientCount(0, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		cv.So(fwd.CountOfOpenClients(), cv.ShouldEqual, 0)

		conn, err := net.Dial("tcp", fwd.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		err = fwd.WaitForClientCount(1, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		// and another
		conn2, err := net.Dial("tcp", fwd.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		err = fwd.WaitForClientCount(2, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		conn.Close()

		err = fwd.WaitForClientCount(1, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		conn2.Close()

		err = fwd.WaitForClientCount(0, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

	})
}
