package pelicantun

import (
	"fmt"
	"testing"
	"time"
)

var specialFastTestReadTimeout time.Duration = 500 * time.Millisecond

func TestFullRoundtripAllCanShutdown009(t *testing.T) {

	web := NewWebServer(WebServerConfig{}, nil, specialFastTestReadTimeout)
	web.Start()
	//defer web.Stop()

	// start a reverse proxy

	rdest := web.Cfg.Listen
	//rdest := NewAddr1("127.0.0.1:9090")

	rev := NewReverseProxy(ReverseProxyConfig{Dest: rdest})
	rev.Start()
	//defer rev.Stop()

	// start the forward proxy, talks to the reverse proxy.

	//dest := NewAddr1("127.0.0.1:9090")

	dest := rev.Cfg.Listen

	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: dest,
	})
	//fmt.Printf("fwd = %#v\n", fwd)
	fwd.Start()

	fwd.Stop()

	rev.Stop()

	web.Stop()

	fmt.Printf("\n done with Test Full Roundtrip All Can Shutdown 009()\n")
}
