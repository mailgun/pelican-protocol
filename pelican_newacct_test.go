package pelican

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPelicanAccountShellShoundEstablishNewKey(t *testing.T) {
	StopAllDockers()
	StartDockerImage(DockerHubTestImage)
	defer StopAllDockers()
	cv.Convey("Given the syadmin has installed pelican and so done 'adduser pelican_newacct' and installed the pelsh shell in /etc/passwd under the pelican_newacct account, and added the pelican public key to the ~pelican/.ssh/authorized_keys", t, func() {

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
