package main

/*
import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/mailgun/pelican-protocol"
	"testing"
)

// func main() {
// 	fmt.Printf("cli.go starting.\n")
//
// 	my_known_hosts_file := "my.known.hosts"
// 	h := pelican.NewKnownHosts(my_known_hosts_file)
// 	defer h.Close()
//
// 	fmt.Printf("cli.go done with NewKnownHosts().\n")
// 	err := h.SshMakeNewAcct(pelican.GetNewAcctPrivateKey(), "localhost", 2200)
// 	panicOn(err)
//
// }
//

func TestServerRecognizesTheNewAcctKey(t *testing.T) {

	sshd := NewSshd()
	ssh.Start()

	my_known_hosts_file := "my.known.hosts"
	CleanupOldKnownHosts(my_known_hosts_file)

	h := NewKnownHosts(my_known_hosts_file)
	defer h.Close()

	err := h.SshMakeNewAcct(pelican.GetNewAcctPrivateKey(), "localhost", 2200)
	panicOn(err)

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
*/
