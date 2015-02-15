package pelican

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"sync"
	"time"
)

// This key-pair is the canonical key-pair for establishing
// a new account, thus it is okay to distribute this
// private key. Everybody will use it to create a
// new login.

func GetNewAcctPublicKey() string {
	return newAcctPublicKey
}

func GetNewAcctPrivateKey() string {
	return newAcctPrivateKey
}

var newAcctPrivateKey string = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`

var newAcctPublicKey string = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDTRNQMhXy8fsP7zM5i+hsrCvtkmJ7X+EZLmemxI0WlVlpJ4m2sdtBiM0jVxA/MZIk+KeLEp6hQvKjbfHxtWm+TObN6iu/DMYCHv394zJHbaFVziHjnV7Q31h/4kUAzYq0RXeumvs7LBwvgoX+gEhjjK8T4rv5z3hpwOZuaIKJFwnBX0nGNinad7uZ0pF1B1KU0OSBAZvYiflWIsA7Hzfq1oDFsV4IHqKlLotXchnHhLhmP85crCo9OnGyZr4tUDCM2+smWD3Y6vpXYpuznCeFoA8Bew+iu0n1XS1qzsUX9VWMh8DhqtcfS2SLV1B1VfGmMuSlZ70DmVXg+6fxRemP9fqIwImbfWh0CXmk5plifoyR4IVC9FN/sVdrfGBdr+eeZXFAwa3nPPsTnbJYsKe1qMzZyaF+/usMsW8BViocm0zCAhdZQeuJdhPDe/ZUidi8DM/fvTkdmi84Faf2oRLccbwHdI1vFEmLWblKf4jknA2hbQ4IuQm7L6RTL39beAPGV0w2LIVdMwCL5ZdsyY+T4k1mbzSPdNx3tgxkZJsSVQnQ7sEtQeuo+GOcEPapQlSSe6IytRby85LwAzIWv/9xQXrntkCn5wKoKmUlMACJtFgGFthmHipaFknqiCnhm+5cxDF8YLdjxQ1zp7NGfGqdwJRwqrYZrIcQV70RumRxFYQ==`

func (h *KnownHosts) SshMakeNewAcct(privKeyPath string, host string, port int) error {

	fmt.Printf("\n ========== Begin SshMakeNewAcct().\n")

	// the callback just after key-exchange to validate server is here
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		pubBytes := ssh.MarshalAuthorizedKey(key)

		hostStatus, err, spubkey := h.HostAlreadyKnown(hostname, remote, key, pubBytes, AddIfNotKnown)
		fmt.Printf("in hostKeyCallback(), hostStatus: '%s', hostname='%s', remote='%s', key.Type='%s'  key.Marshal='%s'\n", hostStatus, hostname, remote, key.Type(), pubBytes)

		h.curStatus = hostStatus
		h.curHost = spubkey

		if hostStatus == Banned {
			fmt.Printf("detected Banned host.\n")
			return fmt.Errorf("banned server")
		}

		// super lenient for the moment.
		fmt.Printf("debug/early dev: returning nil from hostKeyCallback()\n")
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

	privkeyString := GetNewAcctPrivateKey()
	privkey, err := ssh.ParsePrivateKey([]byte(privkeyString))
	panicOn(err)

	externalIp := GetExternalIP()
	cfg := &ssh.ClientConfig{
		User: "newacct",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privkey),
		},
		// HostKeyCallback, if not nil, is called during the cryptographic
		// handshake to validate the server's host key. A nil HostKeyCallback
		// implies that all host keys are accepted => not good for MITM protection!
		HostKeyCallback: hostKeyCallback,
	}
	hostport := fmt.Sprintf("%s:%d", host, port)

	fmt.Printf("Just before Dial.\n")

	// instead of ssh.Dial, we net.Dial + NewClientConn so we can shutdown reverse forwards.

	addr := hostport
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("SshMakeNewAcct() failed at net.Dial to '%s': '%s' ", addr, err.Error()))
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		panic(fmt.Sprintf("SshMakeNewAcct() failed at ssh.NewClientConn to '%s': '%s' ", addr, err.Error()))
	}

	sshClientConn := NewMyClient(c, chans, reqs)

	fmt.Printf("Dial completed without error. sshClientConn is '%#v'\n", sshClientConn)

	msg := channelOpenDirectMsg{
		raddr: host,
		rport: uint32(80), // or 443? todo: enforce 80 or 443 this on the server end too by only accepting those numbers.
		laddr: externalIp,
		lport: uint32(0),
	}
	channel, reqs, err := c.OpenChannel("direct-tcpip", ssh.Marshal(&msg))
	panicOn(err)
	go ssh.DiscardRequests(reqs)

	// read replies
	go func() {
		by := make([]byte, 100)
		for {
			n, err := channel.Read(by)
			if err != nil {
				if err.Error() == "EOF" {
					fmt.Printf("client sees EOF on channel, closing out.")
					channel.Close()
					return
				}
				fmt.Printf("error in client Read-ing of direct tcp: '%s'", err)
			} else {
				fmt.Printf("\n client reads on direct-tcpip channel, %d bytes: '%s'\n", n, string(by))
			}
		}
	}()

	fmt.Printf("\nOpenChannel() completed without error. channel: '%#v'\n", channel)

	for i := 0; i < 10; i++ {
		channel.Write([]byte(fmt.Sprintf("hello from client-side newacct.go msg %d", i)))
		time.Sleep(2 * time.Second)
	}

	select {}

	return nil
}

