package pelicantun

import (
	"fmt"
	"net"
	"sync"
	"time"
)

//
// in this file: ServerRW and its essential members NetConnReader and NetConnWriter
//

// NB to make goroutine stack traces interpretable, rw.go
// is used for the ReverseProxy (server) implementation, and
// crw.go (a duplicate of rw.go renamed as ClientRW) is used
// for the client (forward proxy).

// ===========================================
//
//                ServerRW
//
// ===========================================
//
// ServerRW packages up a reader and a writer for a specific
// net.Conn connection along with a copy of that net connection.
//
// The NetConnReader and the NetConnWriter work as a pair to
// move data from a net.Conn into the corresponding channels
// supplied by SendCh() and RecvCh() for access to the connection
// via channels/goroutines.
//
type ServerRW struct {
	conn            net.Conn
	r               *NetConnReader
	w               *NetConnWriter
	upReadToDnWrite chan []byte // can only receive []byte from upstream
	dnReadToUpWrite chan []byte // can only send []byte to upstream
}

// make a new ServerRW, passing bufsz to NewNetConnReader(). If the notifyWriterDone
// and/or notifyReaderDone channels are not nil, then they will
// receive a pointer to the NetConnReader (NetConnWriter) at Stop() time.
//
func NewServerRW(netconn net.Conn, bufsz int, notifyReaderDone chan *NetConnReader, notifyWriterDone chan *NetConnWriter) *ServerRW {

	// buffered channels here are important: we want
	// exactly buffered channel semantics: don't block
	// on typical access, until we are full up and then
	// we do need the backpressure of blocking.
	upReadToDnWrite := make(chan []byte, 1000)
	dnReadToUpWrite := make(chan []byte, 1000)

	s := &ServerRW{
		conn:            netconn,
		r:               NewNetConnReader(netconn, dnReadToUpWrite, bufsz, notifyReaderDone),
		w:               NewNetConnWriter(netconn, upReadToDnWrite, notifyWriterDone),
		upReadToDnWrite: upReadToDnWrite,
		dnReadToUpWrite: dnReadToUpWrite,
	}
	return s
}

// Start the ServerRW service.
func (s *ServerRW) Start() {
	s.r.Start()
	s.w.Start()
}

// Close is the same as Stop(). Both shutdown the running ServerRW service.
// Start must have been called first.
func (s *ServerRW) Close() {
	s.Stop()
}

// Stop the ServerRW service. Start must be called prior to Stop.
func (s *ServerRW) Stop() {
	s.r.Stop()
	s.w.Stop()
	s.conn.Close()
}

func (s *ServerRW) StopWithoutNotify() {
	s.r.StopWithoutNotify()
	s.w.StopWithoutNotify()
	s.conn.Close()
}

// can only be used to send to internal net.Conn
func (s *ServerRW) SendCh() chan []byte {
	return s.w.SendToDownCh()
}

// can only be used to recv from internal net.Conn
func (s *ServerRW) RecvCh() chan []byte {
	return s.r.RecvFromDownCh()
}

func (s *ServerRW) SendToDownCh() chan []byte {
	return s.w.SendToDownCh()
}

func (s *ServerRW) RecvFromDownCh() chan []byte {
	return s.r.RecvFromDownCh()
}

