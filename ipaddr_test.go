package pelican

import (
	cv "github.com/glycerine/goconvey/convey"
	"testing"
)

func TestRoutableCheckWorks(t *testing.T) {
	cv.Convey("unroutable IPs should be correctly detected", t, func() {
		cv.So(IsRoutableIPv4("127.0.0.1"), cv.ShouldEqual, false)
		cv.So(IsRoutableIPv4("10.0.0.2"), cv.ShouldEqual, false)
		cv.So(IsRoutableIPv4("192.168.0.2"), cv.ShouldEqual, false)
		cv.So(IsRoutableIPv4("191.191.191.191"), cv.ShouldEqual, true)
		cv.So(IsRoutableIPv4("8.8.8.8"), cv.ShouldEqual, true)
	})
}
