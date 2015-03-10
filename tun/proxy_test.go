package pelicantun

import (
	"fmt"
	"net/http"
	"testing"

	//cv "github.com/glycerine/goconvey/convey"
)

func TestFullRoundtripSocksProxyTalksToReverseProxy002(t *testing.T) {

	// setup a mock web server that replies to ping with pong.
	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	web := NewWebServer(WebServerConfig{}, mux)
	web.Start()
	defer web.Stop()

	if !PortIsBound(web.Cfg.Listen.IpPort) {
		panic("web server did not come up")
	}

	// start a reverse proxy
	rev := NewReverseProxy(ReverseProxyConfig{Dest: web.Cfg.Listen})
	rev.Start()
	defer rev.Stop()

	if !PortIsBound(rev.Cfg.Listen.IpPort) {
		panic("rev proxy not up")
	}

	// start the forward proxy, talks to the reverse proxy.
	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: rev.Cfg.Listen,
	})
	fwd.Start()
	//defer fwd.Stop()
	if !PortIsBound(fwd.Cfg.Listen.IpPort) {
		panic("fwd proxy not up")
	}
	fwd.Stop()
	/*
		cv.Convey("Given a ForwardProxy and a ReverseProxy, they should communicate over http", t, func() {

			po("\n fetching url from %v\n", fwd.Cfg.Listen.IpPort)

			by, err := FetchUrl("http://" + fwd.Cfg.Listen.IpPort + "/ping")
			cv.So(err, cv.ShouldEqual, nil)
			//fmt.Printf("by:'%s'\n", string(by))
			cv.So(string(by), cv.ShouldEqual, "pong")
		})
	*/
	fmt.Printf("\n done with TestSocksProxyTalksToReverseProxy002()\n")
}
