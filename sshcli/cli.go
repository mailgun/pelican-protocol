package main

import (
	"fmt"

	"github.com/mailgun/pelican-protocol"
)

func main() {
	fmt.Printf("cli.go starting.\n")

	my_known_hosts_file := "my.known.hosts"
	h := pelican.NewKnownHosts(my_known_hosts_file)
	defer h.Close()

	fmt.Printf("cli.go done with NewKnownHosts().\n")
	_, err := h.SshMakeNewAcct(pelican.GetOriginatorPrivateKey(), "localhost", 2200)
	panicOn(err)

}
