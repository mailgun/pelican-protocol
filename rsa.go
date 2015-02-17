package pelican

import (
	cryptrand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
)

// GenRsaKeyPair generates an RSA keypair of length bits. If rsa_file != "", we write
// the private key to rsa_file and the public key to rsa_file + ".pub". If rsa_file == ""
// the keys are not written to disk.
//
func GenRsaKeyPair(rsa_file string, bits int) (priv *rsa.PrivateKey, sshPriv ssh.Signer, err error) {

	privKey, err := rsa.GenerateKey(cryptrand.Reader, bits)
	panicOn(err)

	var pubKey *rsa.PublicKey = privKey.Public().(*rsa.PublicKey)

	err = privKey.Validate()
	panicOn(err)

	// write to disk
	// save to pem: serialize private key
	privBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privKey),
		},
	)

	sshPrivKey, err := ssh.ParsePrivateKey(privBytes)
	panicOn(err)

	if rsa_file != "" {

		// serialize public key
		pubBytes := RsaToSshPublicKey(pubKey)

		err = ioutil.WriteFile(rsa_file, privBytes, 0600)
		panicOn(err)

		err = ioutil.WriteFile(rsa_file+".pub", pubBytes, 0600)
		panicOn(err)
	}

	return privKey, sshPrivKey, nil
}

// RsaToSshPublicKey convert an RSA Public Key to the SSH authorized_keys format.
func RsaToSshPublicKey(pubkey *rsa.PublicKey) []byte {
	pub, err := ssh.NewPublicKey(pubkey)
	panicOn(err)
	return ssh.MarshalAuthorizedKey(pub)
}

// LoadRSAPrivateKey reads a private key from path on disk.
func LoadRSAPrivateKey(path string) (privkey ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(path)
	panicOn(err)

	privkey, err = ssh.ParsePrivateKey(buf)
	panicOn(err)

	return privkey, err
}

// LoadRSAPublicKey reads a public key from path on disk. By convention
// these keys end in '.pub', but that is not verified here.
func LoadRSAPublicKey(path string) (pubkey ssh.PublicKey, err error) {
	buf, err := ioutil.ReadFile(path)
	panicOn(err)

	pub, _, _, _, err := ssh.ParseAuthorizedKey(buf)
	panicOn(err)

	return pub, err
}
