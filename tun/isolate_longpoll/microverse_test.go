package main

import (
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestMicroverseSimABandLittlePollAlone043(t *testing.T) {

	cv.Convey("Given a standalone LittlePoller and AB microverse, with no client/server traffic, the system should only transmit at long-poll timeouts", t, func() {

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

		//cv.So(string(pack2.respdup.Bytes()), cv.ShouldResemble, msg+"1")
	})
}
