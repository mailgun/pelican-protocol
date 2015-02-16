package pelican

/*

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)


func TestNewPelicanAccountShell(t *testing.T) {
	fmt.Printf("skipping test for now.\n")
		StopAllDockers()
		StartDockerImage(DockerHubTestImage)
		//defer StopAllDockers()

		my_known_hosts_file := "my.known.hosts"
		CleanupOldKnownHosts(my_known_hosts_file)
		h := NewKnownHosts(my_known_hosts_file)
		defer h.Close()

		pw, err := h.SshAsRootIntoDocker([]string{"cat", "/etc/passwd"})
		if err != nil {
			fmt.Printf("error: '%s', output during SshAsRootIntoDocker(): '%s'\n", err, string(pw))
			panic(err)
		}
		fmt.Printf("pw seen: '%s'\n", string(pw))

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
*/
