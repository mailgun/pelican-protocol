package main

import (
	"bytes"
	"net/http"
	"time"
)

type tunnelPacket struct {
	resp    http.ResponseWriter
	respdup *bytes.Buffer // duplicate resp here, to enable testing

	request *http.Request
	key     string // separate from body
	done    chan bool

	replySerial int64 // order the replies by serial number. Empty replies get serial number -1.

	SerReq
}

func NewTunnelPacket() *tunnelPacket {
	p := &tunnelPacket{
		done: make(chan bool),
	}
	return p
}

func ToSerReq(pack *tunnelPacket) *SerReq {
	return &SerReq{
		reqBody:       pack.reqBody,
		requestSerial: pack.requestSerial,
		tm:            time.Now(),
	}
}

type SerReq struct {
	reqBody       []byte
	requestSerial int64 // order the sends with content by serial number
	tm            time.Time
}

type SerResp struct {
	response       []byte
	responseSerial int64 // order the sends with content by serial number
	tm             time.Time
}
