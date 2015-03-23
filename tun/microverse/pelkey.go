package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
)

const KeyLen = 149
const SerialLen = 8
const HeaderLen = KeyLen + SerialLen

const randByteCount = 32

func RandBytes(n int) []byte {

	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Errorf("error on rand.Read: '%s'", err))
	}

	return b
}

func GenPelicanKey() []byte {

	key := RandBytes(2 * randByteCount)

	hmac := Sha256HMAC(key[:randByteCount], key[randByteCount:2*randByteCount])

	if len(hmac) != 32 {
		panic(fmt.Sprintf("expect Sha256HMAC() to return 32 bytes, but got %d instead.", len(hmac)))
	}

	//fmt.Printf("in GenPelicanKey, hmac = '%x' with len: %d\n", hmac, len(hmac))

	signed_key := make([]byte, len(key)+len(hmac))
	//fmt.Printf("\n\n GenPelicanKey, signed_key is len %d\n", len(signed_key))
	copy(signed_key, key)
	copy(signed_key[2*randByteCount:], hmac)

	//fmt.Printf("before alpha, signed_key = '%x'\n", signed_key)

	alpha_signed_key := EncodeBytesBase36(signed_key)

	sz := len(alpha_signed_key)
	if sz > KeyLen {
		panic(fmt.Sprintf("key too long: %d but expected", sz, KeyLen))
	}
	if sz < KeyLen {
		// key is too short because the bignum was too small; prepend with zeros
		// to get an accurate representation out to KeyLen.
		alpha_signed_key = append([]byte(strings.Repeat("0", KeyLen-sz)), alpha_signed_key...)
	}

	if !IsLegitPelicanKey(alpha_signed_key) {
		panic("alpha_signed_key not passing the IsLegitPelicanKey() test")
	}

	//fmt.Printf("\n\n GenPelicanKey, alpha_signed_key is len %d\n", len(alpha_signed_key))
	return alpha_signed_key
}

func IsLegitPelicanKey(alpha_signed_key []byte) bool {
	if len(alpha_signed_key) != KeyLen {
		return false
	}
	_, signed_key, err := Base36toBigInt(alpha_signed_key)
	if err != nil {
		// TODO: no longer constant time with early return. Side channel attack vulnerable.
		// not that this is a secure id anyway, so for now ignore.
		return false
	}

	if len(signed_key) != 97 {
		// TODO: no longer constant time with early return. Side channel vuln.
		return false
	}
	signed_key = signed_key[1:]

	//fmt.Printf("after un-alpha, signed_key = '%x'\n", signed_key)

	hmac := signed_key[2*randByteCount:]

	//fmt.Printf("after un-alpha, expected hmac = '%x'\n", hmac)

	return CheckSha256HMAC(signed_key[:randByteCount], hmac, signed_key[randByteCount:2*randByteCount])
}

// the sequence number is the request sequence going from client -> server,
func ParseRequestHeader(header []byte) (key []byte, ser int64) {
	key = header[:KeyLen]
	serBy := header[KeyLen : KeyLen+SerialLen]
	ser = int64(binary.LittleEndian.Uint64(serBy))
	return
}

func SerialToBytes(serialNum int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(serialNum))
	return b
}

func BytesToSerial(p []byte) int64 {
	if len(p) != SerialLen {
		panic(fmt.Sprintf("p must be of length SerialLen == %d, but was %d", SerialLen, len(p)))
	}
	return int64(binary.LittleEndian.Uint64(p))
}
