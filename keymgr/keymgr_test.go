package main

import (
	"math/big"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestSha1ToUsername(t *testing.T) {

	cv.Convey("encodeSha1AsUsername should return expected encodings of sha1 160 bit numbers in 31 bytes", t, func() {
		cv.So(bigIntToBase36string(big.NewInt(0)), cv.ShouldEqual, "0000000000000000000000000000000")

		cv.So(bigIntToBase36string(big.NewInt(35)), cv.ShouldEqual, "000000000000000000000000000000z")

		cv.So(bigIntToBase36string(big.NewInt(36)), cv.ShouldEqual, "0000000000000000000000000000010")

		cv.So(bigIntToBase36string(big.NewInt(36*36)), cv.ShouldEqual, "0000000000000000000000000000100")

		two := big.NewInt(2)
		b160 := big.NewInt(160)
		max160 := new(big.Int)
		max160.Exp(two, b160, nil)

		/*
			for i := int64(1); i < 160; i++ {
				m := new(big.Int)
				fmt.Printf("2 ^ %d == %s\n", i, m.Exp(two, big.NewInt(i), nil))
			}

			fmt.Printf("max160 = %s\n", max160)
		*/
		cv.So(bigIntToBase36string(max160), cv.ShouldEqual, "twj4yidkw7a8pn4g709kzmfoaol3x8g")

		shaAllBitsSet := make([]byte, 20)
		for i := range shaAllBitsSet {
			shaAllBitsSet[i] = 0xff
		}
		// should be 2^160 -1, so the 'g' at the end becomes an 'f'
		cv.So(encodeSha1AsUsername(shaAllBitsSet), cv.ShouldEqual, "ptwj4yidkw7a8pn4g709kzmfoaol3x8f")
	})

}
