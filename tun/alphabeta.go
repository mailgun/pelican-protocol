package pelicantun

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Similar in spirit to Comet, Ajax-long-polling,
// and BOSH (http://en.wikipedia.org/wiki/BOSH),
// the following struct and methods for
// Chaser comprises the two-socket (two http transactions open
// at most at once) long-polling implementation for
// the client (pelican socks proxy) end. See tunnel.go
// and LongPoller for server side.
//
// The story: alpha and beta are a pair of room-mates
// who hate to be home together. They represent our
// two possible http request-response sockets. The
// job of Chaser is to figure out when to initiate
// an http request.
//
// If alpha arrives home and beta is present,
// alpha kicks out beta and beta goes on a data
// retrieval mission. Even without data on the
// request, this mission allows the server to initiate
// data send by delaying the reply to the request
// for some time until data becomes available to
// send.
//
// When beta gets back if alpha is home, alpha
// is forced to go himself on a data retrieval mission.
//
// If they both find themselves at home at once, then the
// tie is arbitrarily broken and alpha goes (hence
// the name, 'alpha').
//
// In this way we implement the ping-pong of
// long-polling. Within the constraints of only
// having two http connections open, each party
// can send whenever they so desire, with as low
// latency as we can muster within the constraints
// of only using two http connections and the given
// traffic profile of pauses on either end.
//
// The actual logic is implemented in Home, which
// has its own goroutine. The startAlpha() and
// startBeta() methods each start their own
// goroutines respectively, and the three communicate
// over the channels held in Chaser and Home.
//
// See also the diagram in tunnel.go in front of
// the LongPoller struct description.
//
type Chaser struct {
	reqStop chan bool
	Done    chan bool

	incoming    chan []byte
	repliesHere chan []byte
	alphaIsHome bool
	betaIsHome  bool

	alphaArrivesHome chan bool
	betaArrivesHome  chan bool

	alphaDone   chan bool
	betaDone    chan bool
	monitorDone chan bool

	closedChan chan bool
	home       *ClientHome

	key  string
	dest Addr

	// this rw maintains the net.Conn to the upstream client
	rw           *ClientRW
	rwReaderDone chan *NetConnReader
	rwWriterDone chan *NetConnWriter

	notifyDone chan *Chaser
	skipNotify bool

	mut sync.Mutex
	cfg ChaserConfig

	httpClient *HttpClientWithTimeout

	// shutdown after a period on non-use
	shutdownInactiveDur time.Duration
	inactiveTimer       *time.Timer
	lastActiveTm        time.Time
	mutTimer            sync.Mutex

	nextSendSerialNumber     int64
	lastRecvSerialNumberSeen int64

	misorderedReplies map[int64]*SerResp

	tmLastRecv []time.Time
	tmLastSend []time.Time

	hist *HistoryLog
	name string
}

type ChaserConfig struct {
	ConnectTimeout      time.Duration
	TransportTimeout    time.Duration
	BufSize             int
	ShutdownInactiveDur time.Duration
}

func DefaultChaserConfig() *ChaserConfig {
	return &ChaserConfig{
		ConnectTimeout:   2000 * time.Millisecond,
		TransportTimeout: 60 * time.Second,
		//BufSize:             1 * 1024 * 1024,
		BufSize:             64 * 1024,
		ShutdownInactiveDur: 10 * time.Minute,
	}
}

func SetChaserConfigDefaults(cfg *ChaserConfig) {
	def := DefaultChaserConfig()
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = def.ConnectTimeout
	}
	if cfg.TransportTimeout == 0 {
		cfg.TransportTimeout = def.TransportTimeout
	}
	if cfg.BufSize == 0 {
		cfg.BufSize = def.BufSize
	}
	if cfg.ShutdownInactiveDur == 0 {
		cfg.ShutdownInactiveDur = def.ShutdownInactiveDur
	}
}

/*
func example_main() {
	c := NewChaser()
	c.Start()
	globalHome = c.home

	for i := 1; i < 100; i++ {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(i))

		c.incoming <- buf
		rsleep()
		rsleep()
		rsleep()
		rsleep()
	}

}
*/

