package pelican

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"os"
	"testing"
)

func TestKnownHostsSaveAndRestoreWork(t *testing.T) {
	StopAllDockers()
	StartDockerImage("jaten/pelican04")
	defer StopAllDockers()

	my_known_hosts_file := "my.known.hosts"
	os.Remove(my_known_hosts_file + defaultFileFormat())
	h := NewKnownHosts(my_known_hosts_file)
	defer h.Close()

	pw, err := h.SshAsRootIntoDocker([]string{"cat", "/etc/passwd"})
	if err != nil {
		fmt.Printf("error: '%s', output during SshAsRootIntoDocker(): '%s'\n", err, string(pw))
		panic(err)
	}
	fmt.Printf("pw seen: '%s'\n", string(pw))

	cv.Convey("When NewKnownHosts() is given an existing known_hosts file path, we should restore the previously known hosts set.\n", t, func() {
		h2 := NewKnownHosts(my_known_hosts_file)
		defer h2.Close()

		equal, err := KnownHostsEqual(h, h2)
		if !equal {
			fmt.Printf("\n a is '%#v'\n\n b is '%#v'\n\n", h, h2)
			panic(err)
		}
		cv.So(equal, cv.ShouldEqual, true)
	})

}
