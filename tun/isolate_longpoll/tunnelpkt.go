package main

import (
	"bytes"
	"net/http"
)

type tunnelPacket struct {
	resp    http.ResponseWriter
	respdup *bytes.Buffer // duplicate resp here, to enable testing

	request *http.Request
	body    []byte
	key     string // separate from body
	done    chan bool
}
