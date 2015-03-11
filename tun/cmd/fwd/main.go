package main

import (
	"fmt"

	tun "github.com/mailgun/pelican-protocol/tun"
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

	fmt.Printf("fwd listening forever: doing 'select {}'. Use ctrl-c to stop.\n")

	select {}

}
