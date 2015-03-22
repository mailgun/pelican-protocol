package main

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestMisorderingsAreCorrected046(t *testing.T) {

	dn := NewBoundary("downstream")

	ab2lp := make(chan *tunnelPacket)
	lp2ab := make(chan []byte)

	longPollDur := 2 * time.Second
	lp := NewLittlePoll(longPollDur, dn, ab2lp, lp2ab)

	//	up := NewBoundary("upstream")

	//	ab := NewChaser(ChaserConfig{}, up.Generate, up.Absorb, ab2lp, lp2ab)

	dn.Start()
	defer dn.Stop()

	lp.Start()
	defer lp.Stop()

	//	ab.Start()
	//	defer ab.Stop()

	//	up.Start()
	//	defer up.Stop()

	cv.Convey("Given that requests or replies can arrive out of order (while the two http connection race), we should detect this and re-order both requests and replies into sequence.", t, func() {

		c2 := NewMockResponseWriter()

		// First send 2 in requestSerial 2, then send 1 in request serial 1,
		// and we should see them arrive 1 then 2 due to the re-ordering logic.
		//
		body2 := []byte("2")
		reqBody2 := bytes.NewBuffer(body2)
		r2, err := http.NewRequest("POST", "http://example.com/", reqBody2)
		panicOn(err)
		pack2 := &tunnelPacket{
			resp:          c2,
			respdup:       new(bytes.Buffer),
			request:       r2,
			reqBody:       body2,
			done:          make(chan bool),
			key:           "longpoll_test_key",
			requestSerial: 2,
		}

		lp.ab2lp <- pack2

		/*
			lp.ClientPacketRecvd <- pack
			<-pack.done
			po("got back: '%s'", pack.respdup.Bytes())

			// we should get back the body
			cv.So(string(pack.respdup.Bytes()), cv.ShouldResemble, string(body)+"0")
		*/

		c1 := NewMockResponseWriter()

		body1 := []byte("1")
		reqBody1 := bytes.NewBuffer(body1)
		r1, err := http.NewRequest("POST", "http://example.com/", reqBody1)
		panicOn(err)

		pack1 := &tunnelPacket{
			resp:          c1,
			respdup:       new(bytes.Buffer),
			request:       r1,
			reqBody:       body1,
			done:          make(chan bool),
			key:           "longpoll_test_key",
			requestSerial: 1,
		}

		lp.ab2lp <- pack1
		<-pack1.done
		<-pack2.done

		po("pack1 got back: '%s'", pack1.respdup.Bytes())
		po("pack2 got back: '%s'", pack2.respdup.Bytes())

		dh := dn.hist.GetHistory()

		cv.So(len(dh.absorbHistory), cv.ShouldEqual, 2)
		cv.So(len(dh.generateHistory), cv.ShouldEqual, 0)

		cv.So(dh.absorbHistory[0].what, cv.ShouldEqual, "1")
		cv.So(dh.absorbHistory[1].what, cv.ShouldEqual, "2")
	})
}