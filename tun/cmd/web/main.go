package main

import (
	"fmt"
	tun "github.com/mailgun/pelican-protocol/tun"
	"time"
)

func main() {
	lsn := tun.NewAddr2("127.0.0.1", 8080)

	fmt.Printf("web listening on '%#v'\n", lsn)
	web := tun.NewWebServer(tun.WebServerConfig{Listen: lsn}, nil)
	web.Start() // without this, hang doesn't happen
	defer web.Stop()

	time.Sleep(60 * time.Minute)
	fmt.Printf("web done.\n")
}
