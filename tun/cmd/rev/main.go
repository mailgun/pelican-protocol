package main

import (
	"flag"
	"fmt"

	tun "github.com/mailgun/pelican-protocol/tun"
)

var (
	listenAddr = flag.String("listen", ":2222", "local listen address")
	destAddr   = flag.String("dest", fmt.Sprintf("%s:%d", tun.ReverseProxyIp, tun.ReverseProxyPort), "remote destination IP:port")
)

func main() {

	flag.Parse()

	rlsn := tun.NewAddr1(*listenAddr)
	rdest := tun.NewAddr1(*destAddr)

	fmt.Printf("rev starting: '%#v' -> '%#v'\n", rlsn, rdest)

	rev := tun.NewReverseProxy(tun.ReverseProxyConfig{Dest: rdest, Listen: rlsn})
	rev.Start()

	fmt.Printf("rev listening forever: doing 'select {}'. Use ctrl-c to stop.\n")

	select {}
}
