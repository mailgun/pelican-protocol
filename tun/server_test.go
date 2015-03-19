package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestServerSideWebSiteMockStartsUp004(t *testing.T) {
	cv.Convey("When we start a web server on the server side, we should be able to reach it with an http request", t, func() {

		web, err := NewWebServer(WebServerConfig{ReadTimeout: specialFastTestReadTimeout}, nil)
		panicOn(err)
		web.Start("most-downstream-target")
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

func TestReverseProxyToUltimateWebServerMock005(t *testing.T) {

	// setup a mock web server that replies to ping with pong.
	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	web, err := NewWebServer(WebServerConfig{ReadTimeout: specialFastTestReadTimeout}, mux)
	panicOn(err)
	web.Start("ultimate-webserver-mock")
	defer web.Stop()

	if !PortIsBound(web.Cfg.Listen.IpPort) {
		panic("web server did not come up")
	}

	// start a reverse proxy and verify that connections
	// reach the web server.
	rev := NewReverseProxy(ReverseProxyConfig{Dest: web.Cfg.Listen})
	rev.Start()
	defer rev.Stop()

	if !PortIsBound(rev.Cfg.Listen.IpPort) {
		panic("reverse proxy server did not come up")
	}

	cv.Convey("The PelicanReverseProxy should pass requests downstream to the ultimate webserver\n", t, func() {

		tunnel := NewLongPoller(web.Cfg.Listen)
		err := tunnel.Start()
		cv.So(err, cv.ShouldEqual, nil)
		defer tunnel.Stop()
		rev.createQueue <- tunnel

		body := []byte(`GET /ping HTTP/1.1
Host: 127.0.0.1:54284
User-Agent: Go 1.1 package http
Accept-Encoding: gzip

`)

		mockRw := &MockResponseWriter{}
		mockReq, err := http.NewRequest("GET", "/ping", bytes.NewBuffer(body))
		if err != nil {
			panic(err)
		}
		reply, err := rev.injectPacket(mockRw, mockReq, body, tunnel.key)
		cv.So(err, cv.ShouldEqual, nil)
		po("reply = '%s'", string(reply))
		cv.So(strings.HasPrefix(string(reply), `HTTP/1.1 200 OK`), cv.ShouldEqual, true)
		cv.So(strings.Contains(string(reply), `Content-Length: 4`), cv.ShouldEqual, true)
		cv.So(strings.HasSuffix(string(reply), "pong"), cv.ShouldEqual, true)
	})

	fmt.Printf("\n done with TestReverseProxyToUltimateWebServerMock005\n")
}
