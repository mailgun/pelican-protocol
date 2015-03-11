package pelicantun

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestWebServer888(t *testing.T) {

	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	s, err := NewWebServer(WebServerConfig{}, mux, specialFastTestReadTimeout)
	panicOn(err)
	s.Start()
	defer s.Stop()

	cv.Convey("NewWebServer followed by Start() should bring up a web-server", t, func() {
		cv.So(PortIsBound(s.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		by, err := FetchUrl("http://" + s.Cfg.Listen.IpPort + "/ping")
		cv.So(err, cv.ShouldEqual, nil)
		//fmt.Printf("by:'%s'\n", string(by))
		cv.So(string(by), cv.ShouldEqual, "pong")

		fmt.Printf("\n       and Stop() should halt the web server.\n")
		s.Stop()
		cv.So(PortIsBound(s.Cfg.Listen.IpPort), cv.ShouldEqual, false)
	})

}

func TestWebServerPortAlreadyTakenDetected801(t *testing.T) {

	s, err := NewWebServer(WebServerConfig{}, nil, specialFastTestReadTimeout)
	panicOn(err)
	s.Start()
	defer s.Stop()

	cv.Convey("NewWebServer on a port that is already taken should return an error\n", t,
		func() {

			cv.So(PortIsBound(s.Cfg.Listen.IpPort), cv.ShouldEqual, true)

			_, err := NewWebServer(WebServerConfig{Listen: s.Cfg.Listen}, nil, specialFastTestReadTimeout)
			cv.So(err, cv.ShouldNotEqual, nil)
			fmt.Printf("err = '%s'\n", err)
			cv.So(strings.HasPrefix(err.Error(), "NewWebServer error: could not start because port already in-use"), cv.ShouldEqual, true)
		})

}
