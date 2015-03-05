package pelicantun

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type ReverseProxy struct {
	Cfg ReverseProxyConfig
}

type ReverseProxyConfig struct {
	destIP     string
	destPort   string
	destAddr   string
	listenAddr string
}

const (
	readTimeoutMsec = 1000
)

type proxy struct {
	C         chan proxyPacket
	key       string
	conn      net.Conn
	recvCount int
}

type proxyPacket struct {
	resp    http.ResponseWriter
	request *http.Request
	body    []byte
	done    chan bool
}

// print out shortcut
var po = VPrintf

func NewProxy(key, destAddr string) (p *proxy, err error) {
	po("starting with NewProxy\n")
	p = &proxy{C: make(chan proxyPacket), key: key, recvCount: 0}
	log.Println("Attempting connect", destAddr)
	p.conn, err = net.Dial("tcp", destAddr)
	panicOn(err)

	err = p.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	panicOn(err)

	log.Println("ResponseWriter directed to ", destAddr)
	po("done with NewProxy\n")
	return
}

func (p *proxy) handle(pp proxyPacket) {
	p.recvCount++
	po("\n ====================\n server proxy.recvCount = %d    len(pp.body)= %d\n ================\n", p.recvCount, len(pp.body))

	po("in proxy::handle(pp) with pp = '%#v'\n", pp)
	// read from the request body and write to the ResponseWriter
	writeMe := pp.body[KeyLen:]
	n, err := p.conn.Write(writeMe)
	if n != len(writeMe) {
		log.Printf("proxy::handle(pp): could only write %d of the %d bytes to the connection. err = '%v'", n, len(pp.body), err)
	} else {
		po("proxy::handle(pp): wrote all %d bytes of writeMe to the final (sshd server) connection: '%s'.", len(writeMe), string(writeMe))
	}
	pp.request.Body.Close()
	if err == io.EOF {
		p.conn = nil
		log.Printf("proxy::handle(pp): EOF for key '%x'", p.key)
		return
	}
	// read out of the buffer and write it to conn
	pp.resp.Header().Set("Content-type", "application/octet-stream")
	// temp for debug: n64, err := io.Copy(pp.resp, p.conn)

	b500 := make([]byte, 1<<17)

	err = p.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	panicOn(err)

	n64, err := p.conn.Read(b500)
	if err != nil {
		// i/o timeout expected
	}
	po("\n\n server got reply from p.conn of len %d: '%s'\n", n64, string(b500[:n64]))
	_, err = pp.resp.Write(b500[:n64])
	if err != nil {
		panic(err)
	}

	// don't panicOn(err)
	log.Printf("proxy::handle(pp): io.Copy into pp.resp from p.conn moved %d bytes", n64)
	pp.done <- true
	po("proxy::handle(pp) done.\n")
}

var queue = make(chan proxyPacket)
var createQueue = make(chan *proxy)

func handler(c http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	panicOn(err)
	po("top level handler(): in '/' and '/ping' handler, packet len without key: %d: making new proxyPacket, http.Request r = '%#v'. r.Body = '%s'\n", len(body)-KeyLen, *r, string(body))

	pp := proxyPacket{
		resp:    c,
		request: r,
		body:    body, // includes key of KeyLen in prefix
		done:    make(chan bool),
	}
	queue <- pp
	<-pp.done // wait until done before returning, as this will return anything written to c to the client.
}

func (s *ReverseProxy) createHandler(c http.ResponseWriter, r *http.Request) {
	// fix destAddr on server side to prevent being a transport for other actions.

	// destAddr used to be here, but no more. Still have to close the body.
	_, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(c, "Could not read destAddr",
			http.StatusInternalServerError)
		return
	}

	key := genKey()
	po("in createhandler(): Server::createHandler generated key '%s'\n", key)

	p, err := NewProxy(key, s.Cfg.destAddr)
	if err != nil {
		http.Error(c, "Could not connect",
			http.StatusInternalServerError)
		return
	}
	po("Server::createHandler about to send createQueue <- p, where p = %p\n", p)
	createQueue <- p
	po("Server::createHandler(): sent createQueue <- p.\n")

	c.Write([]byte(key))
	po("Server::createHandler done.\n")
}

func proxyMuxer() {
	po("proxyMuxer started\n")
	proxyMap := make(map[string]*proxy)
	for {
		select {
		case pp := <-queue:
			key := make([]byte, KeyLen)
			// read key
			//n, err := pp.req.Body.Read(key)
			if len(pp.body) < KeyLen {
				log.Printf("Couldn't read key, not enough bytes in body. len(body) = %d\n", len(pp.body))
				continue
			}
			copy(key, pp.body)

			po("proxyMuxer: from pp <- queue, we read key '%x'\n", key)
			// find proxy
			p, ok := proxyMap[string(key)]
			if !ok {
				log.Printf("Couldn't find proxy for key = '%x'", key)
				continue
			}
			// handle
			po("proxyMuxer found proxy for key '%x'\n", key)
			p.handle(pp)
		case p := <-createQueue:
			po("proxyMuxer: got p=%p on <-createQueue\n", p)
			proxyMap[p.key] = p
			po("proxyMuxer: after adding key '%x', proxyMap is now: '%#v'\n", p.key, proxyMap)
		}
	}
	po("proxyMuxer done\n")
}

func NewReverseProxy(cfg ReverseProxyConfig) *ReverseProxy {
	return &ReverseProxy{
		Cfg: cfg,
	}
}

func (s *ReverseProxy) ListenAndServe() {

	go proxyMuxer()

	http.HandleFunc("/", handler)
	http.HandleFunc("/create", s.createHandler)
	fmt.Printf("about to ListenAndServer on listenAddr '%#v'. Ultimate destAddr: '%s'\n",
		s.Cfg.listenAddr, s.Cfg.destAddr)
	err := http.ListenAndServe(s.Cfg.listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

func genKey() string {
	key := make([]byte, KeyLen)
	for i := 0; i < KeyLen; i++ {
		key[i] = byte(rand.Int())
	}
	return string(key)
}
