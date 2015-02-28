package main

import (
	"math/big"
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
)

func TestBase36Encode(t *testing.T) {

	var s string
	cv.Convey("BigIntToBase36 should encode big.Ints correctly", t, func() {

		_, s = BigIntToBase36(big.NewInt(0))
		cv.So(s, cv.ShouldResemble, "0")

		_, s = BigIntToBase36(big.NewInt(1))
		cv.So(s, cv.ShouldResemble, "01")

		_, s = BigIntToBase36(big.NewInt(2))
		cv.So(s, cv.ShouldResemble, "02")

		_, s = BigIntToBase36(big.NewInt(10))
		cv.So(s, cv.ShouldResemble, "0a")

		_, s = BigIntToBase36(big.NewInt(35))
		cv.So(s, cv.ShouldResemble, "0z")

		_, s = BigIntToBase36(big.NewInt(36))
		cv.So(s, cv.ShouldResemble, "10")

		_, s = BigIntToBase36(big.NewInt(36 * 36))
		cv.So(s, cv.ShouldResemble, "0100")

		_, s = BigIntToBase36(big.NewInt(36*36 + 35))
		cv.So(s, cv.ShouldResemble, "010z")

		_, s = BigIntToBase36(big.NewInt(36*35 + 35))
		cv.So(s, cv.ShouldResemble, "00zz")

		_, s = BigIntToBase36(big.NewInt(36*36*36 + 1))
		cv.So(s, cv.ShouldResemble, "1001")

	})
}
