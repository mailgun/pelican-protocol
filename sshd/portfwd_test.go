package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/mailgun/pelican-protocol"
	"os"
	"testing"
)

func TestClientToServerPortForward(t *testing.T) {

	sshd := NewSshd()
	ssh.Start()

	my_known_hosts_file := "my.known.hosts"
	CleanupOldKnownHosts(my_known_hosts_file)

	h := NewKnownHosts(my_known_hosts_file)
	defer h.Close()

	err := h.SshMakeNewAcct(pelican.GetNewAcctPrivateKey(), "localhost", 2200)
	panicOn(err)

	cv.Convey("When the client tunnels bidirectional http traffic to the server, the server should forward that traffic to the local website on port 80", t, func() {

	})

}
