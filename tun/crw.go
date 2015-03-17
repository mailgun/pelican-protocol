package pelicantun

import (
	"net"
)

// NB to make goroutine stack traces interpretable, rw.go
// is used for the ReverseProxy (server) implementation, and
// crw.go (a duplicate of rw.go) is used for the client (forward proxy).

// the rw.go:RW implementation and this ClientRW implementation
// are other duplicates and should likely be kept in sync.

// ===========================================
//
//                 ClientRW
//
// ===========================================
//
// RW packages up a reader and a writer for a specific
// net.Conn connection along with a copy of that net connection.
//
// The NetConnReader and the NetConnWriter work as a pair to
// move data from a net.Conn into the corresponding channels
// supplied by SendCh() and RecvCh() for access to the connection
// via channels/goroutines.
//
type ClientRW struct {
	conn            net.Conn
	r               *NetConnReader
	w               *NetConnWriter
	upReadToDnWrite chan []byte // can only receive []byte from upstream
	dnReadToUpWrite chan []byte // can only send []byte to upstream
}

// make a new RW, passing bufsz to NewNetConnReader(). If the notifyWriterDone
// and/or notifyReaderDone channels are not nil, then they will
// receive a pointer to the NetConnReader (NetConnWriter) at Stop() time.
//
func NewClientRW(netconn net.Conn, bufsz int, notifyReaderDone chan *NetConnReader, notifyWriterDone chan *NetConnWriter) *ClientRW {

	// buffered channels here are important: we want
	// exactly buffered channel semantics: don't block
	// on typical access, until we are full up and then
	// we do need the backpressure of blocking.
	upReadToDnWrite := make(chan []byte, 1000)
	dnReadToUpWrite := make(chan []byte, 1000)

	s := &ClientRW{
		conn:            netconn,
		r:               NewNetConnReader(netconn, dnReadToUpWrite, bufsz, notifyReaderDone),
		w:               NewNetConnWriter(netconn, upReadToDnWrite, notifyWriterDone),
		upReadToDnWrite: upReadToDnWrite,
		dnReadToUpWrite: dnReadToUpWrite,
	}
	return s
}

// Start the ClientRW service.
func (s *ClientRW) Start() {
	s.r.Start()
	s.w.Start()
}

// Close is the same as Stop(). Both shutdown the running ClientRW service.
// Start must have been called first.
func (s *ClientRW) Close() {
	s.Stop()
}

// Stop the ClientRW service. Start must be called prior to Stop.
func (s *ClientRW) Stop() {
	s.r.Stop()
	s.w.Stop()
	s.conn.Close()
}

func (s *ClientRW) StopWithoutNotify() {
	s.r.StopWithoutNotify()
	s.w.StopWithoutNotify()
	s.conn.Close()
}

// can only be used to send to internal net.Conn
func (s *ClientRW) SendCh() chan []byte {
	return s.w.SendToDownCh()
}

// can only be used to recv from internal net.Conn
func (s *ClientRW) RecvCh() chan []byte {
	return s.r.RecvFromDownCh()
}

func (s *ClientRW) SendToDownCh() chan []byte {
	return s.w.SendToDownCh()
}

func (s *ClientRW) RecvFromDownCh() chan []byte {
	return s.r.RecvFromDownCh()
}

func (s *ClientRW) IsDone() bool {
	return s.r.IsDone() && s.w.IsDone()
}
