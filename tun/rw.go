package pelicantun

import (
	"fmt"
	"net"
	"time"
)

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

// NetConnReader is the downstream most reader in the reverse proxy.
// It represents a goroutine dedicated to reading from conn and
// writing to the DnReadToUpWrite channel.
type NetConnReader struct {
	ReqStop chan bool
	Done    chan bool
	Ready   chan bool

	bufsz   int
	conn    net.Conn
	timeout time.Duration

	// clients of NewConnReader should get access to the channel
	// via calling RecvFromDownCh() so we can nil the channel when
	// the downstream server is unavailable.
	dnReadToUpWrite chan []byte // can only send []byte upstream
}

const NetConnReaderDefaultBufSizeBytes = 4 * 1024 // 4K

// make a new NetConnReader. if bufsz is 0 then we default
// to using a buffer of size NetConnReaderDefaultBufSizeBytes.
func NewNetConnReader(netconn net.Conn, dnReadToUpWrite chan []byte, bufsz int) *NetConnReader {
	if bufsz <= 0 {
		bufsz = NetConnReaderDefaultBufSizeBytes
	}

	return &NetConnReader{
		Done:            make(chan bool),
		ReqStop:         make(chan bool),
		Ready:           make(chan bool),
		conn:            netconn,
		dnReadToUpWrite: dnReadToUpWrite,
		timeout:         10 * time.Millisecond,
		bufsz:           bufsz,
	}
}

func (s *NetConnReader) RecvFromDownCh() chan []byte {
	select {
	case <-s.ReqStop:
		return nil
	case <-s.Done:
		return nil
	default:
		return s.dnReadToUpWrite
	}
}

func (s *NetConnReader) finish() {
	close(s.dnReadToUpWrite)
	close(s.Done)
}

func (s *NetConnReader) Start() {
	// read from conn and
	// write to dnReadToUpWrite channel
	go func() {
		for {
			if s.IsStopRequested() {
				s.finish()
				return
			}

			err := s.conn.SetReadDeadline(time.Now().Add(s.timeout))
			panicOn(err)

			buf := make([]byte, s.bufsz)

			n64, err := s.conn.Read(buf)
			if IsTimeout(err) {
				continue
			}

			if err != nil {
				panic(err)
			}

			select {
			case s.dnReadToUpWrite <- buf[:n64]:
			case <-s.ReqStop:
				s.finish()
				return
			}

		}
	}()
}

func (s *NetConnReader) Stop() {
	// avoid double closing ReqStop here
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	<-s.Done
}

// NetConnWriter is the downstream most writer in the reverse proxy.
// It represents a goroutine dedicated to reading from UpReadToDnWrite
// channel and then writing conn.
type NetConnWriter struct {
	ReqStop chan bool
	Done    chan bool
	Ready   chan bool
	LastErr error

	conn            net.Conn
	upReadToDnWrite chan []byte // can only receive []byte from upstream
	timeout         time.Duration
}

// make a new NetConnWriter
func NewNetConnWriter(netconn net.Conn, upReadToDnWrite chan []byte) *NetConnWriter {
	return &NetConnWriter{
		Done:            make(chan bool),
		ReqStop:         make(chan bool),
		Ready:           make(chan bool),
		conn:            netconn,
		upReadToDnWrite: upReadToDnWrite,
		timeout:         40 * time.Millisecond,
	}
}

func (s *NetConnWriter) SendToDownCh() chan []byte {
	select {
	case <-s.ReqStop:
		return nil
	case <-s.Done:
		return nil
	default:
		return s.upReadToDnWrite
	}
}

func (s *NetConnWriter) finish() {
	close(s.upReadToDnWrite)
	close(s.Done)
}

func (s *NetConnWriter) Start() {

	// read from upReadToDnWrite and write to conn
	go func() {
		defer func() {
			// proper cleanup on all exit paths
			s.finish()
		}()

		var err error
		var n int
		var wroteOk bool
		var buf []byte

		for {

			select {
			case buf = <-s.upReadToDnWrite:
			case <-s.ReqStop:
				return
			}

			// we never stop trying to deliver, but we timeout
			// to check for shutdown requests
			err = s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
			panicOn(err)

			nbuf := len(buf)
			wroteOk = false

		tryloop:
			for {
				n, err = s.conn.Write(buf)
				if err == nil {
					if n != nbuf {
						panic(fmt.Errorf("short write of %d bytes when expected full %d bytes", n, nbuf))
					}

					// successful write, leave the loop
					wroteOk = true
					break tryloop
				}

				if IsTimeout(err) {
					buf = buf[n:]
					if len(buf) == 0 {
						// weird that we still timed out...? go with it.
						wroteOk = true
						break tryloop
					}
					// else keep trying

					// check for request to shutdown
					if s.IsStopRequested() {
						return
					}
					continue
				}

				if err != nil && !IsTimeout(err) {
					panic(err)
					s.LastErr = err
					return
				}
			} // end try loop

			if !wroteOk {
				panic("internal program logic error: should never get here if could not write!")
			}

		}
	}()

}

