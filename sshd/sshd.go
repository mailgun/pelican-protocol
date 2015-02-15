//  Inspiration from
//  https://gist.github.com/jpillora/b480fde82bff51a06238
//  Thanks to Jamie Pillora for the detailed gist.
//
// A small SSH daemon providing bash sessions
//
// Server:
// cd my/new/dir/
// #generate server keypair
// ssh-keygen -t rsa
// go get -v .
// go run sshd.go
//
// Client:
// ssh foo@localhost -p 2200 #pass=bar

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/kr/pty"
	pelican "github.com/mailgun/pelican-protocol"
	"golang.org/x/crypto/ssh"
)

type User struct {
	PubKey     string
	AcctId     string // 32-byte one-way crypto hash of PubKey. The 'user' who is ssh-ing in.
	Email      string
	SMS        string
	FirstName  string
	MiddleName string
	LastName   string
	FirstIp    string
	FirstTm    time.Time
	LastIp     string
	LastTm     time.Time
	Banned     bool
}

type Users struct {
	KnownClientPubKey map[string]User
}

func NewUsers() *Users {
	u := &Users{
		KnownClientPubKey: make(map[string]User),
	}
	return u
}

// should be constant-time to avoid side-channel timing attacks.
func (u *Users) PermitClientConnection(clientUser string, clientAddr net.Addr, clientPubKey ssh.PublicKey) (bool, error) {

	pubBytes := ssh.MarshalAuthorizedKey(clientPubKey)
	strPubBytes := string(chomp(pubBytes))

	fmt.Printf("\n in PermitClientConnection(): clientUser = '%s', clientAddr = '%#v', clientPubKey = '%s'\n", clientUser, clientAddr, strPubBytes)

	if strPubBytes == pelican.GetNewAcctPublicKey() {
		fmt.Printf("PermitClientConnection detected NewAcct public key, returning true\n")
		return true, fmt.Errorf("new-account")
	}
	fmt.Printf("equal? %v \n pelican.GetNewAcctPublicKey() = \n'%s'\n and strPubBytes = \n'%s'\n", pelican.GetNewAcctPublicKey() == strPubBytes, pelican.GetNewAcctPublicKey(), strPubBytes)

	if clientUser == "newacct" {
		fmt.Printf("PermitClientConnection detected user 'newacct', returning true\n")
		return true, fmt.Errorf("new-account")
	}
	// the username is issued by the server upon the completion of the
	// new account protocol, and is the hmac of the client's public key
	// signed with a secret only the server knows. The secret used
	// to sign the hmac should be preserved as long as the service is
	// alive, since if you loose it none of the account names can
	// be validated.

	/* not so strict at first!

	secretServerId, err := FetchSecretIdForService(".secret_id_for_service")
	panicOn(err)
	hmac := Sha1HMAC(pubBytes, []byte(secretServerId))
	acctid := encodeSha1HmacAsUsername(hmac)

	if clientUser != acctid {
		// somebody is trying to use a public key we've seen before, but
		// they did not receive our signature (acctid) for it, so likely
		// what happened is they didn't complete the new-account-creation
		// protocol. Hence we reject their connection.
		// The final step in completion of the new account protocol is that
		// we give the client their acctid (equivalent of a username);
		// which is an hmac of their public key, signed with a secret
		// known only to the server.
		return false, fmt.Errorf("bad-account-id")
	}
	*/

	user, ok := u.KnownClientPubKey[strPubBytes]
	if ok {
		if user.Banned {
			fmt.Printf("PermitClientConnection returning banned-user\n")
			return false, fmt.Errorf("banned-user")
		}
		now := time.Now()
		user.LastIp = clientAddr.String()
		user.LastTm = now
		if user.FirstIp == "" {
			user.FirstIp = user.LastIp
			user.FirstTm = now
		}
		fmt.Printf("PermitClientConnection true, user known.\n")
		return true, nil
	}

	fmt.Printf("PermitClientConnection false: user-unknown\n")
	return false, fmt.Errorf("user-unknown")
}

func main() {

	users := NewUsers()

	config := &ssh.ServerConfig{
		// must have keys
		NoClientAuth: false,

		// pki based login only
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			ok, err := users.PermitClientConnection(conn.User(), conn.RemoteAddr(), key)
			if !ok {
				return nil, err
			}
			perm := &ssh.Permissions{Extensions: map[string]string{
				"pubkey": string(key.Marshal()),
			}}
			return perm, nil
		},
		// no passwords
		PasswordCallback: nil,
	}

	err := GetOrGenServerKey("./host-key-id-rsa", config)
	panicOn(err)

	// Once a ServerConfig has been configured, connections can be accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2200")
	if err != nil {
		log.Fatalf("Failed to listen on 2200 (%s)", err)
	}

	// Accept all connections
	log.Print("Listening on 2200...")
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Printf("Failed to handshake (%s)", err)
			continue
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		// Discard all global out-of-band Requests
		go ssh.DiscardRequests(reqs)
		// Accept all channels
		go handleChannels(chans)
	}
}

func handleChannels(chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {
	// Since we're handling a shell, we expect a
	// channel type of "session". The also describes
	// "x11", "direct-tcpip" and "forwarded-tcpip"
	// channel types.
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	// At this point, we have the opportunity to reject the client's
	// request for another logical connection
	connection, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel (%s)", err)
		return
	}

	// Fire up bash for this session
	bash := exec.Command("bash")

	// Prepare teardown function
	close := func() {
		connection.Close()
		_, err := bash.Process.Wait()
		if err != nil {
			log.Printf("Failed to exit bash (%s)", err)
		}
		log.Printf("Session closed")
	}

	// Allocate a terminal for this channel
	log.Print("Creating pty...")
	bashf, err := pty.Start(bash)
	if err != nil {
		log.Printf("Could not start pty (%s)", err)
		close()
		return
	}

	//pipe session to bash and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, bashf)
		once.Do(close)
	}()
	go func() {
		io.Copy(bashf, connection)
		once.Do(close)
	}()

	// Sessions have out-of-band requests such as "shell", "pty-req" and "env"
	go func() {
		for req := range requests {
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) == 0 {
					req.Reply(true, nil)
				}
			case "pty-req":
				termLen := req.Payload[3]
				w, h := parseDims(req.Payload[termLen+4:])
				SetWinsize(bashf.Fd(), w, h)
				// Responding true (OK) here will let the client
				// know we have a pty ready for input
				req.Reply(true, nil)
			case "window-change":
				w, h := parseDims(req.Payload)
				SetWinsize(bashf.Fd(), w, h)
			}
		}
	}()
}

// =======================

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

// ======================

// Winsize stores the Height and Width of a terminal.
type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16 // unused
	y      uint16 // unused
}

// SetWinsize sets the size of the given pty.
//  from https://github.com/creack/termios/blob/master/win/win.go
func SetWinsize(fd uintptr, w, h uint32) {
	ws := &Winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}

func chomp(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != '\n' {
			return b[:i+1]
		}
	}
	return b[:0]
}
