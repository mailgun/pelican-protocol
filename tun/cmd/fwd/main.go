package main

import (
	"flag"
	"fmt"

	tun "github.com/mailgun/pelican-protocol/tun"
)

var (
	listenAddr = flag.String("listen", ":1080", "local listen address, IP:port")
	destAddr   = flag.String("dest", ":80", "remote destination, IP:port")
)

func main() {

	flag.Parse()

	flsn := tun.NewAddr1panicOnError(*listenAddr)
	fdest := tun.NewAddr1panicOnError(*destAddr)

	fmt.Printf("fwd starting: '%#v' -> '%#v'\n", flsn, fdest)

	fwd := tun.NewPelicanSocksProxy(tun.PelicanSocksProxyConfig{
		Listen: flsn,
		Dest:   fdest,
	})
	fwd.Start()

	fmt.Printf("fwd listening forever: doing 'select {}'. Use ctrl-c to stop.\n")

	select {}

}
