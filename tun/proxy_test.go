package pelicantun

import (
	"fmt"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestSocksProxyTalksToReverseProxy002(t *testing.T) {

	fmt.Printf("\n before NewReverseProxy\n")
	rev := NewReverseProxy(ReverseProxyConfig{})
	fmt.Printf("\n done with NewReverseProxy\n")
	rev.Start()

	fmt.Printf("\n done with rev.Start(), rev.Cfg.Listen.IpPort = '%v'\n", rev.Cfg.Listen.IpPort)

	revListen := rev.Cfg.Listen
	if !PortIsBound(rev.Cfg.Listen.IpPort) {
		panic("rev proxy not up")
	}

	fwd := NewPelicanSocksProxy(PelicanSocksProxyConfig{
		Dest: revListen,
	})

	cv.Convey("\n Given a ForwardProxy and a ReverseProxy, they should communicate over http\n\n", t, func() {

		fmt.Printf("\n after fwd := NewPelicanSocksProxy, before calling fwd.Start() \n")

		fwd.Start()
		fmt.Printf("fwd proxy chose listen port = '%#v'\n", fwd.Cfg)

		fmt.Printf("\n both fwd and rev started! \n")
		cv.So(PortIsBound(fwd.Cfg.Listen.IpPort), cv.ShouldEqual, true)

		po("\n fetching url from %v\n", fwd.Cfg.Listen.IpPort)
		by, err := FetchUrl("http://" + fwd.Cfg.Listen.IpPort + "/")
		cv.So(err, cv.ShouldEqual, nil)
		cv.So(by, cv.ShouldResemble, []byte("some output"))

		fwd.Stop()
		cv.So(PortIsBound(fwd.Cfg.Listen.IpPort), cv.ShouldEqual, false)

		rev.Stop()
		cv.So(PortIsBound(rev.Cfg.Listen.IpPort), cv.ShouldEqual, false)

		by, err = FetchUrl("http://" + rev.Cfg.Listen.IpPort + "/")
		cv.So(err, cv.ShouldEqual, nil)
		cv.So(by, cv.ShouldResemble, []byte("some output"))
	})

	fmt.Printf("\n done with TestSocksProxyTalksToReverseProxy002()\n")
}