func NewChaser(cfg ChaserConfig, conn net.Conn, key string, notifyDone chan *Chaser, dest Addr) *Chaser {

	if key == "" || len(key) != KeyLen {
		panic(fmt.Errorf("NewChaser() error: key '%s' was not of expected length %d. instead: %d", key, KeyLen, len(key)))
	}

	if dest.IpPort == "" {
		panic(fmt.Errorf("dest.IpPort was empty the string"))
	}
	if dest.Port == 0 {
		panic(fmt.Errorf("dest.Port was 0"))
	}

	SetChaserConfigDefaults(&cfg)

	rwReaderDone := make(chan *NetConnReader)
	rwWriterDone := make(chan *NetConnWriter)

	rw := NewClientRW("ClientRW/Chaser", conn, cfg.BufSize, rwReaderDone, rwWriterDone)

	s := &Chaser{
		rw:           rw,
		rwReaderDone: rwReaderDone,
		rwWriterDone: rwWriterDone,

		reqStop: make(chan bool),
		Done:    make(chan bool),

		alphaDone:   make(chan bool),
		betaDone:    make(chan bool),
		monitorDone: make(chan bool),
		incoming:    rw.RecvCh(), // requests to the remote http
		repliesHere: rw.SendCh(), // replies from remote http are passed upstream here.

		alphaIsHome:          true,
		betaIsHome:           true,
		closedChan:           make(chan bool),
		home:                 NewClientHome(),
		dest:                 dest,
		key:                  key,
		notifyDone:           notifyDone,
		cfg:                  cfg,
		httpClient:           NewHttpClientWithTimeout(cfg.TransportTimeout),
		shutdownInactiveDur:  cfg.ShutdownInactiveDur,
		inactiveTimer:        time.NewTimer(cfg.ShutdownInactiveDur),
		nextSendSerialNumber: 1,
		misorderedReplies:    make(map[int64]*SerResp),

		tmLastSend: make([]time.Time, 0),
		tmLastRecv: make([]time.Time, 0),

		hist: NewHistoryLog("Chaser"),
		name: "Chaser",
	}

	po("\n\n Chaser %p gets NewRW() = %p '%s' with %p NetConnReader and %p NetConnWriter. For conn = %s[remote] -> %s[local]\n\n", s, rw, rw.name, rw.r, rw.w, conn.RemoteAddr(), conn.LocalAddr())

	// always closed
	close(s.closedChan)

	return s
}

func (s *Chaser) ResetActiveTimer() {
	s.mutTimer.Lock()
	defer s.mutTimer.Unlock()
	s.inactiveTimer.Reset(s.shutdownInactiveDur)
	s.lastActiveTm = time.Now()
}

func (s *Chaser) Start() {
	s.home.Start()
	s.startMonitor()
	s.startAlpha()
	s.startBeta()
	s.rw.Start()
	po("Chaser started: %p for conn from '%s'\n\n", s, s.rw.conn.RemoteAddr())
	po("\n\n for Chaser %p we have rw=%p   with reader = %p  writer = %p  home = %p\n\n", s, s.rw, s.rw.r, s.rw.w, s.home)
}

// Stops without reporting anything on the
// notifyDone channel passed to NewChaser().
func (s *Chaser) StopWithoutReporting() {
	s.skipNotify = true
	s.Stop()
}

// Stop the Chaser.
func (s *Chaser) Stop() {
	po("%p Chaser stopping.\n", s)

	s.RequestStop()

	s.rw.Stop()

	<-s.alphaDone
	<-s.betaDone
	<-s.monitorDone
	s.home.Stop()

	po("%p chaser all done.\n", s)
	close(s.Done)
}

// monitor rw for shutdown
func (s *Chaser) startMonitor() {
	go func() {
		defer func() {
			// tell the server that this key/session is closing down.
			// important so that the long-polling routines can shut down.
			// now prefer: let the rev server response to Stop() and
			// don't open any more network connections when we are
			// shutting down.
			// s.DoRequestResponse([]byte{}, "closekey")

			close(s.monitorDone)
			po("%p monitor done.", s)
		}()

		for {
			select {

			case <-s.reqStop:
				po("%p monitor got reqStop.", s)
				s.rw.StopWithoutNotify()
				return
			case <-s.rwReaderDone:
				po("%p monitor got rwReaderDone", s)
				s.rw.StopWithoutNotify()
				s.RequestStop()
				return
			case <-s.rwWriterDone:
				po("%p monitor got rwWriterDone\n", s)
				s.rw.StopWithoutNotify()
				s.RequestStop()
				return
			}
		}
	}()
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *Chaser) RequestStop() bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	select {
	case <-s.reqStop:
		return false
	default:
		close(s.reqStop)
		return true
	}
}

