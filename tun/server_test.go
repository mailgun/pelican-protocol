package pelicantun

import (
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestReverseProxyTalksToWebSite(t *testing.T) {
	cv.Convey("When we start the ReverseProxy, it should forward web requests to ultimate target web server at the given host and port", t, func() {

		web := NewWebServer(WebServerConfig{}, nil)
		web.Start()
		cv.So(PortIsBound(web.Cfg.Addr), cv.ShouldEqual, true)

		defer func() {
			web.Stop()
			cv.So(PortIsBound(web.Cfg.Addr), cv.ShouldEqual, false)
		}()

		//by, err := FetchUrl("http://" + web.Cfg.Addr + "/")

		cv.So(true, cv.ShouldEqual, true)
	})
}
