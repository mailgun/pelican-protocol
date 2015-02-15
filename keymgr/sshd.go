package main

import (
	"crypto/rsa"
)

type Sshd struct{}

func NewSshd(port int, rsa *rsa.PrivateKey) (*Sshd, error) {

	return nil, nil
}

func (s *Sshd) Start() {}
func (s *Sshd) Stop()  {}
