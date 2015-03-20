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

	up.Start()
	defer up.Stop()

	cv.Convey("Given a standalone LittlePoll and AB microverse, with no client/server traffic, the system should only transmit at long-poll timeouts", t, func() {

		time.Sleep(20 * time.Second)
		lp.ShowTmHistory()

		up.showHistory()
		dn.showHistory()
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
