package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/mailgun/pelican-protocol"
	//"net/http"
	"testing"
)

func TestWebServerServes(t *testing.T) {

	port := pelican.GetAvailPort()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	w := NewWebServer(addr, nil)
	w.Start()
	defer w.Stop()
	fmt.Printf("\n web server listing on http://127.0.0.1:%d'\n", port)

	cv.Convey("Verify basic web-serving works: When we start a webserv.go webserver, client requests should return the expected page content", t, func() {
		page := MyCurl(fmt.Sprintf("http://%s", addr))
		cv.So(page, cv.ShouldContainSubstring, "[This is the main static body.]")
	})
}
