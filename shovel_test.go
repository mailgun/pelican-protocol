package pelican

import (
	cv "github.com/glycerine/goconvey/convey"
	"os"
	"testing"
	"time"
)

func TestShovelStops(t *testing.T) {

	cv.Convey("a shovel should stop when requested", t, func() {

		s := NewShovelPair()
		r, err := os.OpenFile("/dev/null", os.O_RDWR, 0600)
		panicOn(err)
		w, err := os.OpenFile("/dev/null", os.O_RDWR, 0600)
		panicOn(err)
		s.Start(w, r)
		<-s.Ready
		time.Sleep(time.Millisecond)
		s.Stop()
		cv.So(true, cv.ShouldResemble, true) // we should get here.
	})
}
