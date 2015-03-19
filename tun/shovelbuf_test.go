package pelicantun

import (
	"bytes"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestShovelStops007(t *testing.T) {

	cv.Convey("a Shovel should stop when requested", t, func() {

		s := NewShovel()

		a := NewMockRwc([]byte("hello_from_a"))
		b := NewMockRwc([]byte("hello_from_b"))

		s.Start(b, a, "b<-a")
		<-s.Ready
		time.Sleep(100 * time.Millisecond)
		s.Stop()
		cv.So(b.sink.String(), cv.ShouldResemble, "hello_from_a")
		cv.So(a.sink.String(), cv.ShouldResemble, "")
	})

	cv.Convey("a ShovelPair should stop when requested", t, func() {

		s := NewShovelPair()

		a := NewMockRwc([]byte("hello_from_a"))
		b := NewMockRwc([]byte("hello_from_b"))

		s.Start(a, b, "a<-b", "b->a")
		<-s.Ready
		time.Sleep(1 * time.Millisecond)
		s.Stop()
		cv.So(b.sink.String(), cv.ShouldResemble, "hello_from_a")
		cv.So(a.sink.String(), cv.ShouldResemble, "hello_from_b")
	})

}

type MockRwc struct {
	src  *bytes.Buffer
	sink *bytes.Buffer
}

func NewMockRwc(src []byte) *MockRwc {
	return &MockRwc{
		src:  bytes.NewBuffer(src),
		sink: bytes.NewBuffer(nil),
	}
}

func (m *MockRwc) Read(p []byte) (n int, err error) {
	return m.src.Read(p)
}

func (m *MockRwc) Write(p []byte) (n int, err error) {
	return m.sink.Write(p)
}

func (m *MockRwc) Close() error {
	return nil
}
