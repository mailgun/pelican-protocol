package main

import (
	"fmt"
	"net/http"
	"time"

	tun "github.com/mailgun/pelican-protocol/tun"
)

func main() {
	lsn := tun.NewAddr2("127.0.0.1", 8080)

	fmt.Printf("web listening on '%#v'\n", lsn)

	// setup a mock web server that replies to ping with pong.
	mux := http.NewServeMux()

	// ping allows our test machinery to function
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		fmt.Fprintf(w, "pong")
	})

	web := tun.NewWebServer(tun.WebServerConfig{Listen: lsn}, mux, 60*time.Minute)
	web.Start() // without this, hang doesn't happen

	web.Stop()

	fmt.Printf("web listening forever: doing 'select {}'. Use ctrl-c to stop.\n")

	select {}
}
