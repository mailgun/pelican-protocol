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

			s := NewChaser()
			cv.So(s.home.alphaHome, cv.ShouldEqual, true)
			cv.So(s.home.betaHome, cv.ShouldEqual, true)

			// don't do s.Start(), since that will begin the automatic
			// alpha and beta making transitions on their own; instead
			// we test the s.home logic in isolation here.

			s.home.Start()
			defer s.home.Stop()
			cv.So(<-s.home.shouldAlphaGoNow, cv.ShouldEqual, true)
			cv.So(<-s.home.shouldBetaGoNow, cv.ShouldEqual, false)

			s.home.alphaDepartsHome <- true
			cv.So(<-s.home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-s.home.shouldBetaGoNow, cv.ShouldEqual, false)

			s.home.alphaArrivesHome <- true
			cv.So(<-s.home.shouldAlphaGoNow, cv.ShouldEqual, true)
			cv.So(<-s.home.shouldBetaGoNow, cv.ShouldEqual, false)

			s.home.betaDepartsHome <- true
			cv.So(<-s.home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-s.home.shouldBetaGoNow, cv.ShouldEqual, false)

			s.home.alphaDepartsHome <- true
			cv.So(<-s.home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-s.home.shouldBetaGoNow, cv.ShouldEqual, false)

			s.home.betaArrivesHome <- true
			cv.So(<-s.home.shouldAlphaGoNow, cv.ShouldEqual, false)
			cv.So(<-s.home.shouldBetaGoNow, cv.ShouldEqual, false)

		})
	})
}
