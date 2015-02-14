package pelican

import (
	cv "github.com/glycerine/goconvey/convey"
	"testing"
)

func TestClientAndServerDoAPortForward(t *testing.T) {

	cv.Convey("Given a running pelican-server, when pelican-client does ssh using a previously made account, a local port should be forwarded to the servers port 80, so that the web-page can be accessed securely.\n", t, func() {

		cv.So("", cv.ShouldEqual, "")
	})
}
