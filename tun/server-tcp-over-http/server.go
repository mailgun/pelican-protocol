package main

import (
	"flag"
	"fmt"

	tun "github.com/mailgun/pelican-protocol/tun"
)

//var destAddr = "127.0.0.1:12222" // tunnel destination
var destAddr = "127.0.0.1:22" // tunnel destination
//var destAddr = "127.0.0.1:1234" // tunnel destination

var listenAddr = flag.String("http", fmt.Sprintf("%s:%d", tun.ReverseProxyIp, tun.ReverseProxyPort), "http listen address")

func main() {
	flag.Parse()

	s := tun.NewReverseProxy(*listenAddr, destAddr)
	s.ListenAndServe()
}
