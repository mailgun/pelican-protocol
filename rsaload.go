package pelican

import (
	"golang.org/x/crypto/ssh"
	"io/ioutil"
)

func LoadRSAPrivateKey(path string) (privkey ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(path)
	panicOn(err)

	privkey, err = ssh.ParsePrivateKey(buf)
	panicOn(err)

	return privkey, err
}

func LoadRSAPublicKey(path string) (pubkey ssh.PublicKey, err error) {
	buf, err := ioutil.ReadFile(path)
	panicOn(err)

	pub, _, _, _, err := ssh.ParseAuthorizedKey(buf)
	panicOn(err)

	return pub, err
}
