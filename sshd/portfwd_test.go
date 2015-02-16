package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/mailgun/pelican-protocol"
	"testing"
)

func TestClientToServerPortForward(t *testing.T) {

	// 1. web server
	//
	webPort := pelican.GetAvailPort()
	webAddr := fmt.Sprintf("127.0.0.1:%d", webPort)
	w := NewWebServer(webAddr, nil)
	w.Start()
	defer w.Stop()

	// 2. sshd in front of web server
	//
	pelPort := pelican.GetAvailPort()
	pelIp := "127.0.0.1"

	peld := NewPelicanServer(&PelSrvCfg{
		PelicanListenPort: pelPort,
		HttpDestIp:        pelIp,
		HttpDestPort:      webPort,
	})
	peld.Start()
	defer peld.Stop()

	// 3. client makes a new account on the pelical-protocol/sshd server,
	// gets the server's hostkey, provides the server a client public key.
	my_known_hosts_file := "my.known.hosts"
	pelican.CleanupOldKnownHosts(my_known_hosts_file)

	h := pelican.NewKnownHosts(my_known_hosts_file)
	defer h.Close()

	newKey := pelican.GetNewAcctPrivateKey()
	acctId, err := h.SshMakeNewAcct(newKey, pelIp, pelPort)
	panicOn(err)

	// 4. fetch some traffic from the website via the tunnel
	//
	localPortToListenOn := pelican.GetAvailPort()
	out, err := h.SshConnect(acctId, acctKey, dockerip, 22, localPortToListenOn)
	if err != nil {
		panic(err)
	}

	cv.Convey("When the client tunnels bidirectional http traffic to the server, the server should forward that traffic to the local webserver", t, func() {

		page := MyCurl(fmt.Sprintf("http://127.0.0.1:%d", localPortToListenOn))
		cv.So(page, cv.ShouldContainSubstring, "[This is the main static body.]")

	})

}
