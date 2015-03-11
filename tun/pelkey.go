package pelicantun

import (
	"crypto/rand"
	"fmt"
)

const KeyLen = 149

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

	//fmt.Printf("in GenPelicanKey, hmac = '%x'\n", hmac)

	signed_key := make([]byte, len(key)+len(hmac))
	//fmt.Printf("\n\n GenPelicanKey, signed_key is len %d\n", len(signed_key))
	copy(signed_key, key)
	copy(signed_key[2*randByteCount:], hmac)

	//fmt.Printf("before alpha, signed_key = '%x'\n", signed_key)

	alpha_signed_key := EncodeBytesBase36(signed_key)

	//fmt.Printf("\n\n GenPelicanKey, alpha_signed_key is len %d\n", len(alpha_signed_key))
	return alpha_signed_key
}

func IsLegitPelicanKey(alpha_signed_key []byte) bool {
	_, signed_key, err := Base36toBigInt(alpha_signed_key)
	if err != nil {
		// TODO: no longer constant time with early return. Side channel vuln.
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
