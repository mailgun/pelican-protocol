package main

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestRequestOneMisorderingIsCorrected046(t *testing.T) {

	dn := NewBoundary("downstream")

	ab2lp := make(chan *tunnelPacket)
	lp2ab := make(chan *tunnelPacket)

	longPollDur := 2 * time.Second
	lp := NewLittlePoll(longPollDur, dn, ab2lp, lp2ab)

	dn.Start()
	defer dn.Stop()

	lp.Start()
	defer lp.Stop()

	cv.Convey("Given that requests can arrive out of order (while the two http connection race), we should detect this and re-order both requests into sequence.", t, func() {

		// test *request* reorder alone (distinct from *reply* reordering):

		c2 := NewMockResponseWriter()

		// First send 2 in requestSerial 2, then send 1 in request serial 1,
		// and we should see them arrive 1 then 2 due to the re-ordering logic.
		//
		body2 := []byte("2")
		reqBody2 := bytes.NewBuffer(body2)
		r2, err := http.NewRequest("POST", "http://example.com/", reqBody2)
		panicOn(err)
		pack2 := &tunnelPacket{
			resp:    c2,
			respdup: new(bytes.Buffer),
			request: r2,
			done:    make(chan bool),
			key:     "longpoll_test_key",
			SerReq: SerReq{
				reqBody:       body2,
				requestSerial: 2,
			},
		}

		lp.ab2lp <- pack2

		c1 := NewMockResponseWriter()

		body1 := []byte("1")
		reqBody1 := bytes.NewBuffer(body1)
		r1, err := http.NewRequest("POST", "http://example.com/", reqBody1)
		panicOn(err)

		pack1 := &tunnelPacket{
			resp:    c1,
			respdup: new(bytes.Buffer),
			request: r1,
			done:    make(chan bool),
			key:     "longpoll_test_key",
			SerReq: SerReq{
				reqBody:       body1,
				requestSerial: 1,
			},
		}

		lp.ab2lp <- pack1

		// we won't get pack1 back immediately, but we will get pack2 back,
		// since it was sent first.
		/*
			select {
			case <-pack1.done:
				// good
				po("got back pack1.done")
			case <-time.After(1 * time.Second):
				dn.hist.ShowHistory()
				panic("should have had pack1 be done by now -- if re-ordering is in effect")
			}
		*/

		select {
		case <-pack2.done:
			// good
			po("got back pack2.done")
		case <-time.After(1 * time.Second):
			dn.hist.ShowHistory()
			panic("should have had pack2 be done by now -- if re-ordering is in effect")
		}

		//po("pack1 got back: '%s'", pack1.respdup.Bytes())
		po("pack2 got back: '%s'", pack2.respdup.Bytes())

		dh := dn.hist.GetHistory()
		dh.ShowHistory()

		cv.So(dh.CountAbsorbs(), cv.ShouldEqual, 1)
		cv.So(dh.CountGenerates(), cv.ShouldEqual, 0)

		cv.So(string(dh.absorbHistory[0].what), cv.ShouldEqual, "12")
	})
}

func TestTwoRequestMisorderingsAreCorrected047(t *testing.T) {

	dn := NewBoundary("downstream")

	ab2lp := make(chan *tunnelPacket)
	lp2ab := make(chan *tunnelPacket)

	longPollDur := 2 * time.Second
	lp := NewLittlePoll(longPollDur, dn, ab2lp, lp2ab)

	dn.Start()
	defer dn.Stop()

	lp.Start()
	defer lp.Stop()

	// service sends on lp2ab without setting up a full ab
	shutdown := make(chan bool)
	go func() {
		for {
			select {
			case got := <-lp2ab:
				po("lp2ab reader got packet: '%s'", string(got.respdup.Bytes()))
			case <-shutdown:
				return
			}
		}
	}()
	defer close(shutdown)

	cv.Convey("Given that requests can arrive out of order (while the two http connection race), we should detect this and re-order both requests into sequence.", t, func() {

		// test *request* reorder alone (distinct from *reply* reordering):

		// send serial numbers out of order

		pack3 := SendHelper(lp.ab2lp, 3)
		pack2 := SendHelper(lp.ab2lp, 2)
		pack1 := SendHelper(lp.ab2lp, 1) // hung here

		po("pack1 got back: '%s'", pack1.respdup.Bytes())
		po("pack2 got back: '%s'", pack2.respdup.Bytes())
		po("pack3 got back: '%s'", pack3.respdup.Bytes())

		time.Sleep(4 * time.Second)

		dh := dn.hist.GetHistory()
		dh.ShowHistory()

		cv.So(dh.CountAbsorbs(), cv.ShouldEqual, 1)
		cv.So(dh.CountGenerates(), cv.ShouldEqual, 0)

		cv.So(string(dh.absorbHistory[0].what), cv.ShouldEqual, "123")

		po("done with order_test")
	})
}

func SendHelper(ch chan *tunnelPacket, serialNum int64) *tunnelPacket {
	c2 := NewMockResponseWriter()

	// First send 2 in requestSerial 2, then send 1 in request serial 1,
	// and we should see them arrive 1 then 2 due to the re-ordering logic.
	//
	body2 := []byte(fmt.Sprintf("%d", serialNum))
	reqBody2 := bytes.NewBuffer(body2)
	r2, err := http.NewRequest("POST", "http://example.com/", reqBody2)
	panicOn(err)

	pack2 := &tunnelPacket{
		resp:    c2,
		respdup: new(bytes.Buffer),
		request: r2,
		done:    make(chan bool),
		key:     "longpoll_test_key",
		SerReq: SerReq{
			reqBody:       body2,
			requestSerial: serialNum,
		},
	}

	ch <- pack2

	// service replies in a timely fashion, or
	// detect lack of re-ordering.
	go func() {
		po("sent serial number %d", serialNum)

		select {
		case <-pack2.done:
			// good
			po("got back pack.done for serial %d", serialNum)
		case <-time.After(10 * time.Second):
			po("helper reader timeout for serial %d", serialNum)
		}
	}()

	return pack2
}
