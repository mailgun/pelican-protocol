package pelican

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	mathrand "math/rand"
	"time"

	"golang.org/x/crypto/pbkdf2"
	sha3 "golang.org/x/crypto/sha3"
)

const gcmNonceByteLen = 12

const RequiredSaltLen = 64

func example_main() {
	originalText := "8 encrypt this golang 123"
	fmt.Println(originalText)

	pass := []byte("hello")

	salt := MakeRandPadding(RequiredSaltLen, RequiredSaltLen)
	// that is weak. /dev/urandom is better:
	if _, err := cryptorand.Read(salt); err != nil {
		panic(err)
	}
	//fmt.Printf("salt = %x\n", salt)

	// encrypt value to base64
	cryptoText := EncryptAes256Gcm(pass, []byte(originalText), salt)
	fmt.Println(string(cryptoText))

	// encrypt base64 crypto to original value
	text := DecryptAes256Gcm(pass, cryptoText)
	fmt.Printf(string(text))
}

// want 32 byte key to select AES-256.
// Don't change this (unless you want to invalidate all
// your existing passwords).
var keyPadding = []byte(`z5L2XDZyCPvskrnktE-dUak2BQHW9tue`)

// XorWrapBytes deterministicallyl XORs two
// byte slices together, wrapping one against
// the other if need be. The result is the
// same length as the longer of a and b
func XorWrapBytes(a []byte, b []byte) []byte {
	na := len(a)
	nb := len(b)

	if na == 0 || nb == 0 {
		panic("must have non zero length slices as inputs")
	}

	min := na
	if nb < min {
		min = nb
	}

	max := na
	if nb > max {
		max = nb
	}

	dst := make([]byte, max)
	ndst := max

	for i := 0; i < max; i++ {
		dst[i%ndst] = b[i%nb] ^ a[i%na]
	}

	return dst
}

func xorWithKeyPadding(pw []byte, salt []byte) []byte {
	if len(keyPadding) != 32 {
		panic("32 bit key needed to invoke AES256")
	}
	dst := make([]byte, len(keyPadding))
	ndst := len(dst)
	npw := len(pw)
	max := npw
	if max < ndst {
		max = ndst
	}
	for i := 0; i < max; i++ {
		dst[i%ndst] = keyPadding[i%ndst] ^ pw[i%npw]
	}

	key := append(dst, salt...)

	kb0 := []byte(key)

	// key stretching; do a bunch of sha1
	N := 10000
	for i := 0; i < N; i++ {
		kb1 := sha1.Sum(kb0)
		kb0 = kb1[:]
		//fmt.Printf("len of keybytes = %d, '%x'\n", len(kb0), string(kb0)) // 20
	}

	// finish with more key stretching with different hashes.
	sha3_512_digest := sha3.Sum512(kb0)

	shaker := sha3.NewShake256()
	_, err := shaker.Write(sha3_512_digest[:])
	if err != nil {
		panic(fmt.Sprintf("could not write into shaker from sha3_512_digest: '%s'", err))
	}
	//fmt.Printf("\n kb0 = '%x'\n", kb0)
	//fmt.Printf("\n sha3_512_digest = '%x'\n", sha3_512_digest)

	shakenNotStirred := make([]byte, 64)
	n, err := shaker.Read(shakenNotStirred)
	if n != 64 || err != nil {
		panic(fmt.Sprintf("could not read 64 bytes for shakenNotStirred: '%s'", err))
	}

	// xor down from 64 to 32 bytes
	res := XorWrapBytes(shakenNotStirred[:32], shakenNotStirred[32:64])

	// more official key stretching by RSA's PBKDF2
	res2 := pbkdf2.Key(res, salt, 4096, 32, sha512.New384)

	//fmt.Printf("res = %x\n", res)
	//fmt.Printf("res2 = %x\n", res2)
	return res2
}

// EncryptAes256Gcm encrypts plaintext using passphrase using AES256-GCM,
// then converts it to base64url encoding.
func EncryptAes256Gcm(passphrase []byte, plaintext []byte, salt []byte) []byte {

	if len(salt) != RequiredSaltLen {
		panic(fmt.Sprintf("salt was length %d, but %d required", len(salt), RequiredSaltLen))
	}

	//fmt.Printf("nz = %d\n", gcm.NonceSize()) // 12

	nonce := make([]byte, gcmNonceByteLen)
	if _, err := cryptorand.Read(nonce); err != nil {
		panic(err)
	}

	key := xorWithKeyPadding(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	// hide length by adding randomness at begin/end
	lenPlain := len(plaintext)
	pad1 := MakeRandPadding(16, 255)
	lenPad1 := len(pad1) // fits in 8 bits
	pad2 := MakeRandPadding(16, 255)
	lenPad2 := len(pad2) // fits in 8 bits

	np := lenPad1 + lenPlain + lenPad2 + 2
	paddedPlain := make([]byte, np)
	paddedPlain[0] = byte(lenPad1)
	paddedPlain[1] = byte(lenPad2)

	copy(paddedPlain[2:], pad1)
	copy(paddedPlain[2+lenPad1:], plaintext)
	copy(paddedPlain[2+lenPad1+lenPlain:], pad2)

	ciphertext := gcm.Seal(nil, nonce, paddedPlain, nil)
	full := append(nonce, ciphertext...)
	full = append(full, salt...)

	// convert to base64
	ret := make([]byte, base64.URLEncoding.EncodedLen(len(full)))
	base64.URLEncoding.Encode(ret, full)
	return ret
}

// DecryptAes256Gcm is the inverse of EncryptAesGcm. It removes the
// base64url encoding, and then decrypts cryptoText using passphrase
// under the assumption that AES256-GCM was used to encrypt it.
func DecryptAes256Gcm(passphrase []byte, cryptoText []byte) []byte {

	dbuf := make([]byte, base64.URLEncoding.DecodedLen(len(cryptoText)))
	n, err := base64.URLEncoding.Decode(dbuf, []byte(cryptoText))
	if err != nil {
		panic(err)
	}
	full := dbuf[:n]
	nz := gcmNonceByteLen
	req := nz + RequiredSaltLen + 1
	if len(full) < req {
		panic(fmt.Errorf("ciphertext too short. len(full)=%d, need at least %d", len(full), req))
	}

	cipherLen := n - RequiredSaltLen

	nonce := full[:nz]
	ciphertext := full[nz:cipherLen]
	salt := full[cipherLen:]

	lenSalt := len(salt)
	if lenSalt != RequiredSaltLen {
		panic(fmt.Sprintf("decoding: salt was length %d, but %d required", lenSalt, RequiredSaltLen))
	}

	key := xorWithKeyPadding(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err)
	}

	lenPad1 := int(plain[0])
	lenPad2 := int(plain[1])
	lenPlain := len(plain)

	return plain[2+lenPad1 : lenPlain-lenPad2]
}

// MakeRandPadding produces non crypto (fast) random bytes for
// prepending to messges/compressed messages to avoid leaking info,
// and to make it harder to recognize if you've actually
// cracked it.
func MakeRandPadding(minBytes int, maxBytes int) []byte {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	span := maxBytes - minBytes
	if span < 0 {
		panic("negative span")
	}
	nbytes := minBytes
	if span > 0 {
		nbytes += int(r.Int63n(int64(span)))
	}

	b := make([]byte, nbytes)
	for i := 0; i < nbytes; i++ {
		b[i] = byte(r.Uint32() % 256)
	}
	return b
}
