package pelicantun

import (
	"net"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
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

	rw := NewRW(conn, upReadToDnWrite, dnReadToUpWrite)
	rw.Start()
	defer rw.Stop()

	m1 := []byte("yippeeee!")
	upReadToDnWrite <- m1

	m2 := <-dnReadToUpWrite

	if string(m1) != string(m2) {
		panic("echo server not echoing")
	}

	po("m1 m2 compare looks good\n")

	cv.Convey("When we start a RW that turns a net.Conn connection into a pair of channels, then reads and writes to/from the netConn should succeed and notice if the connection is dropped.", t, func() {
		cv.So(m1, cv.ShouldResemble, m2)
	})
}
