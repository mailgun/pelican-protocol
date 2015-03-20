package main

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestLongPollerWorksStandAlone041(t *testing.T) {

	cv.Convey("Given a LongPoller stand alone, it should pass bytes to the downstream server, and return replies from downstream to upstream", t, func() {

		srv := NewEchoServer(Addr{}, true)
		srv.Start()
		defer srv.Stop()

		s := NewLongPoller(srv.Listen, 15*time.Second)
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
		cv.So(string(pack.respdup.Bytes()), cv.ShouldResemble, string(body)+"0")

		msg := "yaba daba"
		pack2 := &tunnelPacket{
			resp:    c,
			respdup: new(bytes.Buffer),
			request: r,
			body:    []byte(msg),
			done:    make(chan bool),
			key:     "longpoll_test_key",
		}

		s.ClientPacketRecvd <- pack2
		<-pack2.done
		po("got back: '%s'", pack2.respdup.Bytes())

		cv.So(string(pack2.respdup.Bytes()), cv.ShouldResemble, msg+"1")
	})
}
