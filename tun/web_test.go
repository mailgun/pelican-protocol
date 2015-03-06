package pelicantun_test

import (
	"fmt"
	"net/http"
	"testing"

	tun "github.com/mailgun/pelican-protocol/tun"
	cv "github.com/smartystreets/goconvey/convey"
)

func TestWebServer888(t *testing.T) {

	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	s := tun.NewWebServer(tun.WebServerConfig{}, mux)
	s.Start()
	defer s.Stop()

	cv.Convey("NewWebServer followed by Start() should bring up a web-server", t, func() {
		cv.So(tun.PortIsBound(s.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		by, err := tun.FetchUrl("http://" + s.Cfg.Listen.IpPort + "/ping")
		cv.So(err, cv.ShouldEqual, nil)
		//fmt.Printf("by:'%s'\n", string(by))
		cv.So(string(by), cv.ShouldEqual, "pong")

		fmt.Printf("\n       and Stop() should halt the web server.\n")
		s.Stop()
		cv.So(tun.PortIsBound(s.Cfg.Listen.IpPort), cv.ShouldEqual, false)
	})

}
