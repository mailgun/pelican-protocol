package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"math/big"
)

func Sha1HMAC(message, key []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// CheckMAC returns true if messageMAC is a valid HMAC tag for message.
func CheckSha1HMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}

func encodeSha1HmacAsUsername(sha1 []byte) string {
	i := new(big.Int)
	i.SetBytes(sha1)
	return "p" + bigIntToBase36string(i)
}

var enc36 string = "0123456789abcdefghijklmnopqrstuvwxyz"
var e36 []rune = []rune(enc36)

// i must be between 0 and 35 inclusive.
func encode36(i int64) rune {
	return e36[i]
}

func bigIntToBase36string(val *big.Int) string {
	const N = 31 // ceiling(log(2^160,36))
	res := make([]rune, N)
	left := new(big.Int)
	quo := new(big.Int)
	rem := new(big.Int)
	*left = *val

	div := big.NewInt(36)

	for i := 0; i < N; i++ {
		quo.QuoRem(left, div, rem)
		*left = *quo
		r := rem.Int64()
		e := encode36(r)
		res[N-1-i] = e
	}

	return string(res)
}

func Sha256HMAC(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// CheckMAC returns true if messageMAC is a valid HMAC tag for message.
func CheckSha256HMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
