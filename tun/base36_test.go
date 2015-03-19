package main

import (
	"fmt"
	"math/big"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestBase36Encode(t *testing.T) {

	var s string
	var i *big.Int
	var by []byte
	cv.Convey("BigIntToBase36 should encode big.Ints correctly", t, func() {

		_, s = BigIntToBase36(big.NewInt(0))
		cv.So(s, cv.ShouldResemble, "0")

		_, s = BigIntToBase36(big.NewInt(1))
		cv.So(s, cv.ShouldResemble, "01")

		_, s = BigIntToBase36(big.NewInt(2))
		cv.So(s, cv.ShouldResemble, "02")

		_, s = BigIntToBase36(big.NewInt(10))
		cv.So(s, cv.ShouldResemble, "0a")

		i, by, _ = Base36toBigInt([]byte(s))
		cv.So(i, cv.ShouldResemble, big.NewInt(10))

		_, s = BigIntToBase36(big.NewInt(35))
		cv.So(s, cv.ShouldResemble, "0z")

		_, s = BigIntToBase36(big.NewInt(36))
		cv.So(s, cv.ShouldResemble, "10")

		_, s = BigIntToBase36(big.NewInt(36 * 36))
		cv.So(s, cv.ShouldResemble, "0100")

		_, s = BigIntToBase36(big.NewInt(36*36 + 35))
		cv.So(s, cv.ShouldResemble, "010z")

		i, by, _ = Base36toBigInt([]byte(s))
		//fmt.Printf("\ni = %v, by = '%x'\n", i, by)
		cv.So(i, cv.ShouldResemble, big.NewInt(1331)) // 1331 == 36*36+35

		_, s = BigIntToBase36(big.NewInt(36*35 + 35))
		cv.So(s, cv.ShouldResemble, "00zz")

		_, s = BigIntToBase36(big.NewInt(36*36*36 + 1))
		cv.So(s, cv.ShouldResemble, "1001")

		i, by, _ = Base36toBigInt([]byte(s))
		cv.So(i, cv.ShouldResemble, big.NewInt(46657)) // 46657 == 36*36*36 + 1
		//fmt.Printf("\ni = %v, by = '%x'\n", i, by)
	})
}

func TestBase36Decode(t *testing.T) {

	cv.Convey("Base36toBigInt and decode36 should decode correctly", t, func() {

		for i, r := range e36 {
			cv.So(decode36(r), cv.ShouldEqual, i)
		}

		rby := []byte{0x23, 0xff}

		i := new(big.Int)
		i.SetBytes(rby)

		erby := EncodeBytesBase36(rby)
		cv.So(erby, cv.ShouldResemble, []byte("073z"))

		//fmt.Printf("\n\n   rby = '%d', rby=0x'%x' erby = '%s'   i='%v', i='%s'\n", rby, rby, erby, i, i)

		i2, b2, err := Base36toBigInt(erby)
		//fmt.Printf("\n\n   i2 = '%v', b2='%v'\n", i2, b2)
		cv.So(err, cv.ShouldEqual, nil)
		cv.So(i2.String(), cv.ShouldResemble, fmt.Sprintf("%d", 9215))
		cv.So(b2[2:], cv.ShouldResemble, rby)
	})
}
