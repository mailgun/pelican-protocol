package pelican

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestNewPelicanAccountShell(t *testing.T) {
	StopAllDockers()
	StartDockerImage("mailgun/pelican04")
	defer StopAllDockers()
	SshAsRootIntoDocker([]string{"cat", "/etc/passwd"})

	cv.Convey("When the pelsh is given a public key, ", t, func() {

		cv.Convey("When we ssh login with the pelican public/private key pair, we should begin the new account creation protocol", func() {

			cv.Convey("Then: if the server key is completely new, we accept it the first time and initiate the new-account creation protocol", func() {
				cv.So(0, cv.ShouldEqual, 1)

			})
			cv.Convey("Then: if the server key doesn't match our cache, we initiate the new-account creation protocol", func() {
				cv.So(0, cv.ShouldEqual, 1)

			})
		})
	})

}
