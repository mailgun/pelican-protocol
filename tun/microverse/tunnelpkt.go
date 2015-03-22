package main

import (
	"bytes"
	"net/http"
)

type tunnelPacket struct {
	resp    http.ResponseWriter
	respdup *bytes.Buffer // duplicate resp here, to enable testing

	request *http.Request
	reqBody []byte
	key     string // separate from body
	done    chan bool

	requestSerial int64 // order the sends with content by serial number
	replySerial   int64 // order the replies by serial number. Empty replies get serial number -1.
}
