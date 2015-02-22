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

func TestXor(t *testing.T) {

	cv.Convey("xorWrapBytes() should xor two byte slices together", t, func() {

		a := []byte{0x01, 0x02, 0x03}
		b := []byte{0x10, 0x20, 0x30}
		e := []byte{0x11, 0x22, 0x33}

		o := XorWrapBytes(a, b)
		cv.So(o, cv.ShouldResemble, e)

		a2 := []byte{0x11, 0x22, 0x33}
		e2 := []byte{0x01, 0x02, 0x03}

		o2 := XorWrapBytes(a2, b)
		cv.So(o2, cv.ShouldResemble, e2)

	})
}
