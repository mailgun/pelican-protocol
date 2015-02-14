package pelican

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"os"
	"testing"
)

func KeyDir() string {
	home := os.Getenv("HOME")
	return home + "/.ssh/"
}

func TestKeyPairSaveRestoreWorks(t *testing.T) {
	dir := KeyDir()
	fmt.Println(dir)
	/*
		cv.Convey("When we generate an ssh key pair, we should be able to save and restore the private and public keys\n", t, func() {
			path := dir + "id_rsa_keypairsavetest"

			privkey, err := loadRSAPrivateKey(path)
			panicOn(err)
			equal := true
			cv.So(equal, cv.ShouldEqual, true)
			cv.So(privkey, cv.ShouldEqual, "expected priv key")
		})
	*/
	cv.Convey("When we generate an ssh key pair and install it on our docker server, the save and restore the private and public keys in a format usable by ssh/sshd\n", t, func() {

	})
}
