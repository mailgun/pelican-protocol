package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
)

type KeyPayload struct {
	FirstName      string
	MiddleInitial  string
	LastName       string
	ValidEmail     string
	ValidSMSNumber string
	PublicKey      string

	// AcctUsername is the
	// sha1 hmac of PublicKey, prefixed with "p" (for pelican) and
	// encoded in base 36; A regex for it would be: "p[0-9a-z]{31}"
	// This is to conform to the requirements of linux usernames.
	// See man page for useradd; there is a 32 character limit,
	// and usernames must start with a letter and then contain
	// be only lowercase letters and digits. Underscores and
	// dashes are allowed too but we don't use them.
	AcctUsername string
}

func SendKeyPayload(p *KeyPayload) {

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf) // Will write to buf

	// Encode (send) some values.
	err := enc.Encode(p)
	if err != nil {
		panic(fmt.Sprintf("encode error:", err))
	}

	toWrite := len(buf.Bytes())

	n, err := io.Copy(os.Stdout, &buf)
	if err != nil {
		panic(err)
	}
	if n != int64(toWrite) {
		panic(fmt.Sprintf("did not write all of buf; n=%v\n, len(buf)=%v\n", n, toWrite))
	}
}

func main() {
	gob.Register(KeyPayload{})

	key := &KeyPayload{PublicKey: "0123456789abcdef-hello-public-key"}

	SendKeyPayload(key)
	fmt.Printf("\n done sending: '%#v'.\n", key)
}
