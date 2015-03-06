package pelicantun

import (
	"fmt"
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestSocksProxyTalksToReverseProxy002(t *testing.T) {

	fmt.Printf("\n before NewReverseProxy\n")
	rev := NewReverseProxy(ReverseProxyConfig{})
	fmt.Printf("\n done with NewReverseProxy\n")
	rev.Start()
	fmt.Printf("\n done with rev.Start()\n")

	cv.Convey("\n Given a ForwardProxy and a ReverseProxy, they should communicate over http\n\n", t, func() {

		cv.So(PortIsBound(rev.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		defer func() {
			rev.Stop()
			cv.So(PortIsBound(rev.Cfg.Listen.IpPort), cv.ShouldEqual, false)
		}()

		fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
		fwd.Start()
		cv.So(PortIsBound(fwd.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		defer func() {
			fwd.Stop()
			cv.So(PortIsBound(fwd.Cfg.Listen.IpPort), cv.ShouldEqual, false)
		}()

		by, err := FetchUrl("http://" + rev.Cfg.Listen.IpPort + "/")

		cv.So(err, cv.ShouldEqual, nil)
		cv.So(by, cv.ShouldResemble, []byte("some output"))
	})
}
