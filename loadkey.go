package pelican

import "crypto/md5"

/*
NB This is the SSLeay compatible version of key-generation.
See also for later PKCS8 discussion this link

http://martin.kleppmann.com/2013/05/24/improving-security-of-ssh-private-keys.html

from which the follow excerpt is taken.

Passphrase-protected keys

Next, in order to make life harder for an attacker who manages
to steal your private key file, you protect it with a
passphrase. How does this actually work?

$ ssh-keygen -t rsa -N 'super secret passphrase' -f test_rsa_key
$ cat test_rsa_key
-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-128-CBC,D54228DB5838E32589695E83A22595C7

3+Mz0A4wqbMuyzrvBIHx1HNc2ZUZU2cPPRagDc3M+rv+XnGJ6PpThbOeMawz4Cbu
lQX/Ahbx+UadJZOFrTx8aEWyZoI0ltBh9O5+ODov+vc25Hia3jtayE51McVWwSXg
wYeg2L6U7iZBk78yg+sIKFVijxiWnpA7W2dj2B9QV0X3ILQPxbU/cRAVTd7AVrKT
    ... etc ...
-----END RSA PRIVATE KEY-----

We’ve gained two header lines, and if you try to parse
that Base64 text, you’ll find it’s no longer valid ASN.1.
That’s because the entire ASN.1 structure we saw above
has been encrypted, and the Base64-encoded text is the
output of the encryption. The header tells us the
encryption algorithm that was used: AES-128 in CBC mode.
The 128-bit hex string in the DEK-Info header is the
initialization vector (IV) for the cipher. This is
pretty standard stuff; all common crypto libraries can
handle it.


But how do you get from the passphrase to the AES encryption
key? I couldn’t find it documented anywhere, so I had to dig
through the OpenSSL source to find it:

Append the first 8 bytes of the IV to the passphrase, without a separator (serves as a salt).

Take the MD5 hash of the resulting string (once).

*/

// PasswordToSshPrivKeyUnlocker follows the OpenSSL formula
// for converting a human entered password into the key
// used to decode the AES-128-CBC (or otherwise) encrypted
// private key.
func PasswordToSshPrivKeyUnlocker(password []byte, iv []byte) []byte {
	lenpass := len(password)
	withsalt := make([]byte, lenpass+8)
	copy(withsalt, password)
	copy(withsalt[lenpass:], iv[:8])
	md := md5.Sum(withsalt)
	return md[:]
}
