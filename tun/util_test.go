package pelicantun

import (
	"fmt"
	"net/http"
	"time"
)

// test utilities

var specialFastTestReadTimeout time.Duration = 500 * time.Millisecond

/*
example use of StartTestSystemWithPing() to setup a test:

	web, rev, fwd, err := StartTestSystemWithPing()
	panicOn(err)
	defer web.Stop()
	defer rev.Stop()
	defer fwd.Stop()

*/
func StartTestSystemWithPing() (*WebServer, *ReverseProxy, *PelicanSocksProxy, error) {
	// setup a mock web server that replies to ping with pong.
	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	web := NewWebServer(WebServerConfig{}, mux, specialFastTestReadTimeout)
	web.Start()

	if !PortIsBound(web.Cfg.Listen.IpPort) {
		panic("web server did not come up")
	}

	// start a reverse proxy
	rev := NewReverseProxy(ReverseProxyConfig{Dest: web.Cfg.Listen})
	rev.Start()

	if !PortIsBound(rev.Cfg.Listen.IpPort) {
		panic("rev proxy not up")
	}

	// start the forward proxy, talks to the reverse proxy.
	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: rev.Cfg.Listen,
	})
	fwd.Start()
	if !PortIsBound(fwd.Cfg.Listen.IpPort) {
		panic("fwd proxy not up")
	}

	return web, rev, fwd, nil
}
