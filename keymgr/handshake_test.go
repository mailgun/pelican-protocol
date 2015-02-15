package main

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestSshHandshake(t *testing.T) {
	rsa_file := "./id_rsa"

	rsa, err := GenRsaKeyPair(rsa_file, 1024)
	panicOn(err)

	sshd, err := NewSshd(2022, rsa)
	panicOn(err)
	sshd.Start()
	defer sshd.Stop()

	sshcli, err := NewSshClient("localhost", 2022)
	panicOn(err)

	cv.Convey("our keymgr should accept SSH handshake from ssh clients", t, func() {
		err := sshcli.Handshake()
		cv.So(err, cv.ShouldEqual, nil)
	})

}
