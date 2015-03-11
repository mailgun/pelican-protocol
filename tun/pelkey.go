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
	signed_key := make([]byte, len(key)+len(hmac))
	fmt.Printf("\n\n GenPelicanKey, signed_key is len %d\n", len(signed_key))
	copy(signed_key, key)
	copy(signed_key[2*randByteCount:], hmac)

	alpha_signed_key := EncodeBytesBase36(signed_key)

	fmt.Printf("\n\n GenPelicanKey, alpha_signed_key is len %d\n", len(alpha_signed_key))
	return alpha_signed_key
}

func IsLegitPelicanKey(alpha_signed_key []byte) bool {
	_, key := Base36toBigInt(alpha_signed_key)
	if len(key) != 97 {
		panic(fmt.Errorf("bad len(key): %d", len(key)))
	}
	key = key[:96]
	sig := key[2*randByteCount:]

	return CheckSha256HMAC(key[:randByteCount], sig, key[randByteCount:2*randByteCount])
}
