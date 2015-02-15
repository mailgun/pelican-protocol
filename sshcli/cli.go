package main

import (
	"bytes"
	"fmt"
	"net"

	"github.com/mailgun/pelican-protocol"
	"golang.org/x/crypto/ssh"
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

func (h *KnownHosts) SshConnect(username string, keypath string, host string, port int, command string) ([]byte, error) {

	// the callback just after key-exchange to validate server is here
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {

		pubBytes := ssh.MarshalAuthorizedKey(key)

		hostStatus, err, spubkey := h.HostAlreadyKnown(hostname, remote, key, pubBytes, AddIfNotKnown)
		fmt.Printf("in hostKeyCallback(), hostStatus: '%s', hostname='%s', remote='%s', key.Type='%s'  key.Marshal='%s'\n", hostStatus, hostname, remote, key.Type(), pubBytes)

		h.curStatus = hostStatus
		h.curHost = spubkey

		if hostStatus == Banned {
			return fmt.Errorf("banned server")
		}

		// super lenient for the moment.
		err = nil
		return err
		/*

			if err != nil {
				// this is strict checking of hosts here, any non-nil error
				// will fail the ssh handshake.
				return err
			}

			return nil
		*/
	}
	// end hostKeyCallback closure definition. Has to be a closure to access h.

	privkey, err := LoadRSAPrivateKey(keypath)
	if err != nil {
		panic(err)
	}

	cfg := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privkey),
		},
		// HostKeyCallback, if not nil, is called during the cryptographic
		// handshake to validate the server's host key. A nil HostKeyCallback
		// implies that all host keys are accepted.
		HostKeyCallback: hostKeyCallback,
	}
	hostport := fmt.Sprintf("%s:%d", host, port)
	cli, err := ssh.Dial("tcp", hostport, cfg)
	if err != nil {
		panic(fmt.Sprintf("sshConnect() failed at dial to '%s': '%s' ", hostport, err.Error()))
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	sess, err := cli.NewSession()
	if err != nil {
		panic(fmt.Sprintf("Failed to create session to '%s': err = '%s'", hostport, err.Error()))
	}
	// you can only do one command on a session, so might as well close.
	defer sess.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	sess.Stdout = &b
	if err := sess.Run(command); err != nil {
		panic(fmt.Sprintf("sshConnect() failed to run login to '%s', err: '%s', out: '%s'", hostport, err.Error(), b.String()))
	}
	//fmt.Println(b.String())

	return b.Bytes(), nil
}
