package pelicantun

import (
	"fmt"
	"math/rand"
	"time"
)

var globalHome *Home

func example_main() {
	c := NewChaser()
	c.Start()
	globalHome = c.home

	for i := 1; i < 100; i++ {
		c.incoming <- i
		rsleep()
		rsleep()
		rsleep()
		rsleep()
	}

}

func NewChaser() *Chaser {
	s := &Chaser{
		ReqStop: make(chan bool),
		Done:    make(chan bool),

		alphaDone: make(chan bool),
		betaDone:  make(chan bool),

		incoming:    make(chan int),
		alphaIsHome: true,
		betaIsHome:  true,
		closedChan:  make(chan bool),
		home:        NewHome(),
	}

	// always closed
	close(s.closedChan)

	return s
}

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
// has its own goroutine. The StartAlpha() and
// StartBeta() methods each start their own
// goroutines respectively, and the three communicate
// over the channels held in Chaser and Home.
//
type Chaser struct {
	ReqStop chan bool
	Done    chan bool

	incoming    chan int
	alphaIsHome bool
	betaIsHome  bool

	alphaArrivesHome chan bool
	betaArrivesHome  chan bool

	alphaDone chan bool
	betaDone  chan bool

	closedChan chan bool
	home       *Home
}

func (s *Chaser) Start() {
	s.home.Start()
	s.StartAlpha()
	s.StartBeta()
}

func (s *Chaser) Stop() {
	select {
	case <-s.ReqStop:
	default:
		close(s.ReqStop)
	}
	<-s.alphaDone
	<-s.betaDone
	s.home.Stop()
	close(s.Done)
}

func (s *Chaser) StartAlpha() {
	go func() {
		defer func() { close(s.alphaDone) }()
		var work int
		var goNow bool
		for {
			work = 0

			select {
			case goNow = <-s.home.shouldAlphaGoNow:
			case <-s.ReqStop:
				return
			}
			if !goNow {

				// only I am home, so wait for an event.
				select {
				case work = <-s.incoming:
				// launch with the data in work
				case <-s.ReqStop:
					return
				case <-s.home.tellAlphaToGo:
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
			if work > 0 {
				// quiet compiler
			}

			// send request to server
			s.home.alphaDepartsHome <- true
			//rsleep()

			// if Beta is here, tell him to head out.
			s.home.alphaArrivesHome <- true

			// deliver any response data to our client
			//rsleep()

		}
	}()
}

// Beta is responsible for the second http
// connection.
func (s *Chaser) StartBeta() {
	go func() {
		defer func() { close(s.betaDone) }()
		var work int
		var goNow bool
		for {
			work = 0

			select {
			case goNow = <-s.home.shouldBetaGoNow:
			case <-s.ReqStop:
				return
			}

			if !goNow {

				select {
				case work = <-s.incoming:
					// launch with the data in work
				case <-s.ReqStop:
					return
				case <-s.home.tellBetaToGo:
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
			if work > 0 {
				// quiet compiler
			}

			// send request to server
			s.home.betaDepartsHome <- true
			//rsleep()

			// if Alpha is here, tell him to head out.
			s.home.betaArrivesHome <- true

			// deliver any response data to our client
			//rsleep()
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
	ReqStop chan bool
	Done    chan bool

	IsAlphaHome chan bool
	IsBetaHome  chan bool

	alphaArrivesHome chan bool
	betaArrivesHome  chan bool

	alphaDepartsHome chan bool
	betaDepartsHome  chan bool

	// for measuring latency under simulation
	localWishesToSend chan bool

	shouldAlphaGoNow chan bool
	shouldBetaGoNow  chan bool

	tellBetaToGo  chan bool
	tellAlphaToGo chan bool

	alphaHome bool
	betaHome  bool

	shouldAlphaGoCached bool
	shouldBetaGoCached  bool

	lastHome who

	localReqArrTm  int64
	latencyHistory []int64
}

func NewHome() *Home {

	s := &Home{
		ReqStop: make(chan bool),
		Done:    make(chan bool),

		IsAlphaHome: make(chan bool),
		IsBetaHome:  make(chan bool),

		alphaArrivesHome: make(chan bool),
		betaArrivesHome:  make(chan bool),

		alphaDepartsHome: make(chan bool),
		betaDepartsHome:  make(chan bool),

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
	close(s.ReqStop)
	<-s.Done
}

func (s *Home) String() string {
	return fmt.Sprintf("home:{alphaHome: %v, betaHome: %v}", s.alphaHome, s.betaHome)
}

func (s *Home) Start() {
	go func() {
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

				//VPrintf("++++  home received alphaArrivesHome. state of Home= '%s'\n", s)

				s.lastHome = Alpha
				if s.betaHome {
					select {
					case s.tellBetaToGo <- true:
					default:
					}
				}
				s.update()
				//VPrintf("++++  end of alphaArrivesHome. state of Home= '%s'\n", s)

			case <-s.betaArrivesHome:
				// for latency study
				if s.localReqArrTm > 0 {
					lat := time.Now().UnixNano() - s.localReqArrTm
					s.latencyHistory = append(s.latencyHistory, lat)
					fmt.Printf("\n latency: %v\n", lat)
					s.localReqArrTm = 0
				}
				s.betaHome = true
				//VPrintf("++++  home received betaArrivesHome. state of Home= '%s'\n", s)

				s.lastHome = Beta
				if s.alphaHome {
					select {
					case s.tellAlphaToGo <- true:
					default:
					}
				}
				s.update()
				//VPrintf("++++  end of betaArrivesHome. state of Home= '%s'\n", s)

			case <-s.alphaDepartsHome:
				s.alphaHome = false
				s.update()
				//VPrintf("----  home received alphaDepartsHome. state of Home= '%s'\n", s)

			case <-s.betaDepartsHome:
				s.betaHome = false
				s.update()
				//VPrintf("----  home received betaDepartsHome. state of Home= '%s'\n", s)

			case s.shouldAlphaGoNow <- s.shouldAlphaGoCached:

			case s.shouldBetaGoNow <- s.shouldBetaGoCached:

			case <-s.ReqStop:
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
