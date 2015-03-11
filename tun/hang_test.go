package pelicantun

import (
	"fmt"
	"testing"
)

func TestFullRoundtripAllCanShutdown009(t *testing.T) {

	web := NewWebServer(WebServerConfig{}, nil)
	web.Start() // without this, hang doesn't happen
	//defer web.Stop()

	// start a reverse proxy

	// no leak with only rev + fwd together.

	rdest := web.Cfg.Listen
	//rdest := NewAddr1("127.0.0.1:9090")

	rev := NewReverseProxy(ReverseProxyConfig{Dest: rdest})
	rev.Start()
	//defer rev.Stop()

	// start the forward proxy, talks to the reverse proxy.

	// no leak when fwd is stand alone, and no leak without fwd.

	//dest := NewAddr1("127.0.0.1:9090")

	dest := rev.Cfg.Listen

	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: dest,
	})
	fmt.Printf("fwd = %#v\n", fwd)
	fwd.Start() // fwd must start for the hang to happen
	fwd.Stop()

	fmt.Printf("\n done with Test Full Roundtrip All Can Shutdown 009()\n")
	// hangs for 60 seconds, then finishes???

	rev.Stop()

	web.Stop() // this is where we are hanging, for sure.
}
