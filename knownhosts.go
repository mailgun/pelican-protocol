package pelican

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
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