func (s *Chaser) startAlpha() {
	go func() {
		po("%p alpha at top of startAlpha", s)

		// so we don't notify twice, make alpha alone responsible
		// for reporting on s.notifyDone.
		defer func() {
			if !s.skipNotify {
				select {
				case s.notifyDone <- s:
					po("%p Alpha shutting down, after s.notifyDone <- s finished.", s)
				case <-time.After(1000 * time.Millisecond):
					// needed in case nobody is listening for us anymore
				}
			}
			// cancel any outstanding http req, and close idle connections
			s.httpClient.CancelAllReq()
			s.httpClient.CloseIdleConnections()
			close(s.alphaDone)
			po("%p Alpha done.", s)
		}()

		var work []byte
		var goNow bool
		for {
			work = []byte{}

			select {
			case goNow = <-s.home.shouldAlphaGoNow:
			case <-s.reqStop:
				po("%p Alpha got s.reqStop", s)
				return
			}
			if !goNow {

				// only I am home, so wait for an event.
				select {
				case work = <-s.incoming:
					po("%p alpha got work on s.incoming: '%s'.\n", s, string(work))

				// launch with the data in work
				case <-s.reqStop:
					po("%p Alpha got s.reqStop", s)
					return
				case <-s.betaDone:
					po("%p Alpha got s.betaDone", s)
					return
				case <-s.home.tellAlphaToGo:
					po("%p alpha got s.home.tellAlphaToGo.\n", s)

					// we can launch without data, but
					// make sure there isn't some data waiting,
					// check again just so the random
					// nature of select won't hurt data deliver rates.
					select {
					case work = <-s.incoming:
					default:
						// don't block on it through, go ahead with empty data
						// if we don't have any.
					}
				case <-s.inactiveTimer.C:
					po("%p alpha got <-s.inactiveTimer.C, after %v: returning/shutting down.", s, s.shutdownInactiveDur)
					return
				}
			}

			if len(work) > 0 {
				// actual bytes to send!
				s.ResetActiveTimer()
			}
			// else must send out anyway, since we may be just long-polling for
			// keep-alive and server sending purposes.

			// send request to server
			s.home.alphaDepartsHome <- true

			// ================================
			// request-response cycle here
			// ================================

			po("%p alpha about to call DoRequestResponse('%s')", s, string(work))
			replyBytes, recvSerial, err := s.DoRequestResponse(work, "")
			if err != nil {
				po("%p alpha aborting on error from DoRequestResponse: '%s'", s, err)
				return
			}
			po("%p alpha DoRequestResponse done work:'%s' -> '%s'. with recvSerial: %d\n", s, string(work), string(replyBytes), recvSerial)

			// if Beta is here, tell him to head out.
			s.home.alphaArrivesHome <- true

			if len(replyBytes) > 0 {
				s.ResetActiveTimer()

				by := bytes.NewBuffer(replyBytes)

				tryMe := recvSerial + 1
				for {
					if !s.addIfPresent(&tryMe, by) {
						break
					}
				}
				sendMe := by.Bytes()

				// deliver any response data (body) to our client
				select {
				case s.repliesHere <- sendMe:
					po("%p Alpha sent to repliesHere: '%s'", s, string(sendMe))
				case <-s.reqStop:
					po("%p Alpha got s.reqStop", s)
					return
				}
			}
		}
	}()
}