func (s *ServerRW) IsDone() bool {
	return s.r.IsDone() && s.w.IsDone()
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

// ===========================================
//
//              NetConnReader
//
// ===========================================
//
// NetConnReader and NetConnWriter work as a pair to
// move data from a net.Conn into go channels. Each
// maintains its own independent goroutine.
//
// NetConnReader represents a goroutine dedicated to
// reading from conn and writing to the dnReadToUpWrite channel.
//
// NetConnReader is used as the downstream most reader in the
// reverse proxy.  It is also used as the most upstream reader
// in the forward (socks) proxy. Thus in the socks proxy,
// the dnReadToUpWrite channel should be actually called
// upReadToDnWrite, assuming the client is upstream and
// the server is downstream. Hence the names are meaningful
// only in the reverse proxy context.
//
type NetConnReader struct {
	reqStop chan bool
	Done    chan bool
	Ready   chan bool
	LastErr error

	bufsz   int
	conn    net.Conn
	timeout time.Duration

	// clients of NewConnReader should get access to the channel
	// via calling RecvFromDownCh() so we can nil the channel when
	// the downstream server is unavailable.
	// dnReadToUpWrite will have some capacity, which gives
	// us backpressure.
	dnReadToUpWrite chan []byte // can only send []byte upstream

	// report to the one user of NetConnReader that we have stopped
	// over notifyDoneCh, iff reportDone is true.
	notifyDoneCh chan *NetConnReader
	reportDone   bool
	mut          sync.Mutex
}

// NetConnReaderDefaultBufSizeBytes declares the default read buffer size.
// It may be overriden in the call to NewnetConnReader by setting the bufsz
// parameter.
//
const NetConnReaderDefaultBufSizeBytes = 4 * 1024 // 4K

// make a new NetConnReader. if bufsz is 0 then we default
// to using a buffer of size NetConnReaderDefaultBufSizeBytes.
func NewNetConnReader(
	netconn net.Conn,
	dnReadToUpWrite chan []byte,
	bufsz int,
	notifyDone chan *NetConnReader) *NetConnReader {

	if bufsz <= 0 {
		bufsz = NetConnReaderDefaultBufSizeBytes
	}

	s := &NetConnReader{
		Done:            make(chan bool),
		reqStop:         make(chan bool),
		Ready:           make(chan bool),
		dnReadToUpWrite: dnReadToUpWrite,

		conn:         netconn,
		timeout:      10 * time.Millisecond,
		bufsz:        bufsz,
		notifyDoneCh: notifyDone,
	}
	if s.notifyDoneCh != nil {
		s.reportDone = true
	}
	return s
}

// return the internal s.dnReadToUpWrite channel which allows
// clients of NetConnReader to receive data from the downstream
// server.
func (s *NetConnReader) RecvFromDownCh() chan []byte {
	select {
	case <-s.reqStop:
		return nil
	case <-s.Done:
		return nil
	default:
		return s.dnReadToUpWrite
	}
}

func (s *NetConnReader) finish() {
	s.RequestStop()

	// if clients cached this, problem b/c they'll get lots of spurious receives: close(s.dnReadToUpWrite)
	s.dnReadToUpWrite = nil

	if s.reportDone && s.notifyDoneCh != nil {
		s.notifyDoneCh <- s
	}

	po("rw reader %p shut down complete, last error: '%v'\n", s, s.LastErr)
	close(s.Done)
}

// Start the NetConnReader. Use Stop() to shut it down.
func (s *NetConnReader) Start() {
	// read from conn and
	// write to dnReadToUpWrite channel
	go func() {

		// all exit paths should cleanup properly
		defer func() {
			s.finish()
		}()

		buf := make([]byte, s.bufsz)
		for {
			if s.IsStopRequested() {
				return
			}

			err := s.conn.SetReadDeadline(time.Now().Add(s.timeout))
			panicOn(err)

			n64, err := s.conn.Read(buf) // 010 is looping her, trying to read in rev.
			if IsTimeout(err) {
				if n64 != 0 {
					panic(fmt.Errorf("unexpected: got timeout and read of %d bytes back", n64))
				}
				continue
			}

			if err != nil {
				s.LastErr = err
				po("rw reader %p got error '%s', shutting down\n", s, err)
				return // shuts us down
			}

			if n64 == 0 {
				continue
			}
			po("NetConnReader %p got buf: '%s', of len n64=%d\n", s, string(buf[:n64]), n64)

			buf = buf[:n64]

			select {
			// backpressure gets applied here. When buffer channel dnReadToUpWrite
			// is full, we will block until the consumer end of this
			// channel makes progress.
			case s.dnReadToUpWrite <- buf:
			case <-s.reqStop:
				// avoid deadlock on shutdown
				return
			}

			// create a new buf now that the last one has been
			// sent and is now owned by the receiver.
			buf = make([]byte, s.bufsz)
		}
	}()
}

// Stop the NetConnReader goroutine. Start() must have been called
// first or this will hang your program.
func (s *NetConnReader) Stop() {
	s.RequestStop()
	<-s.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *NetConnReader) RequestStop() bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	select {
	case <-s.reqStop:
		return false
	default:
		close(s.reqStop)
		po("%p rw NetConnReader req.Stop closed", s)
		return true
	}
}

// Stops the reader and reports a pointer to itself on the notifyDoneCh
// if supplied with NewNetConnReader.
func (s *NetConnReader) StopAndNotify() {
	s.reportDone = true
	s.Stop()
}

// Stop the reader without reporting on notifyDoneCh.
func (s *NetConnReader) StopWithoutNotify() {
	s.reportDone = false
	s.Stop()
}

func (r *NetConnReader) IsStopRequested() bool {
	r.mut.Lock()
	defer r.mut.Unlock()

	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *NetConnReader) IsDone() bool {
	select {
	case <-r.Done:
		return true
	default:
		return false
	}
}

