package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	tun "github.com/mailgun/pelican-protocol/tun"
)

var (
	listenAddr = flag.String("listen", ":8080", "local listen address")
)

func main() {
	flag.Parse()
	lsn := tun.NewAddr1panicOnError(*listenAddr)
	lsn.SetIpPort()

	fmt.Printf("web listening on '%#v'\n", lsn)

	// setup a mock web server that replies to ping with pong.
	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	web, err := tun.NewWebServer(tun.WebServerConfig{Listen: lsn}, mux, 60*time.Second)
	if err != nil {
		panic(err)
	}
	web.Start("cmd web web-server: ping->pong")

	fmt.Printf("web serving 'ping' with 'pong'; listening forever: doing 'select {}'. Use ctrl-c to stop.\n")

	select {}
}
