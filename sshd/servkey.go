package main

import (
	"fmt"
	"io/ioutil"

	"github.com/mailgun/pelican-protocol"
	"golang.org/x/crypto/ssh"
)

// todo: expand to support multiple host keys and allow
// them to be rotated too.
func GetOrGenServerKey(filepath string, config *ssh.ServerConfig) error {

	var signer ssh.Signer
	var err error
	if FileExists(filepath) {

		privateBytes, err := ioutil.ReadFile(filepath)
		if err != nil {
			return fmt.Errorf("Failed to load host private key: '%s'", err)
		}

		signer, err = ssh.ParsePrivateKey(privateBytes)
		if err != nil {
			return fmt.Errorf("Failed to parse host private key: '%s'", err)
		}

	} else {
		// no such filepath
		_, signer, err = pelican.GenRsaKeyPair(filepath, 2048)
		if err != nil {
			return fmt.Errorf("Generating host key failed in GenRsaKeyPair(): '%s'", err)
		}
	}

	config.AddHostKey(signer)

	return nil
}
