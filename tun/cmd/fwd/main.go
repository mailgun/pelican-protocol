package main

import (
	"fmt"
	tun "github.com/mailgun/pelican-protocol/tun"
	"time"
)

func main() {

	flsn := tun.NewAddr1("127.0.0.1:7777")
	fdest := tun.NewAddr1("127.0.0.1:9999")

	fmt.Printf("fwd starting: '%#v' -> '%#v'\n", flsn, fdest)

	fwd := tun.NewPelicanSocksProxy(tun.PelicanSocksProxyConfig{
		Listen: flsn,
		Dest:   fdest,
	})
	fwd.Start()

	time.Sleep(60 * time.Minute)
	fwd.Stop()
	fmt.Printf("fwd stopped.\n")
	time.Sleep(600 * time.Minute)
}