func (s *NetConnWriter) Stop() {
	// avoid double closing ReqStop here
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	<-s.Done
}

// RW contains a reader and a writer for a specific
// downstream connection.
type RW struct {
	conn            net.Conn
	r               *NetConnReader
	w               *NetConnWriter
	UpReadToDnWrite chan []byte // can only receive []byte from upstream
	DnReadToUpWrite chan []byte // can only send []byte to upstream
}

// make a new RW, passing bufsz to NewNetConnReader().
func NewRW(c net.Conn, upReadToDnWrite chan []byte, dnReadToUpWrite chan []byte, bufsz int) *RW {
	s := &RW{
		conn:            c,
		r:               NewNetConnReader(c, dnReadToUpWrite, bufsz),
		w:               NewNetConnWriter(c, upReadToDnWrite),
		UpReadToDnWrite: upReadToDnWrite,
		DnReadToUpWrite: dnReadToUpWrite,
	}
	return s
}

func (s *RW) Start() {
	s.r.Start()
	s.w.Start()
}

func (s *RW) Close() {
	s.Stop()
	s.conn.Close()
}

func (s *RW) Stop() {
	s.r.Stop()
	s.w.Stop()
}

func (s *RW) SendToDownCh() chan []byte {
	return s.w.SendToDownCh()
}

func (s *RW) RecvFromDownCh() chan []byte {
	return s.r.RecvFromDownCh()
}

func (s *RW) IsDone() bool {
	return s.r.IsDone() && s.w.IsDone()
}

func (r *NetConnReader) IsStopRequested() bool {
	select {
	case <-r.ReqStop:
		return true
	default:
		return false
	}
}

func (r *NetConnWriter) IsStopRequested() bool {
	select {
	case <-r.ReqStop:
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

func (r *NetConnWriter) IsDone() bool {
	select {
	case <-r.Done:
		return true
	default:
		return false
	}
}

/* example code from rbuf

////////////////////////////

// Read():
//
// from bytes.Buffer.Read(): Read reads the next len(p) bytes
// from the buffer or until the buffer is drained. The return
// value n is the number of bytes read. If the buffer has no data
// to return, err is io.EOF (unless len(p) is zero); otherwise it is nil.
//
//  from the description of the Reader interface,
//     http://golang.org/pkg/io/#Reader
//
//
// Reader is the interface that wraps the basic Read method.
//
// Read reads up to len(p) bytes into p. It returns the number
// of bytes read (0 <= n <= len(p)) and any error encountered.
// Even if Read returns n < len(p), it may use all of p as scratch
// space during the call. If some data is available but not
// len(p) bytes, Read conventionally returns what is available
// instead of waiting for more.
//
// When Read encounters an error or end-of-file condition after
// successfully reading n > 0 bytes, it returns the number of bytes
// read. It may return the (non-nil) error from the same call or
// return the error (and n == 0) from a subsequent call. An instance
// of this general case is that a Reader returning a non-zero number
// of bytes at the end of the input stream may return
// either err == EOF or err == nil. The next Read should
// return 0, EOF regardless.
//
// Callers should always process the n > 0 bytes returned before
// considering the error err. Doing so correctly handles I/O errors
// that happen after reading some bytes and also both of the
// allowed EOF behaviors.
//
// Implementations of Read are discouraged from returning a zero
// byte count with a nil error, and callers should treat that
// situation as a no-op.
//
//
func (b *RW) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.closed == 0 {
		return 0, io.EOF
	}
	extent := b.Beg + b.Readable
	if extent <= b.N {
		n += copy(p, b.A[b.Use][b.Beg:extent])
	} else {
		n += copy(p, b.A[b.Use][b.Beg:b.N])
		if n < len(p) {
			n += copy(p[n:], b.A[b.Use][0:(extent%b.N)])
		}
	}
	if doAdvance {
		b.Advance(n)
	}
	return
}

//
// Write writes len(p) bytes from p to the underlying data stream.
// It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early.
// Write must return a non-nil error if it returns n < len(p).
//
func (b *RW) Write(p []byte) (n int, err error) {
	for {
		if len(p) == 0 {
			// nothing (left) to copy in; notice we shorten our
			// local copy p (below) as we read from it.
			return
		}

		writeCapacity := b.N - b.Readable
		if writeCapacity <= 0 {
			// we are all full up already.
			return n, io.ErrShortWrite
		}
		if len(p) > writeCapacity {
			err = io.ErrShortWrite
			// leave err set and
			// keep going, write what we can.
		}

		writeStart := (b.Beg + b.Readable) % b.N

		upperLim := intMin(writeStart+writeCapacity, b.N)

		k := copy(b.A[b.Use][writeStart:upperLim], p)

		n += k
		b.Readable += k
		p = p[k:]

		// we can fill from b.A[b.Use][0:something] from
		// p's remainder, so loop
	}
}
*/
