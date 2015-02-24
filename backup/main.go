package main

import (
	"fmt"

	pp "github.com/mailgun/pelican-protocol"
)

func main() {
	cfg := pp.ReadMailgunConfig("../.mg.cfg.json")
	body := "test body"
	status, id, err := cfg.SendPPPBackupMailgunMail(body)
	fmt.Printf("statusMsg = '%s', id = '%s', err ='%v'\n", status, id, err)
}