// Beta is responsible for the second http
// connection.
func (s *Chaser) startBeta() {
	go func() {
		po("%p beta at top of startBeta", s)
		defer func() {
			close(s.betaDone)
			po("%p Beta done.", s)
		}()
		var work []byte
		var goNow bool
		for {
			work = []byte{}

			select {
			case goNow = <-s.home.shouldBetaGoNow:
				po("%p Beta got goNow = %v", s, goNow)
			case <-s.reqStop:
				po("%p Beta got s.reqStop", s)
				return
			}

			if !goNow {

				select {

				case work = <-s.incoming:
					po("%p beta got work on s.incoming '%s'.\n", s, string(work))
					// launch with the data in work
				case <-s.reqStop:
					po("%p Beta got s.reqStop", s)
					return
				case <-s.alphaDone:
					return
				case <-s.home.tellBetaToGo:
					po("%p beta got s.home.tellBetaToGo.\n", s)

					// we can launch without data, but
					// make sure there isn't some data waiting,
					// check again just so the random
					// nature of select won't hurt data deliver rates.
					select {
					case work = <-s.incoming:
					default:
						// don't block on it through, go ahead with empty data
						// if we don't have any.
					}
				}
			}

			if len(work) > 0 {
				s.ResetActiveTimer()
			}

			// send request to server
			s.home.betaDepartsHome <- true

			// ================================
			// request-response cycle here
			// ================================

			replyBytes, recvSerial, err := s.DoRequestResponse(work, "")
			if err != nil {
				po("%p beta aborting on error from DoRequestResponse: '%s'", s, err)
				return
			}
			po("%p beta DoRequestResponse done; recvSerial = %d.\n", s, recvSerial)

			// if Alpha is here, tell him to head out.
			s.home.betaArrivesHome <- true

			if len(replyBytes) > 0 {
				s.ResetActiveTimer()

				by := bytes.NewBuffer(replyBytes)

				tryMe := recvSerial + 1
				for {
					if !s.addIfPresent(&tryMe, by) {
						break
					}
				}
				sendMe := by.Bytes()

				// deliver any response data (body) to our client, but only
				// bother if len(replyBytes) > 0, as checked above.
				select {
				case s.repliesHere <- sendMe:
					po("%p Beta sent to repliesHere: '%s'", s, string(sendMe))
				case <-s.reqStop:
					//po("%p Beta got s.reqStop", s)
					return
				}
			}
		}
	}()
}

// sleep for some random interval to simulate time to server and back.
func rsleep() {
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
}

type who int

const Alpha who = 1
const Beta who = 2
const Both who = 3

type ClientHome struct {
	reqStop chan bool
	Done    chan bool

	IsAlphaHome chan bool
	IsBetaHome  chan bool

	alphaArrivesHome chan bool
	betaArrivesHome  chan bool

	alphaDepartsHome chan bool
	betaDepartsHome  chan bool

	//	alphaShutdown chan bool
	//	betaShutdown  chan bool

	// for measuring latency under simulation
	localWishesToSend chan bool

	shouldAlphaGoNow chan bool
	shouldBetaGoNow  chan bool

	tellBetaToGo  chan bool
	tellAlphaToGo chan bool

	alphaHome bool
	betaHome  bool

	//	alphaShut bool
	//	betaShut  bool

	shouldAlphaGoCached bool
	shouldBetaGoCached  bool

	lastHome who

	localReqArrTm  int64
	latencyHistory []int64
	mut            sync.Mutex
}

func NewClientHome() *ClientHome {

	s := &ClientHome{
		reqStop: make(chan bool),
		Done:    make(chan bool),

		IsAlphaHome: make(chan bool),
		IsBetaHome:  make(chan bool),

		alphaArrivesHome: make(chan bool),
		betaArrivesHome:  make(chan bool),

		alphaDepartsHome: make(chan bool),
		betaDepartsHome:  make(chan bool),

		//alphaShutdown: make(chan bool),
		//betaShutdown:  make(chan bool),

		shouldAlphaGoNow: make(chan bool),
		shouldBetaGoNow:  make(chan bool),

		tellBetaToGo:  make(chan bool),
		tellAlphaToGo: make(chan bool),

		localWishesToSend: make(chan bool),

		shouldAlphaGoCached: true,
		shouldBetaGoCached:  false,

		alphaHome: true,
		betaHome:  true,
	}
	return s
}

func (s *ClientHome) Stop() {
	po("%p client home stop requested", s)
	s.RequestStop()
	<-s.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *ClientHome) RequestStop() bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	select {
	case <-s.reqStop:
		return false
	default:
		close(s.reqStop)
		return true
	}
}

func (s *ClientHome) String() string {
	return fmt.Sprintf("home:{alphaHome: %v, betaHome: %v} / ptr=%p", s.alphaHome, s.betaHome, s)
}

