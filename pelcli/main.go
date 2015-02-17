package main

import "fmt"

func main() {
	p := NewPelicanClient()
	p.Start()
	<-p.Done
	fmt.Printf("[PelicanClient done.]\n")
}
