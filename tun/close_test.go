package pelicantun

import (
	"net"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPelicanSocksProxyHandlesClientConnectionClose008(t *testing.T) {

	psp := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
	psp.testonly_dont_contact_downstream = true
	psp.Start()
	defer psp.Stop()

	//	if psp.doneAlarm != nil {
	//		panic("client psp logic error: doneAlarm must be nil until we arm it.")
	//	}

	cv.Convey("Given a started PelicanSocksProxy, we should handle Close of the socket after Accept gracefully\n", t, func() {

		err := psp.WaitForClientCount(0, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		cv.So(psp.OpenClientCount(), cv.ShouldEqual, 0)

		conn, err := net.Dial("tcp", psp.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		err = psp.WaitForClientCount(1, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		// and another
		conn2, err := net.Dial("tcp", psp.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		err = psp.WaitForClientCount(2, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		conn.Close()

		err = psp.WaitForClientCount(1, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

		conn2.Close()

		err = psp.WaitForClientCount(0, time.Second*2)
		cv.So(err, cv.ShouldEqual, nil)

	})
}