func (s *ClientHome) Start() {
	po("%p home starting.", s)

	go func() {
		defer func() {
			po("%p home done.", s)
		}()
		for {
			select {

			case s.IsAlphaHome <- s.alphaHome:
			case s.IsBetaHome <- s.betaHome:

			case <-s.alphaArrivesHome:
				// for latency study
				if s.localReqArrTm > 0 {
					lat := time.Now().UnixNano() - s.localReqArrTm
					s.latencyHistory = append(s.latencyHistory, lat)
					fmt.Printf("\n latency: %v\n", lat)
					s.localReqArrTm = 0
				}

				s.alphaHome = true

				VPrintf("++++  home received alphaArrivesHome. state of Home= '%s'\n", s)

				s.lastHome = Alpha
				if s.betaHome {
					select {
					case s.tellBetaToGo <- true:
					default:
					}
				}
				s.update()
				VPrintf("++++  end of alphaArrivesHome. state of Home= '%s'\n", s)

			case <-s.betaArrivesHome:
				// for latency study
				if s.localReqArrTm > 0 {
					lat := time.Now().UnixNano() - s.localReqArrTm
					s.latencyHistory = append(s.latencyHistory, lat)
					fmt.Printf("\n latency: %v\n", lat)
					s.localReqArrTm = 0
				}
				s.betaHome = true
				VPrintf("++++  home received betaArrivesHome. state of Home= '%s'\n", s)

				s.lastHome = Beta
				if s.alphaHome {
					select {
					case s.tellAlphaToGo <- true:
					default:
					}
				}
				s.update()
				VPrintf("++++  end of betaArrivesHome. state of Home= '%s'\n", s)

			case <-s.alphaDepartsHome:
				s.alphaHome = false
				s.update()
				VPrintf("----  home received alphaDepartsHome. state of Home= '%s'\n", s)

			case <-s.betaDepartsHome:
				s.betaHome = false
				s.update()
				VPrintf("----  home received betaDepartsHome. state of Home= '%s'\n", s)

			case s.shouldAlphaGoNow <- s.shouldAlphaGoCached:

			case s.shouldBetaGoNow <- s.shouldBetaGoCached:

			case <-s.reqStop:
				po("%p home got s.reqStop", s)
				close(s.Done)
				return

			case <-s.localWishesToSend:
				// for measuring latency under simulation
				s.localReqArrTm = time.Now().UnixNano()
				if s.numHome() > 0 {
					s.latencyHistory = append(s.latencyHistory, 0)
					fmt.Printf("\n latency: %v\n", time.Duration(0))
					s.localReqArrTm = 0 // send done instantly, reset to indicate no pending send.
				}

				//			case <-s.alphaShutdown:
				//				s.alphaShut = true
				//			case <-s.betaShutdown:
				//				s.betaShut = true
			}
		}
	}()
}

func (s *ClientHome) shouldAlphaGo() (res bool) {
	if s.numHome() == 2 {
		return true
	}
	return false
}

func (s *ClientHome) shouldBetaGo() (res bool) {
	// in case of tie, arbitrarily alpha goes first.
	return false
}

func (s *ClientHome) numHome() (res int) {
	if s.alphaHome && s.betaHome {
		return 2
	}
	if s.alphaHome || s.betaHome {
		return 1
	}
	return 0
}

func (s *ClientHome) update() {
	s.shouldAlphaGoCached = s.shouldAlphaGo()
	s.shouldBetaGoCached = s.shouldBetaGo()

}

func (s *Chaser) getNextSendSerNum() int64 {
	s.mut.Lock()
	defer s.mut.Unlock()
	v := s.nextSendSerialNumber
	s.nextSendSerialNumber++
	return v
}

