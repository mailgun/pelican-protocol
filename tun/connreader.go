package pelicantun

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

const bufSize = 1024

type ConnReader struct {
	conn net.Conn
	//readCh chan []byte
	bufsz int

	Ready      chan bool
	ReqStop    chan bool
	Done       chan bool
	key        string
	notifyDone chan *ConnReader
	noReport   bool
	dest       Addr
}

func NewConnReader(conn net.Conn, bufsz int, key string, notifyDone chan *ConnReader, dest Addr) *ConnReader {
	re := &ConnReader{
		conn:       conn,
		bufsz:      bufsz,
		Ready:      make(chan bool),
		ReqStop:    make(chan bool),
		Done:       make(chan bool),
		key:        key,
		notifyDone: notifyDone,
		dest:       dest,
	}
	return re
}

func (r *ConnReader) IsStopRequested() bool {
	select {
	case <-r.ReqStop:
		return true
	default:
		return false
	}
}

func (r *ConnReader) Stop() {
	if r.IsStopRequested() {
		return
	}
	close(r.ReqStop)
	<-r.Done
}

func (r *ConnReader) StopWithoutReporting() {
	r.noReport = true
	r.Stop()
}

func (r *ConnReader) Start() {
	go func() {
		close(r.Ready)
		//fmt.Printf("\n debug: ConnReader::Start %p starting!!\n", r)

		// Insure we close r.Done when exiting this goroutine.
		defer func() {
			//fmt.Printf("\n debug: ConnReader::Start %p finished!!\n", r)
			if !r.noReport {
				r.notifyDone <- r
			}
			close(r.Done)
		}()

		for {
			b := make([]byte, r.bufsz)

			// read deadline needed here
			const clientReadTimeoutMsec = 100
			err := r.conn.SetReadDeadline(time.Now().Add(time.Millisecond * clientReadTimeoutMsec))
			panicOn(err)

			n, err := r.conn.Read(b)
			if err != nil {
				//fmt.Printf("\n debug: ConnReader got error '%s' reading from r.reader. Shutting down.\n", err)
				// typical: "debug: ConnReader got error 'EOF' reading from r.reader. Shutting down."
				if !r.IsStopRequested() {
					close(r.ReqStop)
				}
				return
			}

			err = r.sendThenRecv(r.dest, r.key, bytes.NewBuffer(b[:n]))
			if err != nil {
				po("ConnReader loop: error during sendThenRecv: '%s'", err)
			}

			if r.IsStopRequested() {
				return
			}
		}
	}()
}

func (reader *ConnReader) sendThenRecv(dest Addr, key string, buf *bytes.Buffer) error {
	// write buf to new http request, starting with key

	//po("\n\n debug: sendThenRecv called with dest: '%#v', key: '%s', and buf: '%s'\n", dest, key, string(buf.Bytes()))

	if dest.IpPort == "" {
		return fmt.Errorf("dest.IpPort was empty the string")
	}
	if dest.Port == 0 {
		return fmt.Errorf("dest.Port was 0")
	}

	if key == "" || len(key) != KeyLen {
		return fmt.Errorf("sendThenRecv error: key '%s' was not of expected length %d", key, KeyLen)
	}

	req := bytes.NewBuffer([]byte(key))
	buf.WriteTo(req) // drains buf into req
	resp, err := http.Post(
		"http://"+dest.IpPort+"/",
		"application/octet-stream",
		req)
	defer func() {
		if resp != nil && resp.Body != nil {
			ioutil.ReadAll(resp.Body) // read anything leftover, so connection can be reused.
			resp.Body.Close()
		}
	}()

	if err != nil && err != io.EOF {
		log.Println(err.Error())
		//continue
		return err
	}

	// write http response to conn

	// we take apart the io.Copy to print out the response for debugging.
	//_, err = io.Copy(conn, resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	panicOn(err)
	po("client: resp.Body = '%s'\n", string(body))
	_, err = reader.conn.Write(body)
	panicOn(err)
	return nil
}
