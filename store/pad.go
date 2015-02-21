package main

import (
	mathrand "math/rand"
	"time"
)

// Padding produces non crypto (fast) random bytes for
// appending to compressed messages to avoid leaking info.
func Padding(nbytes int) []byte {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	b := make([]byte, nbytes)
	for i := 0; i < nbytes; i++ {
		b[i] = byte(r.Uint32() % 256)
	}
	return b
}
