package pelicantun

import (
	"fmt"
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestPelicanSocksProxyAcceptsClientConnections003(t *testing.T) {
	cv.Convey("When we start the PelicanSocksProxy, it should accept connections on the configured port from web browsers\n", t, func() {

		psp := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
		psp.Start()
		defer psp.Stop()

		cv.So(PortIsBound(psp.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		url := fmt.Sprintf("http://%s/hello/world", psp.Cfg.Listen.IpPort)
		_, err := FetchUrl(url)

		lastUrl, err := psp.LastRemote()
		cv.So(err, cv.ShouldEqual, nil)
		host, err := SplitOutHostFromUrl(lastUrl.String())
		cv.So(err, cv.ShouldEqual, nil)
		cv.So(host, cv.ShouldEqual, "127.0.0.1")
	})
}
