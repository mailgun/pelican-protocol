package pelican

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"

	//"code.google.com/p/go.crypto/ssh"
	"golang.org/x/crypto/ssh"
)

func defaultFileFormat() string {
	// either ".gob.snappy" or ".json.snappy"
	return ".json.snappy"
	//return ".gob.snappy"
}

const AddIfNotKnown = true
const IgnoreIfNotKnown = false

type HostState int

const Unknown HostState = 0
const Banned HostState = 1
const KnownOK HostState = 2
const KnownRecordMismatch HostState = 3

func (s HostState) String() string {
	switch s {
	case Unknown:
		return "Unknown"
	case Banned:
		return "Banned"
	case KnownOK:
		return "KnownOK"
	case KnownRecordMismatch:
		return "KnownRecordMismatch"
	}
	return ""
}

func (h *KnownHosts) HostAlreadyKnown(hostname string, remote net.Addr, key ssh.PublicKey, pubBytes []byte, addIfNotKnown bool) (HostState, error, *ServerPubKey) {
	strPubBytes := string(pubBytes)

	fmt.Printf("\n in HostAlreadyKnonw...starting.\n")

	record, ok := h.Hosts[strPubBytes]
	if ok {
		if record.ServerBanned {
			return Banned, fmt.Errorf("the key '%s' has been marked as banned.", strPubBytes), record
		}

		if strings.HasPrefix(hostname, "localhost") || strings.HasPrefix(hostname, "127.0.0.1") {
			// no host checking when coming from localhost
			return KnownOK, nil, record
		}
		if record.Hostname != hostname {
			err := fmt.Errorf("hostname mismatch for key '%s': '%s' in records, '%s' supplied now.", strPubBytes, record.Hostname, hostname)
			fmt.Printf("\n in HostAlreadyKnown, returning KnownRecordMismatch: '%s'", err)
			return KnownRecordMismatch, err, record
		}
		if !reflect.DeepEqual(record.remote, remote) {
			err := fmt.Errorf("remote address mismatch for key '%s': '%s' in records, '%s' supplied now.", strPubBytes, record.remote, remote)
			fmt.Printf("\n in HostAlreadyKnown, returning KnownRecordMismatch: '%s'", err)
			return KnownRecordMismatch, err, record
		}
		return KnownOK, nil, record
	}

	if addIfNotKnown {
		record = &ServerPubKey{
			Hostname: hostname,
			remote:   remote,
			//Key:      key,
			HumanKey: strPubBytes,
		}

		h.Hosts[strPubBytes] = record
		h.Sync()
	}

	return Unknown, nil, record
}

func (h *KnownHosts) SshConnect(username string, keypath string, host string, port int, localPortToListenOn int) ([]byte, error) {

	// the callback just after key-exchange to validate server is here
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {

		pubBytes := ssh.MarshalAuthorizedKey(key)

		hostStatus, err, spubkey := h.HostAlreadyKnown(hostname, remote, key, pubBytes, AddIfNotKnown)
		fmt.Printf("in hostKeyCallback(), hostStatus: '%s', hostname='%s', remote='%s', key.Type='%s'  key.Marshal='%s'\n", hostStatus, hostname, remote, key.Type(), pubBytes)

		h.curStatus = hostStatus
		h.curHost = spubkey

		if hostStatus == Banned {
			return fmt.Errorf("banned server")
		}

		// super lenient for the moment.
		err = nil
		return err
		/*

			if err != nil {
				// this is strict checking of hosts here, any non-nil error
				// will fail the ssh handshake.
				return err
			}

			return nil
		*/
	}
	// end hostKeyCallback closure definition. Has to be a closure to access h.

	privkey, err := LoadRSAPrivateKey(keypath)
	if err != nil {
		panic(err)
	}

	cfg := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privkey),
		},
		// HostKeyCallback, if not nil, is called during the cryptographic
		// handshake to validate the server's host key. A nil HostKeyCallback
		// implies that all host keys are accepted.
		HostKeyCallback: hostKeyCallback,
	}
	hostport := fmt.Sprintf("%s:%d", host, port)
	sshClientConn, err := ssh.Dial("tcp", hostport, cfg)
	if err != nil {
		panic(fmt.Sprintf("sshConnect() failed at dial to '%s': '%s' ", hostport, err.Error()))
	}

	channelToSshd, reqs, err := sshClientConn.Conn.OpenChannel("forwarded-tcpip", nil)
	panicOn(err)

	fmt.Printf("OpenChannel() completed without error. channelToSshd: '%#v'\n", channelToSshd)

	go ssh.DiscardRequests(reqs)

	// make a local server listening on localPortToListenOn
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPortToListenOn))
	if err != nil {
		panic(err) // todo handle error
	}
	for {
		fromBrowser, err := ln.Accept()
		if err != nil {
			panic(err) // todo handle error
		}

		// here is the heart of the ssh-secured tunnel functionality:
		// we start the two shovels that keep traffic flowing
		// in both directions from browser over to sshd:
		// reads on fromBrowser are forwarded to channelToSshd;
		// reads on channelToSshd are forwarded to fromBrowser.

		// Copy channelToSshd.Reader to fromBrowser.Writer
		go func() {
			_, err := io.Copy(fromBrowser, channelToSshd)
			if err != nil {
				fmt.Printf("io.Copy failed: %v\n", err)
				fromBrowser.Close()
				channelToSshd.Close()
				return
			}
		}()

		// Copy fromBrowser.Reader to channelToSshd.Writer
		go func() {
			_, err := io.Copy(channelToSshd, fromBrowser)
			if err != nil {
				fmt.Printf("io.Copy failed: %v\n", err)
				fromBrowser.Close()
				channelToSshd.Close()
				return
			}
		}()
	}

	return []byte{}, nil
}

// Fingerprint performs a SHA256 BASE64 fingerprint of the PublicKey, similar to OpenSSH.
// See: https://anongit.mindrot.org/openssh.git/commit/?id=56d1c83cdd1ac
func Fingerprint(k ssh.PublicKey) string {
	hash := sha256.Sum256(k.Marshal())
	return base64.StdEncoding.EncodeToString(hash[:])
}
