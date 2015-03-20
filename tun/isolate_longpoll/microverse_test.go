package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"testing"
	"time"
)

func TestMicroverseSimABandLittlePollAlone043(t *testing.T) {

	cv.Convey("Given a standalone LittlePoller and AB microverse, with no client/server traffic, the system should only transmit at long-poll timeouts", t, func() {

		dn := NewDownstream()

		ab2lp := make(chan []byte)
		lp2ab := make(chan []byte)

		lp := NewLittlePoll(15*time.Second, dn, ab2lp, lp2ab)

		up := NewUpstream()
		ab := NewChaser(ChaserConfig{}, up.Generate, up.Absorb, ab2lp, lp2ab)

		// keep compiler happy
		po("lp: %p, ab: %p, ab2lp: %p, lp2ab: %p, up: %p, dn:%p", lp, ab, &ab2lp, &lp2ab, &up, &dn)

		dn.Start()
		defer dn.Stop()

		lp.Start()
		defer lp.Stop()

		ab.Start()
		defer ab.Stop()

		up.Start()
		defer up.Stop()

		fmt.Printf("so we should not deadlock on shutdown after this")
		//cv.So(string(pack2.respdup.Bytes()), cv.ShouldResemble, msg+"1")
	})
}

func TestMicroverseShutdownCleanly044(t *testing.T) {

	cv.Convey("Given a standalone LittlePoller and AB microverse, it should startup and shutdown without deadlock", t, func() {

		dn := NewDownstream()

		ab2lp := make(chan []byte)
		lp2ab := make(chan []byte)

		lp := NewLittlePoll(15*time.Second, dn, ab2lp, lp2ab)

		up := NewUpstream()
		ab := NewChaser(ChaserConfig{}, up.Generate, up.Absorb, ab2lp, lp2ab)

		// keep compiler happy
		po("lp: %p, ab: %p, ab2lp: %p, lp2ab: %p, up: %p, dn:%p", lp, ab, &ab2lp, &lp2ab, &up, &dn)

		dn.Start()
		lp.Start()
		ab.Start()
		up.Start()

		//		dn.Stop()
		//		lp.Stop()
		//		ab.Stop()
		//		up.Stop()

		up.Stop()
		ab.Stop()
		lp.Stop()
		dn.Stop()

		fmt.Printf("so we should not deadlock on shutdown after this")
		cv.So(true, cv.ShouldEqual, true)
	})
}
