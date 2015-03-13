package pelicantun

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPelicanSocksProxyHomePolicyMakerForLongPolling014(t *testing.T) {
	cv.Convey("When we start the PelicanSocksProxy, the long-polling implementation over two sockets should prefer to have one connection long-polling to the server (for the server to reply with data on) when the other connection is 'at home' waiting for the client to send data on.\n", t, func() {

		poller := NewLongPoller(Addr{})

		//		psp := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
		//		psp.Start()
		//		defer psp.Stop()

	})
}
