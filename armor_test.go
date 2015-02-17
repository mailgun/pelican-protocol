package pelican

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"testing"
)

func TestAsciiArmor(t *testing.T) {

	cv.Convey("WrapInAsciiArmor() and RemoveAsciiArmor() should be inverses", t, func() {

		wrap, err := WrapInAsciiArmor(data)
		panicOn(err)

		fmt.Printf("wrap = '%s'\n", string(wrap))

		unwrap, err := RemoveAsciiArmor(wrap)
		panicOn(err)

		cv.So(unwrap, cv.ShouldResemble, data)
	})
}
