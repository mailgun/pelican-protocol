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
	ReqStop chan bool
	Done    chan bool
	Ready   chan bool
	LastErr error

	bufsz   int
	conn    net.Conn
	timeout time.Duration

	// clients of NewConnReader should get access to the channel
	// via calling RecvFromDownCh() so we can nil the channel when
	// the downstream server is unavailable.
	dnReadToUpWrite chan []byte // can only send []byte upstream

	// report to the one user of NetConnReader that we have stopped
	// over notifyDone, iff reportDone is true.
	notifyDone chan *NetConnReader
	reportDone bool
}

// NetConnReaderDefaultBufSizeBytes declares the default read buffer size.
// It may be overriden in the call to NewnetConnReader by setting the bufsz
// parameter.
//
const NetConnReaderDefaultBufSizeBytes = 4 * 1024 // 4K

// make a new NetConnReader. if bufsz is 0 then we default
// to using a buffer of size NetConnReaderDefaultBufSizeBytes.
func NewNetConnReader(netconn net.Conn, dnReadToUpWrite chan []byte, bufsz int, notifyDone chan *NetConnReader) *NetConnReader {
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
		notifyDone:      notifyDone,
	}
}

// return the internal s.dnReadToUpWrite channel which allows
// clients of NetConnReader to receive data from the downstream
// server.
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
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	close(s.dnReadToUpWrite)
	s.dnReadToUpWrite = nil

	if s.reportDone && s.notifyDone != nil {
		s.notifyDone <- s
	}
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

		for {
			if s.IsStopRequested() {
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
				s.LastErr = err
				return // shuts us down
			}

			select {
			case s.dnReadToUpWrite <- buf[:n64]:
			case <-s.ReqStop:
				return
			}

		}
	}()
}

// Stop the NetConnReader goroutine. Start() must have been called
// first or this will hang your program.
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

	// report to the one user of NetConnWriter that we have stopped
	// over notifyDone, iff reportDone is true.
	notifyDone chan *NetConnWriter
	reportDone bool
}

// make a new NetConnWriter
func NewNetConnWriter(netconn net.Conn, upReadToDnWrite chan []byte, notifyDone chan *NetConnWriter) *NetConnWriter {
	return &NetConnWriter{
		Done:            make(chan bool),
		ReqStop:         make(chan bool),
		Ready:           make(chan bool),
		conn:            netconn,
		upReadToDnWrite: upReadToDnWrite,
		timeout:         40 * time.Millisecond,
		notifyDone:      notifyDone,
	}
}

// returns the channel on which to send data to the downstream server.
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
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	close(s.upReadToDnWrite)
	s.upReadToDnWrite = nil

	if s.reportDone && s.notifyDone != nil {
		s.notifyDone <- s
	}
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
					s.LastErr = err // okay for io.EOF; don't close the conn since reader may be using.
					return
				}
			} // end try loop

			if !wroteOk {
				panic("internal program logic error: should never get here if could not write!")
			}

		}
	}()

}

// Stop the NetConnWriter. Start() must have been called first or else
// you will hang your program waiting for s.Done to be closed.
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
// net.Conn connection. It contains both a
// NetConnReader and a NetConnWriter; these work as a pair to
// move data from a net.Conn into the corresponding channels
// upReadToDnWrite and dnReadToUpWrite.
//
type RW struct {
	conn            net.Conn
	r               *NetConnReader
	w               *NetConnWriter
	upReadToDnWrite chan []byte // can only receive []byte from upstream
	dnReadToUpWrite chan []byte // can only send []byte to upstream
}

// make a new RW, passing bufsz to NewNetConnReader().
func NewRW(netconn net.Conn, bufsz int) *RW {

	upReadToDnWrite := make(chan []byte)
	dnReadToUpWrite := make(chan []byte)

	s := &RW{
		conn:            netconn,
		r:               NewNetConnReader(netconn, dnReadToUpWrite, bufsz, nil),
		w:               NewNetConnWriter(netconn, upReadToDnWrite, nil),
		upReadToDnWrite: upReadToDnWrite,
		dnReadToUpWrite: dnReadToUpWrite,
	}
	return s
}

// Start the RW service.
func (s *RW) Start() {
	s.r.Start()
	s.w.Start()
}

// Close is the same as Stop(). Both shutdown the running RW service.
// Start must have been called first.
func (s *RW) Close() {
	s.Stop()
}

// Stop the RW service. Start must be called prior to Stop.
func (s *RW) Stop() {
	s.r.Stop()
	s.w.Stop()
	s.conn.Close()
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
