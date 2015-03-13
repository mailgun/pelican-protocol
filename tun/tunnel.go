package pelicantun

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// A LongPoller (aka tunnel) connects the http client (our pelican socks proxy)
// with the downstream target, typically an http server or sshd.
//
// A LongPoller represents a 1:1, one client to one server connection,
// if you ignore the socks-proxy and reverse-proxy in the middle.
// A ReverseProxy can have many LongPollers, mirroring the number of
// connections on the client side to the socks proxy. The key
// distinguishes them. The LongerPoller is where we implement the
// server side of the long polling.
//
type LongPoller struct {
	ReqStop           chan bool
	Done              chan bool
	ClientPacketRecvd chan *tunnelPacket

	rw        *RW // manage the goroutines that read and write dnConn
	recvCount int
	conn      net.Conn

	// server issues a unique key for the connection, which allows multiplexing
	// of multiple client connections from this one ip if need be.
	// The ssh integrity checks inside the tunnel prevent malicious tampering.
	key string

	Dest Addr
}

func NewLongPoller(dest Addr) *LongPoller {
	key := GenPelicanKey()
	dest.SetIpPort()

	s := &LongPoller{
		ReqStop:           make(chan bool),
		Done:              make(chan bool),
		ClientPacketRecvd: make(chan *tunnelPacket),
		key:               string(key),
		Dest:              dest,
	}

	return s
}

func (s *LongPoller) Stop() {
	// avoid double closing ReqStop here
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	<-s.Done
}

func (s *LongPoller) finish() {
	s.rw.Stop()
	close(s.Done)
}

// LongPoller::Start() implements the long-polling logic.
//
// When a new client request comes in (2nd one), we bump any
// already waiting long-poll into replying to its request.
//
//     new reader ------> bumps waiting/long-polling reader & takes its place.
//       ^                      |
//       |                      V
//       ^                      |
//       |                      V
//    client <-- returns to <---/
//
// it's a closed loop track with only one goroutine per tunnel
// actively holding on a long poll.
//
// There are only ever two client requests outstanding.
//
func (s *LongPoller) Start() error {

	err := s.dial()
	if err != nil {
		return fmt.Errorf("LongPoller could not dial '%s': '%s'", s.Dest.IpPort, err)
	}

	// s.dial() sets s.conn on success.
	s.rw = NewRW(s.conn, 0)
	s.rw.Start()

	go func() {
		defer func() { s.finish() }()

		var pack *tunnelPacket

		for {
			if pack == nil {
				// special case of first time through: no client packet has arrived.
				//
				// Or: we've replied to our last packet because the server had
				// something to say, and thus we have no pending packet available
				// for when the server has something more to say.
				//
				// In either case, we can't grab content from the downstream
				// server until we have a client packet to reply with.
				select {
				case pack = <-s.ClientPacketRecvd:
				case <-s.ReqStop:
					return
				}
			}

			// INVAR: pack is not nil
			if pack == nil {
				panic("pack should never nil at this point")
			}

			po("in tunnel::handle(pack) with pack = '%#v'\n", pack)
			// read from the request body and write to the ResponseWriter

			wait := 10 * time.Second
			select {
			case s.rw.SendToDownCh() <- pack.body:
			case <-time.After(wait):
				po("unable to send to downstream in receiveOnPacket after '%v'; aborting\n", wait)
				return
			}

			// read out of the buffer and write it to dnConn
			pack.resp.Header().Set("Content-type", "application/octet-stream")

			// wait for a read for a possibly long duration. this is the "long poll" part.
			dur := 30 * time.Second
			// the client will spin up another goroutine/thread/sender if it has
			// an additional send in the meantime.

			var n64 int64
			longPollTimeUp := time.After(dur)

			select {
			case <-s.ReqStop:
				close(pack.done)
				pack = nil
				return

			case b500 := <-s.rw.RecvFromDownCh():
				n64 += int64(len(b500))
				_, err := pack.resp.Write(b500)
				if err != nil {
					panic(err)
				}

				_, err = pack.respdup.Write(b500)
				if err != nil {
					panic(err)
				}
				close(pack.done)
				pack = nil

			case <-longPollTimeUp:
				// send it along its way anyhow
				close(pack.done)
				pack = nil

			case newpacket := <-s.ClientPacketRecvd:
				s.recvCount++
				// finish previous packet without data, because client sent another packet
				close(pack.done)
				pack = newpacket
			}
		}
	}()
	return nil
}

func (s *LongPoller) dial() error {

	po("ReverseProxy::NewTunnel: Attempting connect to our target '%s'\n", s.Dest.IpPort)
	dialer := net.Dialer{
		Timeout:   1000 * time.Millisecond,
		KeepAlive: 30 * time.Second,
	}

	var err error
	s.conn, err = dialer.Dial("tcp", s.Dest.IpPort)
	switch err.(type) {
	case *net.OpError:
		if strings.HasSuffix(err.Error(), "connection refused") {
			// could not reach destination
			return err
		}
	default:
		panicOn(err)
	}

	return err
}
