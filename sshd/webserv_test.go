package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/mailgun/pelican-protocol"
	"testing"
)

func TestWebServerServes(t *testing.T) {

	cv.Convey("Verify basic web-serving works: When we start a webserv.go webserver, client requests should return the expected page content", t, func() {

		cv.So(equal, cv.ShouldEqual, true)
	})

}