func NewMyClient(c ssh.Conn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) *myClient {
	conn := &myClient{
		Conn:            c,
		channelHandlers: make(map[string]chan ssh.NewChannel, 1),
	}
	go conn.handleGlobalRequests(reqs)
	go conn.handleChannelOpens(chans)
	go func() {
		conn.Wait()
		conn.forwards.closeAll()
	}()
	go conn.forwards.handleChannels(conn.HandleChannelOpen("direct-tcpip"))
	return conn
}

// Client implements a traditional SSH client that supports shells,
// subprocesses, port forwarding and tunneled dialing.
//
// myClient restricts and eliminates alot of those features from ssh.Client.
type myClient struct {
	ssh.Conn

	forwards        forwardList // forwarded tcpip connections from the remote side
	mu              sync.Mutex
	channelHandlers map[string]chan ssh.NewChannel
}

func (c *myClient) handleGlobalRequests(incoming <-chan *ssh.Request) {
	for r := range incoming {
		// This handles keepalive messages and matches
		// the behaviour of OpenSSH.
		r.Reply(false, nil)
	}
}

// handleChannelOpens channel open messages from the remote side.
func (c *myClient) handleChannelOpens(in <-chan ssh.NewChannel) {
	for ch := range in {
		c.mu.Lock()
		handler := c.channelHandlers[ch.ChannelType()]
		c.mu.Unlock()
		if handler != nil {
			handler <- ch
		} else {
			ch.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %v", ch.ChannelType()))
		}
	}
	c.mu.Lock()
	for _, ch := range c.channelHandlers {
		close(ch)
	}
	c.channelHandlers = nil
	c.mu.Unlock()
}

//// from github.com/golang/crypto/ssh/tcpip.go
// sadly, forwardList is not publicly available, so must be duplicated here.

// forwardList stores a mapping between remote
// forward requests and the tcpListeners.
type forwardList struct {
	sync.Mutex
	entries []forwardEntry
}

// forwardEntry represents an established mapping of a laddr on a
// remote ssh server to a channel connected to a tcpListener.
type forwardEntry struct {
	laddr net.TCPAddr
	c     chan forward
}

// forward represents an incoming forwarded tcpip connection. The
// arguments to add/remove/lookup should be address as specified in
// the original forward-request.
type forward struct {
	newCh ssh.NewChannel // the ssh client channel underlying this forward
	raddr *net.TCPAddr   // the raddr of the incoming connection
}

func (l *forwardList) add(addr net.TCPAddr) chan forward {
	l.Lock()
	defer l.Unlock()
	f := forwardEntry{
		addr,
		make(chan forward, 1),
	}
	l.entries = append(l.entries, f)
	return f.c
}

// See RFC 4254, section 7.2
type forwardedTCPPayload struct {
	Addr       string
	Port       uint32
	OriginAddr string
	OriginPort uint32
}

// this version, for myClient, rejects all forward requests
// from the server for security purposes.
func (l *forwardList) handleChannels(in <-chan ssh.NewChannel) {
	for ch := range in {
		ch.Reject(ssh.Prohibited, "no forward for address")
	}
}

// remove removes the forward entry, and the channel feeding its
// listener.
func (l *forwardList) remove(addr net.TCPAddr) {
	l.Lock()
	defer l.Unlock()
	for i, f := range l.entries {
		if addr.IP.Equal(f.laddr.IP) && addr.Port == f.laddr.Port {
			l.entries = append(l.entries[:i], l.entries[i+1:]...)
			close(f.c)
			return
		}
	}
}

// closeAll closes and clears all forwards.
func (l *forwardList) closeAll() {
	l.Lock()
	defer l.Unlock()
	for _, f := range l.entries {
		close(f.c)
	}
	l.entries = nil
}

func (l *forwardList) forward(laddr, raddr net.TCPAddr, ch ssh.NewChannel) bool {
	l.Lock()
	defer l.Unlock()
	for _, f := range l.entries {
		if laddr.IP.Equal(f.laddr.IP) && laddr.Port == f.laddr.Port {
			f.c <- forward{ch, &raddr}
			return true
		}
	}
	return false
}

// HandleChannelOpen returns a channel on which NewChannel requests
// for the given type are sent. If the type already is being handled,
// nil is returned. The channel is closed when the connection is closed.
func (c *myClient) HandleChannelOpen(channelType string) <-chan ssh.NewChannel {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channelHandlers == nil {
		// The SSH channel has been closed.
		c := make(chan ssh.NewChannel)
		close(c)
		return c
	}

	ch := c.channelHandlers[channelType]
	if ch != nil {
		return nil
	}

	ch = make(chan ssh.NewChannel, 16)
	c.channelHandlers[channelType] = ch
	return ch
}

// direct from ~/go/src/github.com/golang/crypto/ssh/tcpip.go

// RFC 4254 7.2
type channelOpenDirectMsg struct {
	raddr string
	rport uint32
	laddr string
	lport uint32
}

func (c *myClient) dial(laddr string, lport int, raddr string, rport int) (ssh.Channel, error) {
	msg := channelOpenDirectMsg{
		raddr: raddr,
		rport: uint32(rport),
		laddr: laddr,
		lport: uint32(lport),
	}
	ch, in, err := c.OpenChannel("direct-tcpip", ssh.Marshal(&msg))
	go ssh.DiscardRequests(in)
	return ch, err
}
