package main

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
	pelican "github.com/mailgun/pelican-protocol"
)

func TestSshHandshake(t *testing.T) {
	rsa_file := "./id_rsa"

	rsa, err := pelican.GenRsaKeyPair(rsa_file, 4096)
	panicOn(err)

	sshd, err := NewSshd(2022, rsa)
	panicOn(err)
	sshd.Start()
	defer sshd.Stop()

	sshcli, err := NewSshClient("localhost", 2022)
	panicOn(err)

	cv.Convey("our pelican-server should accept SSH handshake from ssh clients", t, func() {
		err := sshcli.Handshake()
		cv.So(err, cv.ShouldEqual, nil)
	})

}
