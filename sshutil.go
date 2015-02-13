package pelican

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"io/ioutil"
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

// callers should do defer sess.Close() with the sess returned in the first return value.
func sshConnect(username string, keypath string, host string, port int, command string) (*ssh.Session, []byte, error) {

	privkey, err := loadRSAPrivateKey(keypath)
	if err != nil {
		panic(err)
	}

	cfg := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privkey),
		},
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

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	sess.Stdout = &b
	if err := sess.Run(command); err != nil {
		panic(fmt.Sprintf("sshConnect() failed to run login to '%s', err: '%s', out: '%s'", hostport, err.Error(), b.String()))
	}
	//fmt.Println(b.String())

	return sess, b.Bytes(), nil
}
