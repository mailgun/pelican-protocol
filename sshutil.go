package pelican

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"

	"code.google.com/p/go.crypto/ssh"
)

type KnownHosts struct {
	FilePath  string
	Hosts     map[string]*ServerPubKey
	curHost   *ServerPubKey
	curStatus HostState
}

type ServerPubKey struct {
	Hostname     string
	Remote       net.Addr
	key          ssh.PublicKey
	HumanKey     string // serialized and readable version of Key, the key for Hosts map in KnownHosts.
	ServerBanned bool
	//OurAcctKeyPair ssh.Signer
}

func NewKnownHosts(filepath string) *KnownHosts {

	h := &KnownHosts{}

	fn := filepath

	if FileExists(fn) {
		fmt.Printf("fn '%s' exists in NewKnownHosts()\n", fn)

		h.readJsonSnappy(fn)
		//h.readGobSnappy(fn)

		fmt.Printf("after reading from file, h = '%#v'\n", h)

	} else {
		fmt.Printf("making h.Hosts in NewKnownHosts()\n")
		h.Hosts = make(map[string]*ServerPubKey)
	}
	h.FilePath = filepath

	return h
}

func (h *KnownHosts) Sync() {
	//h.saveGobSnappy(fn)
	h.saveJsonSnappy(h.FilePath)
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

	record, ok := h.Hosts[strPubBytes]
	if ok {
		if record.ServerBanned {
			return Banned, fmt.Errorf("the key '%s' has been marked as banned.", strPubBytes), record
		}

		if record.Hostname != hostname {
			return KnownRecordMismatch, fmt.Errorf("hostname mismatch for key '%s': '%s' in records, '%s' supplied now.", strPubBytes, record.Hostname, hostname), record
		}
		if !reflect.DeepEqual(record.Remote, remote) {
			return KnownRecordMismatch, fmt.Errorf("remote address mismatch for key '%s': '%s' in records, '%s' supplied now.", strPubBytes, record.Remote, remote), record
		}
		return KnownOK, nil, record
	}

	if addIfNotKnown {
		record = &ServerPubKey{
			Hostname: hostname,
			Remote:   remote,
			//Key:      key,
			HumanKey: strPubBytes,
		}

		h.Hosts[strPubBytes] = record
		h.Sync()
	}

	return Unknown, nil, record
}

func loadRSAPrivateKey(path string) (privkey ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	privkey, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		panic(err)
	}
	return privkey, err
}

func (h *KnownHosts) sshConnect(username string, keypath string, host string, port int, command string) ([]byte, error) {

	// the kallback just after key-exchange to validate server is here
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {

		pubBytes := ssh.MarshalAuthorizedKey(key)

		hostStatus, err, spubkey := h.HostAlreadyKnown(hostname, remote, key, pubBytes, AddIfNotKnown)
		fmt.Printf("in hostKeyCallback(), hostStatus: '%s', hostname='%s', remote='%s', key.Type='%s'  key.Marshal='%s'\n", hostStatus, hostname, remote, key.Type(), pubBytes)

		h.curStatus = hostStatus
		h.curHost = spubkey

		if hostStatus == Banned {
			return fmt.Errorf("banned server")
		}

		if err != nil {
			return err
		}

		return nil
	}
	// end hostKeyCallback closure definition. Has to be a closure to access h.

	privkey, err := loadRSAPrivateKey(keypath)
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
	cli, err := ssh.Dial("tcp", hostport, cfg)
	if err != nil {
		panic(fmt.Sprintf("sshConnect() failed at dial to '%s': '%s' ", hostport, err.Error()))
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	sess, err := cli.NewSession()
	if err != nil {
		panic(fmt.Sprintf("Failed to create session to '%s': err = '%s'", hostport, err.Error()))
	}
	// you can only do one command on a session, so might as well close.
	defer sess.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	sess.Stdout = &b
	if err := sess.Run(command); err != nil {
		panic(fmt.Sprintf("sshConnect() failed to run login to '%s', err: '%s', out: '%s'", hostport, err.Error(), b.String()))
	}
	//fmt.Println(b.String())

	return b.Bytes(), nil
}
