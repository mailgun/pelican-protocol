package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/mailgun/pelican-protocol"
	"testing"
)

// It is desirable to allow clients to choose what identity to
// present based on the the identity of the server. We may wish
// to stay anonymous for Teletubbies fan sites, whilst providing a
// different identity to the bank.
//
// Since clients choose amongst a choice of keys for a given server,
// (a client may even choose to generate a new account every time, or
// to cycle between accounts), the client may not in general know what
// creds to provide to the server until we know who the server is.
//
// Make sure we can choose our key *after* we know the server's
// authenticated public key. Test this by switching the keys once
// we know the server's identity.
//
// This should be possible since the Diffie-Hellman key exchange
// happens and authenticates the server via server host key checking
// (a shared secret is obtained, signed by the server's private
// key, and then verified by the client decrypting that with
// the server's public key to the same shared secret). In the
// protocol, the key exchange happens *before* any user
// authorization. http://tools.ietf.org/html/rfc4253#section-8
//
// We simply wish to check that the API provided by
// "golang.org/x/crypto/ssh" lets us swap RSA keys before
// doing the userauth step. Reading the code, this looks
// pretty straight forward. The callback for host key
// verification is the point at which the client should
// then choose the key for this server; or generate a
// new one.
//
// The userauth step should use
// "keyboard-interactive" auth to allow multiple rounds of
// authentication (e.g. by 2-factor auth / SMS code / rsa hardware device
// and in particular, the ubiquitous email verification.
// See http://www.ietf.org/rfc/rfc4256.txt, entitled "Generic Message
// Exchange Authentication for the Secure Shell Protocol (SSH)" for
// details about "keyboard-interactive". Keyboard interactive
// enables the use of one-time passwords.
//
// Notice though that the protocol will run all keys that clients
// provide past the server to see if any of them are acceptable.
// So keep identities not loaded until we know who on the server
// end we are dealing with. ~/go/src/github.com/golang/crypto/ssh/client_auth.go:180
//
func TestClientCredentialsChosenAfterServerHostKeyVerified(t *testing.T) {

	// 1. sshd
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
