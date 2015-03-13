package main

import (
	"flag"
	"fmt"

	tun "github.com/mailgun/pelican-protocol/tun"
)

var (
	listenAddr = flag.String("listen", ":80", "local listen address")
	destAddr   = flag.String("dest", ":8080", "remote destination IP:port")
)

func main() {

	flag.Parse()

	rlsn := tun.NewAddr1panicOnError(*listenAddr)
	rdest := tun.NewAddr1panicOnError(*destAddr)

	fmt.Printf("rev starting: '%#v' -> '%#v'\n", rlsn, rdest)

	rev := tun.NewReverseProxy(tun.ReverseProxyConfig{Dest: rdest, Listen: rlsn})
	rev.Start()

	fmt.Printf("rev listening forever: doing 'select {}'. Use ctrl-c to stop.\n")

	select {}
}
