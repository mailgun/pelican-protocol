package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	addr := "localhost:1234"
	fmt.Printf("listening on addr '%s'\n", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	fmt.Printf("in handleConnection\n")
	for i := 0; i < 2; i++ {
		req, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Printf("dest got req = '%s'\n", string(req))
		_, err = fmt.Fprintf(conn, "789_and_i=%d\n", i)
		if err != nil {
			panic(err)
		}
		fmt.Printf("dest done with sending reply starting with 789.\n")
	}
	/*
		err = conn.Close()
		if err != nil {
			panic(err)
		}
	*/
}
