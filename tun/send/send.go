package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	//addr := "localhost:1234"
	addr := "localhost:2222"
	fmt.Printf("about to dial/send on '%s'\n", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 2; i++ {
		fmt.Fprintf(conn, "0123456\n")
		status, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Printf("got status = '%s'\n", string(status))
	}
}
