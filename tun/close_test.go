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
	if psp.doneAlarm != nil {
		panic("client psp logic error: doneAlarm must be nil until we arm it.")
	}
	po("topcount = %d\n", psp.GetTopLoopCount())

	cv.Convey("Given a started PelicanSocksProxy, we should handle Close of the socket after Accept gracefully\n", t, func() {

		cv.So(psp.OpenClientCount(), cv.ShouldEqual, 0)
		topcount := psp.GetTopLoopCount()
		po("sequential, topcount = %d\n", topcount)

		po("dialing %v\n", psp.Cfg.Listen.IpPort)
		conn, err := net.Dial("tcp", psp.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}

		var trycount int64 = 1
		c2 := psp.GetTopLoopCount()
		po("c2 = %d, topcount = %d\n", c2, topcount)
		for c2 < topcount-trycount+3 {
			time.Sleep(1000 * time.Millisecond)
			c2 = psp.GetTopLoopCount()
			trycount++
			po("c2 = %d, topcount-trycount = %d\n", c2, topcount-trycount)
		}
		cv.So(psp.OpenClientCount(), cv.ShouldEqual, 1) // failing

		po("topcount = %d\n", psp.GetTopLoopCount())

		// have to prevent race by waiting until the
		// effect of conn.Close() percolates through.
		ind2 := psp.GetDoneReaderIndicator()
		conn.Close()
		<-ind2

		cv.So(psp.OpenClientCount(), cv.ShouldEqual, 0)

		po("topcount = %d\n", psp.GetTopLoopCount())

	})
}
