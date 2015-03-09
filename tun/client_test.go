package pelicantun

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPelicanSocksProxyAcceptsClientConnections003(t *testing.T) {
	cv.Convey("When we start the PelicanSocksProxy, it should accept connections on the configured port from web browsers\n", t, func() {

		psp := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
		psp.Start()

		po("003 after psp.Start\n")

		defer psp.Stop()

		// the PortIsBound call will cause a connection that should then be closed.
		cv.So(PortIsBound(psp.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		url := fmt.Sprintf("http://%s/hello/world", psp.Cfg.Listen.IpPort)
		_, err := FetchUrl(url)

		po("003 after FetchUrl\n")

		lastUrl, err := psp.LastRemote()

		po("003 after psp.LastRemote\n")

		cv.So(err, cv.ShouldEqual, nil)
		host, err := SplitOutHostFromUrl(lastUrl.String())
		cv.So(err, cv.ShouldEqual, nil)
		cv.So(host, cv.ShouldEqual, "127.0.0.1")

		po("done with 003\n")
	})
}
