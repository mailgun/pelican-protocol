package pelicantun

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

const bufSize = 1024

// take a reader, and turn it into a channel of bufSize chunks of []byte
func makeReadChan(r io.Reader, bufSize int) chan []byte {
	read := make(chan []byte)
	go func() {
		for {
			b := make([]byte, bufSize)
			n, err := r.Read(b)
			if err != nil {
				return
			}
			read <- b[0:n]
		}
	}()
	return read
}

type PelicanSocksProxyConfig struct {
	Listen           addr
	Dest             addr
	tickIntervalMsec int
}

type PelicanSocksProxy struct {
	Cfg     PelicanSocksProxyConfig
	ReqStop chan bool
	Done    chan bool
}

func NewPelicanSocksProxy(cfg PelicanSocksProxyConfig) *PelicanSocksProxy {

	p := &PelicanSocksProxy{
		Cfg:     cfg,
		ReqStop: make(chan bool),
		Done:    make(chan bool),
	}
	p.SetDefaultPorts()
	return p
}

func (f *PelicanSocksProxy) SetDefaultPorts() {
	if f.Cfg.Listen.Port == 0 {
		f.Cfg.Listen.Port = GetAvailPort()
	}
	if f.Cfg.Listen.Ip == "" {
		f.Cfg.Listen.Ip = "127.0.0.1"
	}
	f.Cfg.Listen.SetIpPort()

	if f.Cfg.Dest.Port == 0 {
		f.Cfg.Dest.Port = 80
	}
	if f.Cfg.Dest.Ip == "" {
		f.Cfg.Dest.Ip = "127.0.0.1"
	}
	f.Cfg.Dest.SetIpPort()
}

func (f *PelicanSocksProxy) Start() {
	go f.ListenAndServe()
}

func (f *PelicanSocksProxy) Stop() {
	f.ReqStop <- true
	close(f.Done)
}

func (f *PelicanSocksProxy) ListenAndServe() error {
	listener, err := net.Listen("tcp", f.Cfg.Listen.IpPort)
	if err != nil {
		panic(err)
	}
	log.Printf("listen on '%v', with revProxAddr '%v'", f.Cfg.Listen.IpPort, f.Cfg.Dest.IpPort)

	conn, err := listener.Accept()
	if err != nil {
		panic(err)
	}
	log.Println("accept conn", "localAddr.", conn.LocalAddr(), "remoteAddr.", conn.RemoteAddr())

	buf := new(bytes.Buffer)

	sendCount := 0

	// initiate new session and read key
	log.Println("Attempting connect HttpTun Server.", f.Cfg.Dest.IpPort)
	//buf.Write([]byte(*destAddr))
	resp, err := http.Post(
		"http://"+f.Cfg.Dest.IpPort+"/create",
		"text/plain",
		buf)
	panicOn(err)
	key, err := ioutil.ReadAll(resp.Body)
	panicOn(err)
	resp.Body.Close()

	log.Printf("client main(): after Post('/create') we got ResponseWriter with key = '%x'", key)

	// ticker to set a rate at which to hit the server
	tick := time.NewTicker(time.Duration(int64(f.Cfg.tickIntervalMsec)) * time.Millisecond)
	read := makeReadChan(conn, bufSize)
	buf.Reset()
	for {
		select {
		case <-f.ReqStop:
			close(f.Done)
			return nil
		case b := <-read:
			// fill buf here
			po("client: <-read of '%s'; hex:'%x' of length %d added to buffer\n", string(b), b, len(b))
			buf.Write(b)
			po("client: after write to buf of len(b)=%d, buf is now length %d\n", len(b), buf.Len())

		case <-tick.C:
			sendCount++
			po("\n ====================\n client sendCount = %d\n ====================\n", sendCount)
			po("client: sendCount %d, got tick.C. key as always(?) = '%x'. buf is now size %d\n", sendCount, key, buf.Len())
			// write buf to new http request, starting with key
			req := bytes.NewBuffer(key)
			buf.WriteTo(req)
			resp, err := http.Post(
				"http://"+f.Cfg.Dest.IpPort+"/ping",
				"application/octet-stream",
				req)
			if err != nil && err != io.EOF {
				log.Println(err.Error())
				continue
			}

			// write http response response to conn

			// we take apart the io.Copy to print out the response for debugging.
			//_, err = io.Copy(conn, resp.Body)

			body, err := ioutil.ReadAll(resp.Body)
			panicOn(err)
			po("client: resp.Body = '%s'\n", string(body))
			_, err = conn.Write(body)
			panicOn(err)
			resp.Body.Close()
		}
	}
}
