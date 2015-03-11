package main

import (
	"fmt"
	tun "github.com/mailgun/pelican-protocol/tun"
	"net/http"
	"time"
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

	web := tun.NewWebServer(tun.WebServerConfig{Listen: lsn}, mux)
	web.Start() // without this, hang doesn't happen

	time.Sleep(60 * time.Minute)

	web.Stop()

	fmt.Printf("web done.\n")

	time.Sleep(600 * time.Minute)
}
