package pelican

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestLoadPasswordProtectedSshKeys(t *testing.T) {

	PasswordToSshPrivKeyUnlocker(pass, iv)
	cv.Convey("We should be able to load an ssh private key that is password protected ", t, func() {
		// TODO: implement once this is important enough.
		//cv.So(0, cv.ShouldEqual, 0)
	})
}
