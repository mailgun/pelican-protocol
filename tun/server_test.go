package pelicantun

import (
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestServerSideWebSiteMockStartsUp004(t *testing.T) {
	cv.Convey("When we start a web server on the server side, we should be able to reach it with an http request", t, func() {

		web := NewWebServer(WebServerConfig{}, nil)
		web.Start()
		cv.So(PortIsBound(web.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		defer func() {
			web.Stop()
			cv.So(PortIsBound(web.Cfg.Listen.IpPort), cv.ShouldEqual, false)
		}()

		by, err := FetchUrl("http://" + web.Cfg.Listen.IpPort + "/")

		cv.So(err, cv.ShouldEqual, nil)
		cv.So(string(by), cv.ShouldResemble, "404 page not found\n")
	})
}
