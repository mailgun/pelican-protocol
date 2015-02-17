package pelican

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp/armor"
	"io"
)

// WrapInAsciiArmor returns data as an Ascii-armored
// text block, using the type PELICAN-PROTOCOL-FORMAT.
// This makes its contents more resilliant to being forwarded
// through email.
func WrapInAsciiArmor(data []byte) ([]byte, error) {

	var out bytes.Buffer
	w, err := armor.Encode(&out, "PELICAN-PROTOCOL-FORMAT", nil)
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

// RemoveAsciiArmor is the inverse of WrapInAsciiArmor. It
// removes the armor from data.
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
