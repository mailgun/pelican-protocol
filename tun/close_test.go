package pelicantun

import (
	"net"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPelicanSocksProxyHandlesClientConnectionClose008(t *testing.T) {
	cv.Convey("Given a started PelicanSocksProxy, we should handle Close of the socket after Accept gracefully\n", t, func() {

		psp := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
		psp.Start()
		defer psp.Stop()

		cv.So(psp.OpenClientCount(), cv.ShouldEqual, 0)

		conn, err := net.Dial("tcp", psp.Cfg.Listen.IpPort)
		if err != nil {
			panic(err)
		}
		cv.So(psp.OpenClientCount(), cv.ShouldEqual, 1)

		conn.Close()
		cv.So(psp.OpenClientCount(), cv.ShouldEqual, 0)

	})
}
