package pelican

/*
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"

	"golang.org/x/crypto/ssh"
)

func agentExample() {

	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatal(err)
	}

	agent := agent.NewClient(sock)

	signers, err := agent.Signers()
	if err != nil {
		log.Fatal(err)
	}

	auths := []ssh.AuthMethod{ssh.PublicKeys(signers...)}

	cfg := &ssh.ClientConfig{
		User: "username",
		Auth: auths,
	}
	cfg.SetDefaults()

	client, err := ssh.Dial("tcp", "aws-hostname:22", cfg)
	if err != nil {
		log.Fatal(err)
	}

	session, err = client.NewSession()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("we have a session!")
}
*/
