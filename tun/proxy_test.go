package pelicantun

import (
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestSocksProxyTalksToReverseProxy(t *testing.T) {
	cv.Convey("Given a ForwardProxy and a ReverseProxy, they should communicate over http", t, func() {

		rev := NewRevServer(RevServerConfig{})
		rev.Start()
		cv.So(PortIsBound(rev.Cfg.Addr), cv.ShouldEqual, true)

		defer func() {
			rev.Stop()
			cv.So(PortIsBound(rev.Cfg.Addr), cv.ShouldEqual, false)
		}()

		fwd := NewFwdServer(FwdServerConfig{})
		fwd.Start()
		cv.So(PortIsBound(fwd.Cfg.Addr), cv.ShouldEqual, true)

		defer func() {
			fwd.Stop()
			cv.So(PortIsBound(fwd.Cfg.Addr), cv.ShouldEqual, false)
		}()

		by, err := FetchUrl("http://" + rev.Cfg.Addr + "/")

		cv.So(true, cv.ShouldEqual, true)
	})
}