// DoRequestResponse():
// issue a post to the urlPath (omit the leading slash! deault to ""),
// submitting 'work', returning 'back' and any error.
// the urlPath should normally be "", but can be "closekey" to tell
// the server to return any outstanding request for key immediately and
// to shutdown the downstream connection to target server.
// We don't want to actually close the actual physical net.Conn
// because other keys/clients can re-use it for other/on-going work.
// Indeed it might be in use right now for another key's packets.
//
func (s *Chaser) DoRequestResponse(work []byte, urlPath string) (back []byte, recvSerial int64, err error) {

	//po("debug: DoRequestResponse called with dest: '%#v', key: '%s', and work: '%s'", s.dest, s.key, string(work))

	// only assign serial numbers to client payload, not to internal zero-byte
	// alpha/beta requests that are there just to give the server a reply medium.
	reqSer := int64(-1)
	if len(work) > 0 {
		reqSer = s.getNextSendSerNum()
	}

	// assemble key + work into request
	req := bytes.NewBuffer([]byte(s.key))

	serBy := SerialToBytes(reqSer)
	po("debug: serial = %d", reqSer)

	req.Write(serBy) // add seqnum after key

	req.Write(work) // add work after key + seqnum

	po("%p Chaser.DoRequestResponse(url='%s') just before Post of work = '%s'. s.cfg.ConnectTimeout = %v, s.cfg.TransportTimeout = %v. requestSerial = %d\n", s, urlPath, string(work), s.cfg.ConnectTimeout, s.cfg.TransportTimeout, reqSer)

	url := "http://" + s.dest.IpPort + "/" + urlPath

	//resp, err := http.Post(url, "application/octet-stream", req)
	//
	// preferred over http.Post() for its tunable timeouts:
	resp, err := s.httpClient.Post(url, "application/octet-stream", req)

	po("%p '%s' Chaser.DoRequestResponse(url='%s') just after Post.", s, string(s.key[:5]), urlPath)

	defer func() {
		if resp != nil && resp.Body != nil {
			ioutil.ReadAll(resp.Body) // read anything leftover, so connection can be reused.
			resp.Body.Close()
		}
	}()

	if err != nil && err != io.EOF {
		po("%p '%s' Chaser.DoRequestResponse(url='%s') : '%s'", err.Error(), s, string(s.key[:5]))
		return []byte{}, -1, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	panicOn(err)

	recvSerial = -1 // default for empty bytes in body
	if len(body) >= SerialLen {
		// if there are any bytes, then the replySerial number will be the last 8
		serStart := len(body) - SerialLen
		recvSerial = BytesToSerial(body[serStart:])
		body = body[:serStart]
	}

	po("%p chaser '%s' / '%s', resp.Body = '%s'. With recvSerial = %d\n", s, s.key[:5], s.rw.name, string(body), recvSerial)

	if recvSerial >= 0 {
		// adjust s.lastRecvSerialNumberSeen and s.misorderedReplies under lock
		s.mut.Lock()
		defer s.mut.Unlock()

		if recvSerial != s.lastRecvSerialNumberSeen+1 {

			s.misorderedReplies[recvSerial] = &SerResp{
				response:       back,
				responseSerial: recvSerial,
				tm:             time.Now(),
			}
			// wait to send upstream: indicate this by giving back 0 length.
			back = back[:0]
		} else {
			s.lastRecvSerialNumberSeen++
		}
	}

	return body, recvSerial, err
}

// Helper for startAlpha/startBeta;
// returns true iff we found and deleted tryMe from the s.misorderedReplies map.
//  Along with the delete we write the contents of the found.response to 'by'.
func (s *Chaser) addIfPresent(tryMe *int64, by *bytes.Buffer) bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	ooo, ok := s.misorderedReplies[*tryMe]

	if !ok {
		return false
	}
	po("ab reply misordering being corrected, reply sn: %d, data: '%s'",
		*tryMe, string(ooo.response))
	by.Write(ooo.response)
	delete(s.misorderedReplies, *tryMe)
	s.lastRecvSerialNumberSeen = *tryMe
	(*tryMe)++

	return true
}

type SerResp struct {
	response       []byte
	responseSerial int64 // order the sends with content by serial number
	tm             time.Time
}

/// logging

func (r *Chaser) NoteTmRecv() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastRecv = append(r.tmLastRecv, time.Now())
}

func (r *Chaser) NoteTmSent() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastSend = append(r.tmLastSend, time.Now())
}

func (r *Chaser) ShowTmHistory() {
	r.mut.Lock()
	defer r.mut.Unlock()
	po("Chaser.ShowTmHistory() called.")

	nr := len(r.tmLastRecv)
	ns := len(r.tmLastSend)
	min := nr
	if ns < min {
		min = ns
	}

	fmt.Printf("%s history: ns=%d.  nr=%d.  min=%d.\n", r.name, ns, nr, min)

	for i := 0; i < ns; i++ {
		fmt.Printf("%s history of Send from AB to LP '%v'  \n",
			r.name,
			r.tmLastSend[i])
	}

	for i := 0; i < nr; i++ {
		fmt.Printf("%s history of Recv from LP at AB '%v'  \n",
			r.name,
			r.tmLastRecv[i])
	}

	for i := 0; i < min; i++ {
		fmt.Printf("%s history: elap: '%s'    Send '%v'   Recv '%v'    \n",
			r.name,
			r.tmLastSend[i].Sub(r.tmLastRecv[i]),
			r.tmLastSend[i],
			r.tmLastRecv[i])
	}

	for i := 0; i < min-1; i++ {
		fmt.Printf("%s history: send-to-send elap: '%s'\n", r.name, r.tmLastSend[i+1].Sub(r.tmLastSend[i]))
	}
	for i := 0; i < min-1; i++ {
		fmt.Printf("%s history: recv-to-recv elap: '%s'\n", r.name, r.tmLastRecv[i+1].Sub(r.tmLastRecv[i]))
	}

}
