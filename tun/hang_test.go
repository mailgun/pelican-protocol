package pelicantun

import (
	"fmt"
	"testing"
)

func TestFullRoundtripAllCanShutdown009(t *testing.T) {

	web := NewWebServer(WebServerConfig{}, nil)
	web.Start() // without this, hang doesn't happen
	defer web.Stop()

	// start a reverse proxy
	rev := NewReverseProxy(ReverseProxyConfig{Dest: web.Cfg.Listen})
	rev.Start()
	defer rev.Stop()

	// start the forward proxy, talks to the reverse proxy.
	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: rev.Cfg.Listen,
	})
	fmt.Printf("fwd = %#v\n", fwd)
	fwd.Start() // fwd must start for the hang to happen, looks like leftover goroutine is from pwd.
	fwd.Stop()

	fmt.Printf("\n done with Test Full Roundtrip All Can Shutdown 009()\n")

	// hangs for 60 seconds, then finishes???
}
