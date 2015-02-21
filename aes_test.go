package pelican

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestAes(t *testing.T) {

	cv.Convey("EncryptAes256Gcm() and DecryptAes256Gcm() should be inverses\n", t, func() {

		originalText := "8 encrypt this golang 123"
		fmt.Println(originalText)

		pass := []byte("hello")

		// encrypt value to base64
		cryptoText := EncryptAes256Gcm(pass, []byte(originalText))
		fmt.Println(string(cryptoText))

		// encrypt base64 crypto to original value
		text := DecryptAes256Gcm(pass, cryptoText)

		cv.So(string(text), cv.ShouldEqual, originalText)

	})
}
