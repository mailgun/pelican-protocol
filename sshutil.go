package pelican

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"strings"

	"crypto/sha256"

	//"code.google.com/p/go.crypto/ssh"
	"golang.org/x/crypto/ssh"
)

type KnownHosts struct {
	Hosts     map[string]*ServerPubKey
	curHost   *ServerPubKey
	curStatus HostState

	// FilepathPrefix doesn't have the .json.snappy suffix on it.
	FilepathPrefix string

	// PersistFormat doubles as the file suffix as well as
	// the format indicator
	PersistFormat string
}

type ServerPubKey struct {
	Hostname     string
	HumanKey     string // serialized and readable version of Key, the key for Hosts map in KnownHosts.
	ServerBanned bool
	//OurAcctKeyPair ssh.Signer

	remote net.Addr      // unmarshalled form of Hostname
	key    ssh.PublicKey // unmarshalled form of HumanKey
}

func defaultFileFormat() string {
	// either ".gob.snappy" or ".json.snappy"
	return ".json.snappy"
	//return ".gob.snappy"
}

// filepathPrefix does not include the PersistFormat suffix
func NewKnownHosts(filepathPrefix string) *KnownHosts {

	h := &KnownHosts{
		PersistFormat: defaultFileFormat(),
	}

	fn := filepathPrefix + h.PersistFormat

	if FileExists(fn) {
		fmt.Printf("fn '%s' exists in NewKnownHosts()\n", fn)

		switch h.PersistFormat {
		case ".json.snappy":
			err := h.readJsonSnappy(fn)
			panicOn(err)
		case ".gob.snappy":
			err := h.readGobSnappy(fn)
			panicOn(err)
		default:
			panic(fmt.Sprintf("unknown persistence format", h.PersistFormat))
		}

		fmt.Printf("after reading from file, h = '%#v'\n", h)

	} else {
		fmt.Printf("fn '%s' does not exist already in NewKnownHosts()\n", fn)
		fmt.Printf("making h.Hosts in NewKnownHosts()\n")
		h.Hosts = make(map[string]*ServerPubKey)
	}
	h.FilepathPrefix = filepathPrefix

	return h
}

func KnownHostsEqual(a, b *KnownHosts) (bool, error) {
	for k, v := range a.Hosts {
		v2, ok := b.Hosts[k]
		if !ok {
			return false, fmt.Errorf("KnownHostsEqual detected difference at key '%s': a.Hosts had this key, but b.Hosts did not have this key", k)
		}
		if v.HumanKey != v2.HumanKey {
			return false, fmt.Errorf("KnownHostsEqual detected difference at key '%s': a.HumanKey = '%s' but b.HumanKey = '%s'", v.HumanKey, v2.HumanKey)
		}
		if v.Hostname != v2.Hostname {
			return false, fmt.Errorf("KnownHostsEqual detected difference at key '%s': a.Hostname = '%s' but b.Hostname = '%s'", v.Hostname, v2.Hostname)
		}
		if v.ServerBanned != v2.ServerBanned {
			return false, fmt.Errorf("KnownHostsEqual detected difference at key '%s': a.ServerBanned = '%s' but b.ServerBanned = '%s'", v.ServerBanned, v2.ServerBanned)
		}
	}
	for k := range b.Hosts {
		_, ok := a.Hosts[k]
		if !ok {
			return false, fmt.Errorf("KnownHostsEqual detected difference at key '%s': b.Hosts had this key, but a.Hosts did not have this key", k)
		}
	}
	return true, nil
}

func (h *KnownHosts) Sync() {
	fn := h.FilepathPrefix + h.PersistFormat
	switch h.PersistFormat {
	case ".json.snappy":
		err := h.saveJsonSnappy(fn)
		panicOn(err)
	case ".gob.snappy":
		err := h.saveGobSnappy(fn)
		panicOn(err)
	default:
		panic(fmt.Sprintf("unknown persistence format", h.PersistFormat))
	}
}

func (h *KnownHosts) Close() {
	h.Sync()
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

func LoadRSAPrivateKey(path string) (privkey ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(path)
	panicOn(err)

	privkey, err = ssh.ParsePrivateKey(buf)
	panicOn(err)

	return privkey, err
}

func LoadRSAPublicKey(path string) (pubkey ssh.PublicKey, err error) {
	buf, err := ioutil.ReadFile(path)
	panicOn(err)

	pub, _, _, _, err := ssh.ParseAuthorizedKey(buf)
	panicOn(err)

	return pub, err
}

func (h *KnownHosts) SshConnect(username string, keypath string, host string, port int, command string) ([]byte, error) {

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

	channel, reqs, err := sshClientConn.Conn.OpenChannel("forwarded-tcpip", nil)
	panicOn(err)

	fmt.Printf("OpenChannel() completed without error. channel: '%#v'\n", channel)

	go ssh.DiscardRequests(reqs)

	////////////////////////////////

	return []byte{}, nil
}

// Fingerprint performs a SHA256 BASE64 fingerprint of the PublicKey, similar to OpenSSH.
// See: https://anongit.mindrot.org/openssh.git/commit/?id=56d1c83cdd1ac
func Fingerprint(k ssh.PublicKey) string {
	hash := sha256.Sum256(k.Marshal())
	return base64.StdEncoding.EncodeToString(hash[:])
}
