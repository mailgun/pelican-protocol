package main

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

// print out shortcut
func po(format string, a ...interface{}) {
	if Verbose {
		TSPrintf("\n\n"+format+"\n\n", a...)
	}
}

type tunnelPacket struct {
	resp    http.ResponseWriter
	respdup *bytes.Buffer // duplicate resp here, to enable testing

	request *http.Request
	body    []byte
	key     string // separate from body
	done    chan bool
}

func TestLongPollerWorksStandAlone041(t *testing.T) {

	cv.Convey("Given a LongPoller stand alone, it should pass bytes to the downstream server, and return replies from downstream to upstream", t, func() {

		srv := NewBcastServer(Addr{})
		srv.Start()
		defer srv.Stop()

		s := NewLongPoller(srv.Listen, 5*time.Second)
		s.Start()

		c := NewMockResponseWriter()

		reqBody := bytes.NewBufferString("my test body for longpoll")
		r, err := http.NewRequest("POST", "http://example.com/", reqBody)
		panicOn(err)
		body := []byte("longpoll_test_body")
		pack := &tunnelPacket{
			resp:    c,
			respdup: new(bytes.Buffer),
			request: r,
			body:    body, // body no longer includes key of KeyLen in prefix
			done:    make(chan bool),
			key:     "longpoll_test_key",
		}

		s.ClientPacketRecvd <- pack

		<-pack.done

		po("got back: '%s'", pack.respdup.Bytes())

		// we should get back the body
		cv.So(pack.respdup.Bytes(), cv.ShouldResemble, body)

	})
}
