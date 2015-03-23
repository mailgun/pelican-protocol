package main

import (
	"fmt"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestMicroverseEchoWorks043(t *testing.T) {

	dn := NewBoundary("downstream")

	ab2lp := make(chan *tunnelPacket)
	lp2ab := make(chan *tunnelPacket)

	lp := NewLittlePoll(5*time.Second, dn, ab2lp, lp2ab)

	up := NewBoundary("upstream")
	ab := NewChaser(ChaserConfig{}, up.Generate, up.Absorb, ab2lp, lp2ab)

	dn.SetEcho(true)
	dn.Start()
	defer dn.Stop()

	lp.Start()
	defer lp.Stop()

	ab.Start()
	defer ab.Stop()

	up.SendEvery(1 * time.Second)
	up.Start()
	defer up.Stop()

	cv.Convey("Given a standalone LittlePoll and AB microverse, with one or more sends (generates) from Upstream, Downstream should see the same number of recevies (absorbs)", t, func() {

		up.hist.ShowHistory()
		time.Sleep(4500 * time.Millisecond)

		lp.ShowTmHistory()

		ab.ShowTmHistory()

		fmt.Printf("\n\n")
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

		dn := NewBoundary("downstream")

		ab2lp := make(chan *tunnelPacket)
		lp2ab := make(chan *tunnelPacket)

		lp := NewLittlePoll(5*time.Second, dn, ab2lp, lp2ab)

		up := NewBoundary("upstream")

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

// long-poll timeouts happen during idle no traffic situations.
func TestMicroverseLongPollTimeoutsCausePacketCirculationOtherwiseIdle042(t *testing.T) {

	dn := NewBoundary("downstream")

	ab2lp := make(chan *tunnelPacket)
	lp2ab := make(chan *tunnelPacket)

	longPollDur := 2 * time.Second
	lp := NewLittlePoll(longPollDur, dn, ab2lp, lp2ab)

	up := NewBoundary("upstream")
	ab := NewChaser(ChaserConfig{}, up.Generate, up.Absorb, ab2lp, lp2ab)

	dn.Start()
	defer dn.Stop()

	lp.Start()
	defer lp.Stop()

	ab.Start()
	defer ab.Stop()

	up.Start()
	defer up.Stop()

	cv.Convey("Given a standalone LittlePoll and AB microverse, even without any downstream/upstream Boundary traffic whatsoever, the AB and LP should exchange messages every long-poll timeout; and this should be the only traffic seen.", t, func() {

		// above set long-poll dur to 2 sec, so we should see 2 in this 5 second interval.
		sleep := 5 * time.Second
		time.Sleep(sleep)
		po("after %v sleep", sleep)

		lp.ShowTmHistory()
		ab.ShowTmHistory()

		fmt.Printf("\n\n")
		up.hist.ShowHistory()
		dn.hist.ShowHistory()

		uh := up.hist.GetHistory()
		dh := dn.hist.GetHistory()

		// neither upstream or downstream boundary should
		// have received any packets during idle time.
		cv.So(uh.CountGenerates(), cv.ShouldEqual, 0)
		cv.So(uh.CountAbsorbs(), cv.ShouldEqual, 0)
		cv.So(dh.CountGenerates(), cv.ShouldEqual, 0)
		cv.So(dh.CountAbsorbs(), cv.ShouldEqual, 0)

		alphaRTT := ab.home.GetAlphaRoundtripDurationHistory() // []time.Dur
		betaRTT := ab.home.GetBetaRoundtripDurationHistory()   //

		fmt.Printf("alpha RTT: '%v'\n", alphaRTT)
		fmt.Printf("beta RTT: '%v'\n", betaRTT)

		// we should have seen at least 2 idle (2 second timeout) trips happen
		// with preferrence for alpha
		cv.So(len(alphaRTT)+len(betaRTT), cv.ShouldBeGreaterThanOrEqualTo, 3)

		countLongPolls := 0

		tol := time.Duration(100 * time.Millisecond).Nanoseconds()
		for _, v := range alphaRTT {
			if int64Abs(v.Nanoseconds()-longPollDur.Nanoseconds()) < tol {
				countLongPolls++
			}
		}
		for _, v := range betaRTT {
			if int64Abs(v.Nanoseconds()-longPollDur.Nanoseconds()) < tol {
				countLongPolls++
			}
		}
		cv.So(countLongPolls, cv.ShouldBeGreaterThanOrEqualTo, 2)

	})
}

func int64Abs(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}
