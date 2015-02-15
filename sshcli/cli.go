package main

import (
	"fmt"

	"github.com/mailgun/pelican-protocol"
)

func main() {
	my_known_hosts_file := "my.known.hosts"
	h := pelican.NewKnownHosts(my_known_hosts_file)
	defer h.Close()

	out, err := h.SshConnect("root", "./id_rsa", "localhost", 2200, "pwd")
	if err != nil {
		panic(err)
	}
	fmt.Printf("out = '%s'\n", string(out))
}
