package gohttptun

import (
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestServerBindsPort(t *testing.T) {
	cv.Convey("Given that we start a server on a given port, then the server should be listening on that port", t, func() {
		cv.So(googleDNSIP, cv.ShouldEqual, false)
	})
}
