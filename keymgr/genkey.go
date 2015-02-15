package main

import (
	"code.google.com/p/go.crypto/ssh"
	cryptrand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"github.com/mailgun/pelican-protocol"
	"io/ioutil"
	"reflect"
)

func GenRsaKeyPair(rsa_file string, bits int) (priv *rsa.PrivateKey, err error) {

	privKey, err := rsa.GenerateKey(cryptrand.Reader, bits)
	panicOn(err)
	fmt.Printf("done generating key.\n")
	err = priv.Validate()
	panicOn(err)

	pubKey := priv.Public()
	// write to disk

	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	panicOn(err)

	pub2, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
	panicOn(err)
	if !reflect.DeepEqual(pubKey, pub2) {
		panic("error on serializing and re-reading RSA Public key")
	}

	err = ioutil.WriteFile(rsa_file, pubBytes, 0600)
	panicOn(err)

	priv2, err := pelican.LoadRSAPrivateKey(rsa_file)
	panicOn(err)

	if !reflect.DeepEqual(privKey, priv2) {
		panic("error on serializing and re-reading RSA Private key")
	}

	return privKey, nil
}
