package pelicantun

import (
	"net"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
	"github.com/glycerine/rbuf"
)

func TestRW017(t *testing.T) {

	upReadToDnWrite := make(chan []byte)
	dnReadToUpWrite := make(chan []byte)

	echo := NewEchoServer(Addr{Ip: "127.0.0.1"})
	echo.Start()
	defer echo.Stop()

	conn, err := net.Dial("tcp", echo.Listen.IpPort)
	if err != nil {
		panic(err)
	}

	// use buffer of size 3 to simulate needing multiple reads
	rw := NewRW(conn, upReadToDnWrite, dnReadToUpWrite, 3)
	rw.Start()
	defer rw.Stop()

	m1 := []byte("yippeeee! yes!")
	upReadToDnWrite <- m1

	// accumulate reads into here
	circ := rbuf.NewFixedSizeRingBuf(200)

for10msec:
	for {
		select {
		case m2 := <-dnReadToUpWrite:
			circ.Write(m2)
		case <-time.After(10 * time.Millisecond):
			break for10msec
		}
	}

	s1 := string(m1)
	m2 := circ.Bytes()
	s2 := string(m2)
	if s1 != s2 {
		po("m1 = '%s' while m2 = '%s'\n", s1, s2)
		panic("echo server not echoing")
	}

	po("m1 m2 compare looks good\n")

	circ.Reset()

	a1 := "more 0123"
	upReadToDnWrite <- []byte(a1)

forb:
	for {
		select {
		case b := <-dnReadToUpWrite:
			circ.Write(b)
		case <-time.After(10 * time.Millisecond):
			break forb
		}
	}

	b1 := string(circ.Bytes())

	cv.Convey("When we start a RW that turns a net.Conn connection into a pair of channels, then reads and writes to/from the netConn should succeed and notice if the connection is dropped.", t, func() {
		cv.So(s1, cv.ShouldResemble, s2)
		cv.So(a1, cv.ShouldResemble, b1)
	})
}
