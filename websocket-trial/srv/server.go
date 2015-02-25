package main

import (
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/websocket"
)

// Echo the data received on the WebSocket.
func EchoServer(ws *websocket.Conn) {
	fmt.Printf("EchoServer callback here happened.\n")

	//	io.Copy(ws, ws)

	p := make([]byte, 500)
	n, err := ws.Read(p)
	fmt.Printf("done with read\n")
	switch err {
	case nil:
	case io.EOF:
	default:
		fmt.Printf("see err = %v\n", err)
		panic(err)
	}
	fmt.Printf("done with switch\n")

	fmt.Printf("echo server read %d bytes: '%s'\n", n, string(p[:n]))
	p = p[:n]
	r := append([]byte("server-saw:"), p...)
	ws.Write(r)

}

// This example demonstrates a trivial echo server.
func main() {
	http.Handle("/echo", websocket.Handler(EchoServer))
	addr := ":12345"
	fmt.Printf("server handling /echo, listening on '%s'\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