// ===========================================
//
//                NetConnWriter
//
// ===========================================
//
// NetConnWriter is the downstream most writer in the reverse proxy.
// It represents a goroutine dedicated to reading from UpReadToDnWrite
// channel and then writing conn.
type NetConnWriter struct {
	reqStop chan bool
	Done    chan bool
	Ready   chan bool
	LastErr error

	conn            net.Conn
	upReadToDnWrite chan []byte // can only receive []byte from upstream
	timeout         time.Duration

	// report to the one user of NetConnWriter that we have stopped
	// over notifyDoneCh, iff reportDone is true.
	notifyDoneCh chan *NetConnWriter
	reportDone   bool
	mut          sync.Mutex
}

// make a new NetConnWriter
func NewNetConnWriter(netconn net.Conn, upReadToDnWrite chan []byte, notifyDone chan *NetConnWriter) *NetConnWriter {
	s := &NetConnWriter{
		Done:            make(chan bool),
		reqStop:         make(chan bool),
		Ready:           make(chan bool),
		conn:            netconn,
		upReadToDnWrite: upReadToDnWrite,
		timeout:         40 * time.Millisecond,
		notifyDoneCh:    notifyDone,
	}
	if s.notifyDoneCh != nil {
		s.reportDone = true
	}
	return s
}

// returns the channel on which to send data to the downstream server.
func (s *NetConnWriter) SendToDownCh() chan []byte {
	select {
	case <-s.reqStop:
		return nil
	case <-s.Done:
		return nil
	default:
		return s.upReadToDnWrite
	}
}

func (s *NetConnWriter) finish() {
	s.RequestStop()

	//always leave open, don't close: close(s.upReadToDnWrite)
	s.upReadToDnWrite = nil

	if s.reportDone && s.notifyDoneCh != nil {
		s.notifyDoneCh <- s
	}

	po("rw writer %p shut down complete, last error: '%v'\n", s, s.LastErr)
	close(s.Done)
}

// Start the NetConnWriter.
func (s *NetConnWriter) Start() {

	// read from upReadToDnWrite and write to conn
	go func() {
		defer func() {
			// proper cleanup on all exit paths
			s.finish()
		}()

		var err error
		var n int
		var buf []byte

		for {

			select { // 010 is blocked here
			case buf = <-s.upReadToDnWrite:
			case <-s.reqStop:
				return
			}

			// we never stop trying to deliver, but we timeout
			// to check for shutdown requests
			err = s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
			panicOn(err)

			nbuf := len(buf)

		tryloop:
			for {
				n, err = s.conn.Write(buf)
				if err == nil {
					if n != nbuf {
						panic(fmt.Errorf("short write of %d bytes when expected full %d bytes", n, nbuf))
					}

					// successful write, leave the loop
					break tryloop
				}

				if IsTimeout(err) {
					buf = buf[n:]
					if len(buf) == 0 {
						// weird that we still timed out...? go with it.
						break tryloop
					}
					// else keep trying

					// check for request to shutdown
					if s.IsStopRequested() {
						return
					}
					continue tryloop
				}

				if err != nil && !IsTimeout(err) {
					s.LastErr = err // okay for io.EOF; don't close the conn since reader may be using.
					return
				}
			} // end try loop
		}
	}()

}

// Stop the NetConnWriter. Start() must have been called first or else
// you will hang your program waiting for s.Done to be closed.
func (s *NetConnWriter) Stop() {
	s.RequestStop()
	<-s.Done
}

func (r *NetConnWriter) IsStopRequested() bool {
	r.mut.Lock()
	defer r.mut.Unlock()

	select {
	case <-r.reqStop:
		return true
	default:
		return false
	}
}

func (r *NetConnWriter) IsDone() bool {
	select {
	case <-r.Done:
		return true
	default:
		return false
	}
}

// Stops the writer and reports a pointer to itself on the notifyDoneCh
// if supplied with NewNetConnWriter.
func (s *NetConnWriter) StopAndNotify() {
	s.reportDone = true
	s.Stop()
}

// Stop the writer without reporting on notifyDoneCh.
func (s *NetConnWriter) StopWithoutNotify() {
	s.reportDone = false
	s.Stop()
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *NetConnWriter) RequestStop() bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	select {
	case <-s.reqStop:
		return false
	default:
		close(s.reqStop)
		po("%p rw NetConnWriter req.Stop closed", s)
		return true
	}
}
