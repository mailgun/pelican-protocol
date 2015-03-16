package pelicantun

import (
	"net"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPelicanSocksProxyHandlesClientConnectionClose008(t *testing.T) {

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
