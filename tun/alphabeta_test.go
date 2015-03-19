package pelicantun

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

//		psp := NewPelicanSocksProxy(PelicanSocksProxyConfig{})
//		psp.Start()
//		defer psp.Stop()

func TestPelicanSocksProxyHomePolicyMakerForLongPolling014(t *testing.T) {
	cv.Convey("When we start the PelicanSocksProxy, the long-polling implementation over two sockets should prefer to have one connection long-polling to the server (for the server to reply with data on) when the other connection is 'at home' waiting for the client to send data on.\n", t, func() {

		cv.Convey("Initiatially both alpha and beta txn are at home, and the home controller should indicate that beta should stay and alpha should go\n", func() {

			home := NewClientHome()
			cv.So(home.alphaHome, cv.ShouldEqual, true)
			cv.So(home.betaHome, cv.ShouldEqual, true)

			// don't do s.Start(), since that will begin the automatic
			// alpha and beta making transitions on their own; instead
			// we test the s.home logic in isolation here.

			home.Start()
			defer home.Stop()
			cv.So(<-home.shouldAlphaGoNow, cv.ShouldEqual, true)
			cv.So(<-home.shouldBetaGoNow, cv.ShouldEqual, false)

			home.alphaDepartsHome <- true
			cv.So(<-home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-home.shouldBetaGoNow, cv.ShouldEqual, false)

			home.alphaArrivesHome <- true
			cv.So(<-home.shouldAlphaGoNow, cv.ShouldEqual, true)
			cv.So(<-home.shouldBetaGoNow, cv.ShouldEqual, false)

			home.betaDepartsHome <- true
			cv.So(<-home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-home.shouldBetaGoNow, cv.ShouldEqual, false)

			home.alphaDepartsHome <- true
			cv.So(<-home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-home.shouldBetaGoNow, cv.ShouldEqual, false)

			home.betaArrivesHome <- true
			cv.So(<-home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-home.shouldBetaGoNow, cv.ShouldEqual, false)

		})
	})
}

func TestPelicanSocksProxyChaserHasSmallLatency015(t *testing.T) {
	cv.Convey("When we start the PelicanSocksProxy, the client end long-poller called 'Chaser' should provide small latency of desire-to-send to time-to-send to the client and the server.\n", t, func() {

		// simulate two Chaser's talking to each other

	})
}
