package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
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

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	ret := make([]byte, base64.URLEncoding.EncodedLen(len(ciphertext)))
	base64.URLEncoding.Encode(ret, ciphertext)
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
	ciphertext := dbuf[:n]

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext
}
