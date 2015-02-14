package pelican

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPelicanAccountShellShoundEstablishNewKey(t *testing.T) {
	StopAllDockers()
	StartDockerImage(DockerHubTestImage)
	defer StopAllDockers()
	cv.Convey("Given we have made a pelican account and installed the pelican_newacct shell in /etc/passwd under the pelican account, and added the pelican public key to the ~pelican/.ssh/authorized_keys", t, func() {

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
