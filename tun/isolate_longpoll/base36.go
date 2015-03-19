package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

func Sha1HMAC(message, key []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

func EncodeBytesBase36(by []byte) []byte {
	i := new(big.Int)
	i.SetBytes(by)
	b, _ := BigIntToBase36(i)
	return b
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
	_, str := BigIntToBase36(i)
	return "p" + str
}

var enc36 string = "0123456789abcdefghijklmnopqrstuvwxyz"
var e36 []rune = []rune(enc36)

// i must be between 0 and 35 inclusive.
func encode36(i int64) rune {
	return e36[i]
}

func decode36(r rune) int64 {
	switch {
	case r >= rune('0') && r <= rune('9'):
		return int64(r - rune('0'))
	case r >= rune('a') && r <= rune('z'):
		return int64(r-rune('a')) + 10
	default:
		panic(fmt.Sprintf("'%v' is outside our domain", r))
	}
	return 0
}

// check to see if all runes are in the enc36 range
func IsAlphaBase36(ru []rune) bool {

	okay := true
	for _, r := range ru {
		switch {
		case r >= rune('0') && r <= rune('9'):
			continue
		case r >= rune('a') && r <= rune('z'):
			continue
		default:
			okay = false
		}
	}
	return okay
}

// returns both a []byte and a string encoding of the bigInt val
// using only a-z0-9 characters. Assumes a positive bigInt val.
// Negative val results are undefined.
func BigIntToBase36(val *big.Int) ([]byte, string) {

	by := val.Bytes()
	nBytes := len(by)
	nBits := nBytes * 8

	// zero special case
	if nBytes == 0 {
		return []byte{0}, "0"
	}

	// compute how many bits we'll need to
	// encode nBytes in base 36. We want
	// Ceil(Log(2^nBits, 36)) == Ceil(nBits * Log(2, 36))
	//  and
	// log(2, 36) == 0.1934264
	const log2base36 float64 = 0.1934264
	NeededDigits36 := int(math.Ceil(log2base36 * float64(nBits)))

	N := NeededDigits36
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

	s := string(res)
	return []byte(s), s
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

// inverse of BigIntToBase36. the []byte return
// argument is padded on the left with zeros to
// reach the largest possible big.Int number
// for len(a) bytes.
func Base36toBigInt(a []byte) (*big.Int, []byte, error) {
	ru := []rune(string(a))

	if !IsAlphaBase36(ru) {
		return nil, []byte{}, fmt.Errorf("not base36 input byte string when translated -> string -> []rune")
	}

	var r rune
	var next *big.Int
	tot := new(big.Int)
	mult := big.NewInt(1)
	b36 := big.NewInt(36)
	N := len(ru)
	for i := N - 1; i >= 0; i-- {
		r = ru[i]
		next = big.NewInt(decode36(r))

		pre := new(big.Int)
		pre.Mul(next, mult)
		//fmt.Printf("\n   a = '%s', ru='%v', at i = %d, r = '%v', pre = %v, next = %v\n\n", string(a), ru, i, r, pre, next)
		tot.Add(tot, pre)

		mult.Mul(mult, b36)
	}

	// pad res on the right with zeros if need be
	max := new(big.Int)
	max.Mul(mult, b36)

	M := len(max.Bytes())
	res := make([]byte, M)
	skip := 0
	totbytes := tot.Bytes()
	lentotbytes := len(totbytes)

	if lentotbytes < M {
		skip = M - lentotbytes
	}
	// pad res on left with zeros
	copy(res[skip:], totbytes)

	return tot, res, nil
}
