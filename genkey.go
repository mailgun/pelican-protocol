package main

import (
	cryptrand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"code.google.com/p/go.crypto/ssh"
)

func GenRsaKeyPair(rsa_file string, bits int) (priv *rsa.PrivateKey, err error) {

	privKey, err := rsa.GenerateKey(cryptrand.Reader, bits)
	panicOn(err)

	var pubKey *rsa.PublicKey = privKey.Public().(*rsa.PublicKey)

	err = privKey.Validate()
	panicOn(err)

	if rsa_file != "" {

		// write to disk
		// save to pem: serialize private key
		privBytes := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privKey),
			},
		)

		// serialize public key
		pubBytes := RsaToSshPublicKey(pubKey)

		err = ioutil.WriteFile(rsa_file, privBytes, 0600)
		panicOn(err)

		err = ioutil.WriteFile(rsa_file+".pub", pubBytes, 0600)
		panicOn(err)
	}

	return privKey, nil
}

// convert RSA Public Key to an SSH authorized_keys format
func RsaToSshPublicKey(pubkey *rsa.PublicKey) []byte {
	pub, err := ssh.NewPublicKey(pubkey)
	panicOn(err)
	return ssh.MarshalAuthorizedKey(pub)
}
