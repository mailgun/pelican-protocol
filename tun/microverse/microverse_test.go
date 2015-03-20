package main

import (
	"fmt"
	cv "github.com/glycerine/goconvey/convey"
	"testing"
	"time"
)

func TestMicroverseSimABandLittlePollAlone043(t *testing.T) {

	dn := NewDownstream()

	ab2lp := make(chan []byte)
	lp2ab := make(chan []byte)

	lp := NewLittlePoll(5*time.Second, dn, ab2lp, lp2ab)

	up := NewUpstream()
	ab := NewChaser(ChaserConfig{}, up.Generate, up.Absorb, ab2lp, lp2ab)

	dn.Start()
	defer dn.Stop()

	lp.Start()
	defer lp.Stop()

	ab.Start()
	defer ab.Stop()

	up.SendEvery(1 * time.Second)
	up.Start()
	defer up.Stop()

	//	cv.Convey("Given a standalone LittlePoll and AB microverse, with no client/server traffic, the system should only transmit at long-poll timeouts", t, func() {

	cv.Convey("Given a standalone LittlePoll and AB microverse, with one or more sends (generates) from Upstream, Downstream should see the same number of recevies (absorbs)", t, func() {

		up.hist.ShowHistory()
		time.Sleep(20 * time.Second)

		lp.ShowTmHistory()

		ab.ShowTmHistory()

		up.hist.ShowHistory()
		dn.hist.ShowHistory()

		uh := up.hist.GetHistory()
		dh := dn.hist.GetHistory()

		cv.So(len(uh.generateHistory), cv.ShouldBeGreaterThan, 1)
		cv.So(len(dh.absorbHistory), cv.ShouldBeGreaterThan, 1)

		po("uh.CountGenerates() = %v", uh.CountGenerates())
		po("dh.CountGenerates() = %v", dh.CountGenerates())
		po("uh.CountAbsorbs() = %v", uh.CountAbsorbs())
		po("dh.CountAbsorbs() = %v", dh.CountAbsorbs())

		// sent by upstream should be received by downstream
		cv.So(uh.CountGenerates(), cv.ShouldEqual, dh.CountAbsorbs())

		// sent by downstream should be received by upstream
		cv.So(dh.CountGenerates(), cv.ShouldEqual, uh.CountAbsorbs())

	})
}

func TestMicroverseShutdownCleanly044(t *testing.T) {

	cv.Convey("Given a standalone LittlePoller and AB microverse, it should startup and shutdown without deadlock", t, func() {

		dn := NewDownstream()

		ab2lp := make(chan []byte)
		lp2ab := make(chan []byte)

		lp := NewLittlePoll(15*time.Second, dn, ab2lp, lp2ab)

		up := NewUpstream()
		up.SendEvery(24 * time.Hour)
		ab := NewChaser(ChaserConfig{}, up.Generate, up.Absorb, ab2lp, lp2ab)

		dn.Start()
		lp.Start()
		ab.Start()
		up.Start()

		// either order works
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
