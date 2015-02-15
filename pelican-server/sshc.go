package main

import (
	"fmt"
)

type SshClient struct {
}

func NewSshClient(host string, port int) (*SshClient, error) {
	fmt.Printf("NewSshClient called\n")
	return nil, nil
}

func (c *SshClient) Handshake() error {
	return nil
}
