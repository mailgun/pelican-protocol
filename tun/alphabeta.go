package pelicantun

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
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
	home       *Home

	key  string
	dest Addr

	// this rw maintains the net.Conn to the upstream client
	rw           *RW
	rwReaderDone chan *NetConnReader
	rwWriterDone chan *NetConnWriter

	notifyDone chan *Chaser
	skipNotify bool

	mut sync.Mutex
	cfg ChaserConfig

	httpClient *http.Client
}

type ChaserConfig struct {
	ConnectTimeout   time.Duration
	TransportTimeout time.Duration
}

func DefaultChaserConfig() *ChaserConfig {
	return &ChaserConfig{
		ConnectTimeout:   2000 * time.Millisecond,
		TransportTimeout: 60 * time.Second,
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

func NewChaser(cfg ChaserConfig, conn net.Conn, bufsz int, key string, notifyDone chan *Chaser, dest Addr) *Chaser {

	if key == "" || len(key) != KeyLen {
		panic(fmt.Errorf("NewChaser() error: key '%s' was not of expected length %d. instead: %d", key, KeyLen, len(key)))
	}

	if dest.IpPort == "" {
		panic(fmt.Errorf("dest.IpPort was empty the string"))
	}
	if dest.Port == 0 {
		panic(fmt.Errorf("dest.Port was 0"))
	}

	rwReaderDone := make(chan *NetConnReader)
	rwWriterDone := make(chan *NetConnWriter)

	rw := NewRW(conn, bufsz, rwReaderDone, rwWriterDone)

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

		alphaIsHome: true,
		betaIsHome:  true,
		closedChan:  make(chan bool),
		home:        NewHome(),
		dest:        dest,
		key:         key,
		notifyDone:  notifyDone,
		cfg:         cfg,
		httpClient:  NewTimeoutClient(cfg.TransportTimeout),
	}

	po("\n\n Chaser %p gets NewRW() = %p with %p NetConnReader and %p NetConnWriter. For conn = %s[remote] -> %s[local]\n\n", s, rw, rw.r, rw.w, conn.RemoteAddr(), conn.LocalAddr())

	// always closed
	close(s.closedChan)

	return s
}

func (s *Chaser) Start() {
	s.home.Start()
	s.startMonitor()
	s.startAlpha()
	s.startBeta()
	s.rw.Start()
	fmt.Printf("\n\n Chaser started: %p for conn from '%s'\n\n", s, s.rw.conn.RemoteAddr())
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
	fmt.Printf("\n\n Chaser %p stopping.\n", s)

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
				//select {
				s.notifyDone <- s
				po("%p Alpha shutting down, after s.notifyDone <- s finished.", s)
				//case <-time.After(10 * time.Millisecond):
				//}
				po("%p Alpha done.", s)
			}
		}()

		defer func() { close(s.alphaDone) }()
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
				}
			}

			if len(work) == 0 {
				continue
			}

			// send request to server
			s.home.alphaDepartsHome <- true

			// ================================
			// request-response cycle here
			// ================================

			po("%p alpha about to call DoRequestResponse('%s')", s, string(work))
			replyBytes, err := s.DoRequestRespnose(work)
			if err != nil {
				po("%p alpha aborting on error from DoRequestResponse: '%s'", s, err)
				return
			}
			po("%p alpha DoRequestResponse done work:'%s' -> '%s'.\n", s, string(work), string(replyBytes))

			// if Beta is here, tell him to head out.
			s.home.alphaArrivesHome <- true

			// deliver any response data (body) to our client
			select {
			case s.repliesHere <- replyBytes:
			case <-s.reqStop:
				po("%p Alpha got s.reqStop", s)
				return
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

			if len(work) == 0 {
				continue
			}

			// send request to server
			s.home.betaDepartsHome <- true

			// ================================
			// request-response cycle here
			// ================================

			replyBytes, err := s.DoRequestRespnose(work)
			if err != nil {
				po("%p beta aborting on error from DoRequestResponse: '%s'", s, err)
				return
			}
			po("%p beta DoRequestResponse done.\n", s)

			// if Alpha is here, tell him to head out.
			s.home.betaArrivesHome <- true

			// deliver any response data (body) to our client
			select {
			case s.repliesHere <- replyBytes:
			case <-s.reqStop:
				po("%p Beta got s.reqStop", s)
				return
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

type Home struct {
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

func NewHome() *Home {

	s := &Home{
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

func (s *Home) Stop() {
	s.RequestStop()
	<-s.Done
}

// RequestStop makes sure we only close
// the s.reqStop channel once. Returns
// true iff we closed s.reqStop on this call.
func (s *Home) RequestStop() bool {
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

func (s *Home) String() string {
	return fmt.Sprintf("home:{alphaHome: %v, betaHome: %v} / ptr=%p", s.alphaHome, s.betaHome, s)
}

func (s *Home) Start() {
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

func (s *Home) shouldAlphaGo() (res bool) {
	if s.numHome() == 2 {
		return true
	}
	return false
}

func (s *Home) shouldBetaGo() (res bool) {
	// in case of tie, arbitrarily alpha goes first.
	return false
}

func (s *Home) numHome() (res int) {
	if s.alphaHome && s.betaHome {
		return 2
	}
	if s.alphaHome || s.betaHome {
		return 1
	}
	return 0
}

func (s *Home) update() {
	s.shouldAlphaGoCached = s.shouldAlphaGo()
	s.shouldBetaGoCached = s.shouldBetaGo()

}

func (s *Chaser) DoRequestRespnose(work []byte) (back []byte, err error) {

	//modeled after: func (reader *ConnReader) sendThenRecv(dest Addr, key string, buf *bytes.Buffer) error {
	// write buf to new http request, starting with key

	//po("\n\n debug: DoRequestRespnose called with dest: '%#v', key: '%s', and work: '%s'\n", s.dest, s.key, string(work))

	// assemble key + work into request
	req := bytes.NewBuffer([]byte(s.key))
	req.Write(work) // add work after key

	po("in DoRequestResponse just before Post of work = '%s'. s.cfg.ConnectTimeout = %v, s.cfg.TransportTimeout = %v\n", string(work), s.cfg.ConnectTimeout, s.cfg.TransportTimeout)

	// this stuff *sloooows* everything down: it messes with the web server shutdown times.
	/*
		goreq.SetConnectTimeout(s.cfg.ConnectTimeout)
		resp, err := goreq.Request{
			Method:      "POST",
			Uri:         "http://" + s.dest.IpPort + "/",
			ContentType: "application/octet-stream",
			Body:        req,
			Timeout:     s.cfg.TransportTimeout,
		}.Do()
	*/

	// also slows the shutdown, what the ??
	//resp, err := s.httpClient.Post(

	resp, err := http.Post(
		"http://"+s.dest.IpPort+"/",
		"application/octet-stream",
		req)

	po("in DoRequestResponse just after Post\n")

	defer func() {
		if resp != nil && resp.Body != nil {
			ioutil.ReadAll(resp.Body) // read anything leftover, so connection can be reused.
			resp.Body.Close()
		}
	}()

	if err != nil && err != io.EOF {
		log.Println(err.Error())
		//continue
		return []byte{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	panicOn(err)
	po("client: resp.Body = '%s'\n", string(body))

	return body, err
}
