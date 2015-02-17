package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/mailgun/pelican-protocol"
	"testing"
	"time"
)

func TestClientToServerPortForward(t *testing.T) {

	// 1. web server
	//
	webPort := pelican.GetAvailPort()
	webIp := "127.0.0.1"
	webAddr := fmt.Sprintf("%s:%d", webIp, webPort)
	fmt.Printf("web addr is '%s'\n", webAddr)
	w := NewWebServer(webAddr, nil)
	w.Start()
	defer w.Stop()

	// 2. sshd in front of web server
	//
	pelPort := pelican.GetAvailPort()
	if pelPort == webPort {
		panic(fmt.Sprintf("port conflict! both web and pelican server on port %d", pelPort))
	}
	pelIp := "127.0.0.1"
	fmt.Printf("pelican-server addr is '%s:%d'\n", pelIp, pelPort)

	peld := NewPelicanServer(&PelSrvCfg{
		PelicanListenIp:   pelIp,
		PelicanListenPort: pelPort,
		HttpDestIp:        webIp,
		HttpDestPort:      webPort,
		HttpDialTimeout:   5 * time.Second,
	})
	peld.Start()
	fmt.Printf("after Start()\n")
	defer peld.Stop()

	// 3. client makes a new account on the pelical-protocol/sshd server,
	// gets the server's hostkey, provides the server a client public key.
	// The client's public key is what lets the server know that it is
	// the same client returning from before.
	fmt.Printf("pre known hosts.\n")
	my_known_hosts_file := "my.known.hosts"
	pelican.CleanupOldKnownHosts(my_known_hosts_file)

	h := pelican.NewKnownHosts(my_known_hosts_file)
	defer h.Close()

	fmt.Printf("before new private key.\n")
	privKeyFile := "temp-2048-key"
	_, _, err := pelican.GenRsaKeyPair(privKeyFile, 2048)
	panicOn(err)

	fmt.Printf("done with recalling the one new account private key. stored in: '%s'\n", privKeyFile)

	/* // skip this for now, is make new acct not a new activity??

	acctId, err := h.SshMakeNewAcct(acctKey, pelIp, pelPort)
	panicOn(err)

	fmt.Printf("done with SshMakeNewAcct(). acctId = '%s'\n", acctId)
	*/

	// 4. fetch some traffic from the website via the tunnel
	//
	acctId := "newacct" // essential at the moment, with the current state of sshutil.go
	localPortToListenOn := pelican.GetAvailPort()
	fmt.Printf("sshConnect will listen on port %d\n", localPortToListenOn)
	_, err = h.SshConnect(acctId, privKeyFile, pelIp, pelPort, localPortToListenOn)
	fmt.Printf("\n done with h.SshConnect().\n")
	if err != nil {
		panic(err)
	}

	cv.Convey("When the client tunnels bidirectional http traffic to the server, the server should forward that traffic to the local webserver", t, func() {

		page := MyCurl(fmt.Sprintf("http://127.0.0.1:%d", localPortToListenOn))
		cv.So(page, cv.ShouldContainSubstring, "[This is the main static body.]")

	})

}
