package pelican

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"io/ioutil"
	"net"
)

func loadRSAPrivateKey(path string) (privkey ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	privkey, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		panic(err)
	}
	return privkey, err
}

func hostKeyCallback(hostname string, remote net.Addr, key ssh.PublicKey) error {
	fmt.Printf("in hostKeyCallback(), hostname='%s', remote='%s', key='%s'\n", hostname, remote, key)
	return nil
}

func sshConnect(username string, keypath string, host string, port int, command string) ([]byte, error) {

	privkey, err := loadRSAPrivateKey(keypath)
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
