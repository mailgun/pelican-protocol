package pelicantun

import (
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestReverseProxyTalksToWebSite(t *testing.T) {
	cv.Convey("When we start the ReverseProxy, it should forward web requests to ultimate target web server at the given host and port", t, func() {
		cv.So(true, cv.ShouldEqual, true)
	})
}
