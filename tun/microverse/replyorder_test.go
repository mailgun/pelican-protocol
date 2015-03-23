package main

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestReplyMisorderingsAreCorrected048(t *testing.T) {

	dn := NewBoundary("downstream")

	ab2lp := make(chan *tunnelPacket)
	lp2ab := make(chan *tunnelPacket)

	longPollDur := 500 * time.Millisecond
	lp := NewLittlePoll(longPollDur, dn, ab2lp, lp2ab)
	lp.SetReplySerialReordering([]int64{5, 1, 3, 2, 4})

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

	cv.Convey("Previous test was for request order, this is for reply order: Given that replies can arrive out of order (while the two http connection race), we should detect this and re-order replies into sequence.", t, func() {
		// test reply reorder:

		up.Gen([]byte{'5'})
		up.Gen([]byte{'1'})
		up.Gen([]byte{'3'})
		up.Gen([]byte{'2'})
		up.Gen([]byte{'4'})
		time.Sleep(10 * time.Millisecond)

		uh := up.hist.GetHistory()

		uh.ShowHistory()

		cv.So(uh.CountAbsorbs(), cv.ShouldEqual, 3)
		cv.So(uh.CountGenerates(), cv.ShouldEqual, 5)

		cv.So(string(uh.absorbHistory[0].what), cv.ShouldEqual, "1")
		cv.So(string(uh.absorbHistory[1].what), cv.ShouldEqual, "23")
		cv.So(string(uh.absorbHistory[2].what), cv.ShouldEqual, "45")

	})
}

func ReplyToAbHelper(ch chan *tunnelPacket, serialNum int64) *tunnelPacket {
	c := NewMockResponseWriter()

	body := []byte(fmt.Sprintf("%d", serialNum))
	reqBody := bytes.NewBuffer(body)
	r, err := http.NewRequest("POST", "http://example.com/", reqBody)
	panicOn(err)

	pack := &tunnelPacket{
		resp:    c,
		respdup: new(bytes.Buffer),
		request: r,
		done:    make(chan bool),
		key:     "longpoll_test_key",
		SerReq: SerReq{
			reqBody:       body,
			requestSerial: serialNum,
		},
	}

	ch <- pack

	// service replies in a timely fashion, or
	// detect lack of re-ordering.
	go func() {
		po("sent serial number %d", serialNum)

		select {
		case <-pack.done:
			// good
			po("got back pack.done for serial %d", serialNum)
		case <-time.After(10 * time.Second):
			po("helper reader timeout for serial %d", serialNum)
		}
	}()

	return pack
}
