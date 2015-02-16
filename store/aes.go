package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func example_main() {
	originalText := "8 encrypt this golang 123"
	fmt.Println(originalText)

	pass := []byte("hello")

	// encrypt value to base64
	cryptoText := encrypt(pass, []byte(originalText))
	fmt.Println(string(cryptoText))

	// encrypt base64 crypto to original value
	text := decrypt(pass, cryptoText)
	fmt.Printf(string(text))
}

// want 32 byte key to select AES-256
var keyPadding = []byte(`z5L2XDZyCPvskrnktE-dUak2BQHW9tue`)

func XorWithKeyPadding(pw []byte) []byte {
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
	return dst
}

// encrypt string to base64 crypto using AES
func encrypt(passphrase []byte, plaintext []byte) []byte {

	key := XorWithKeyPadding(passphrase)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("nz = %d\n", gcm.NonceSize()) // 12

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	full := append(nonce, ciphertext...)

	// convert to base64
	ret := make([]byte, base64.URLEncoding.EncodedLen(len(full)))
	base64.URLEncoding.Encode(ret, full)
	return ret
}

// decrypt from base64 to decrypted string
func decrypt(passphrase []byte, cryptoText []byte) []byte {

	key := XorWithKeyPadding(passphrase)

	dbuf := make([]byte, base64.URLEncoding.DecodedLen(len(cryptoText)))
	n, err := base64.URLEncoding.Decode(dbuf, []byte(cryptoText))
	if err != nil {
		panic(err)
	}
	full := dbuf[:n]

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	nz := gcm.NonceSize()
	if len(full) < nz {
		panic("ciphertext too short")
	}

	nonce := full[:nz]
	ciphertext := full[nz:]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err)
	}

	return plain
}
