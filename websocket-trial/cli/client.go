package main

import (
	"fmt"
	"io"
	"log"

	"golang.org/x/net/websocket"
)

func main() {
	origin := "http://localhost/"
	url := "ws://localhost:12345/echo"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}
	hello := []byte("hello, world!\nhello two\n")
	m, err := ws.Write(hello)
	if err != nil {
		panic(err)
	}
	if m != len(hello) {
		fmt.Printf("not all bytes sent: only %d of %d\n", m, len(hello))
	}
	var msg = make([]byte, 512)
	var n int
	n, err = ws.Read(msg)
	switch err {
	case io.EOF:
	case nil:
	default:
		panic(err)
	}
	fmt.Printf("Received %d bytes: '%s'.\n", n, msg[:n])
}
