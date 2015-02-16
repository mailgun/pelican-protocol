package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp/armor"
	"io"
)

func WrapInAsciiArmor(data []byte) ([]byte, error) {

	var out bytes.Buffer
	w, err := armor.Encode(&out, "PELICAN-PROTOCOL", nil)
	panicOn(err)

	goaln := len(data)
	n, err := w.Write(data)
	panicOn(err)
	if n != goaln {
		panic(fmt.Errorf("WrapInAsciiArmor tried to write %d bytes, but only wrote %d", goaln, n))
	}
	err = w.Close()
	panicOn(err)

	return out.Bytes(), nil
}

func RemoveAsciiArmor(data []byte) ([]byte, error) {
	var p *armor.Block
	var err error
	p, err = armor.Decode(bytes.NewBuffer(data))
	if err == io.EOF {
		return []byte{}, fmt.Errorf("ascii armor not found")
	}
	var by bytes.Buffer
	_, err = io.Copy(&by, p.Body)
	return by.Bytes(), err
}
