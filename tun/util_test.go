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

	web, err := NewWebServer(WebServerConfig{}, mux, specialFastTestReadTimeout)
	panicOn(err)
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

// srv, rev, fwd, cli: all except client are started.
// typically you'll want to do:
//
//	 cli.Start()
//	 <-srv.FirstClient // for the WaitUntilServerIsUp client
//	 <-srv.SecondClient // for the actual client
//
// and only then proceed to
//   srv.Bcast("BREAKING NEWS")
//
// to avoid racing the client finding the server against
// the server broadcasting out to nobody.
//
func StartTestSystemWithBcast() (*BcastClient, *BcastServer, *ReverseProxy, *PelicanSocksProxy, error) {

	// start broadcast server (to test long-poll functionality/server initiated message)
	srv := NewBcastServer(Addr{})
	srv.Start()

	// start a reverse proxy
	rev := NewReverseProxy(ReverseProxyConfig{Dest: srv.Listen})
	rev.Start()

	if !TempDisablePortIsBoundChecks {
		if !PortIsBound(rev.Cfg.Listen.IpPort) {
			panic("rev proxy not up")
		}
	}

	// start the forward proxy, talks to the reverse proxy.
	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: rev.Cfg.Listen,
	})
	fwd.Start()

	if !TempDisablePortIsBoundChecks {
		if !PortIsBound(fwd.Cfg.Listen.IpPort) {
			panic("fwd proxy not up")
		}
	}

	// start broadcast client (to test receipt of long-polled data from server)
	cli := NewBcastClient(Addr{Port: fwd.Cfg.Listen.Port})

	return cli, srv, rev, fwd, nil
}
