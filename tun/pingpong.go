package pelicantun

import (
	"math/rand"
	"sync"
	"time"
)

func example_main() {
	c := NewChaser()
	c.Start()

	for i := 0; i < 100; i++ {
		c.incoming <- i
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

	mutex      sync.Mutex
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

// alpha and beta are a pair of room-mates
// who hate to be home together.
// If alpha arrives home and beta is present,
// alpha kicks out beta and beta goes on a data
// retrieval mission. When beta gets back if
// alpha is home, alpha is forced to go himself
// on a data retrieval mission. If they both
// find themselves at home at once, then the
// tie is arbitrarily broken like this: alpha
// goes out (to server to fetch any data server has).
//
// In this way we implement the ping-pong of
// long-polling. Within the constraints of only
// having two http connections open, each party
// can send whenever they so desire, with as low
// latency as we can muster within the constraints
// of only using two http connections.
//
func (s *Chaser) StartAlpha() {
	go func() {
		var work int
		for {
			work = 0

			goNow := <-s.home.shouldAlphaGoNow
			if !goNow {

				// only I am home, so wait for an event.
				select {
				case work = <-s.incoming:
				// launch with the data in work
				case <-s.ReqStop:
					close(s.alphaDone)
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

			// send request to server
			po("Alpha got work %d, and now is away delivering that work\n", work)
			rsleep()

			po("Alpha back\n")

			// if Beta is here, tell him to head out.
			s.home.alphaArrivesHome <- true

			// deliver any response data to our client
			po("Alpha back and delivering any response data to client\n")
			rsleep()

		}
	}()
}

// Beta is responsible for the second http
// connection.
func (s *Chaser) StartBeta() {
	go func() {
		var work int
		for {
			work = 0

			goNow := <-s.home.shouldBetaGoNow
			if !goNow {

				select {
				case work = <-s.incoming:
					// launch with the data in work
				case <-s.ReqStop:
					close(s.betaDone)
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

			// send request to server

			po("Beta got work %d, and now is away delivering that work\n", work)
			rsleep()

			po("Beta back.\n")
			// if Alpha is here, tell him to head out.
			s.home.betaArrivesHome <- true

			// deliver any response data to our client
			po("Beta back and delivering any response data to client\n")
			rsleep()
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
	alphaArrivesHome chan bool
	betaArrivesHome  chan bool

	alphaDepartsHome chan bool
	betaDepartsHome  chan bool

	shouldAlphaGoNow chan bool
	shouldBetaGoNow  chan bool

	alphaHome bool
	betaHome  bool
	lastHome  who

	ReqStop chan bool
	Done    chan bool

	tellBetaToGo  chan bool
	tellAlphaToGo chan bool
}

func NewHome() *Home {
	s := &Home{
		alphaArrivesHome: make(chan bool),
		betaArrivesHome:  make(chan bool),

		alphaDepartsHome: make(chan bool),
		betaDepartsHome:  make(chan bool),

		shouldAlphaGoNow: make(chan bool),
		shouldBetaGoNow:  make(chan bool),
		alphaHome:        true,
		betaHome:         true,
		ReqStop:          make(chan bool),
		Done:             make(chan bool),

		tellBetaToGo:  make(chan bool),
		tellAlphaToGo: make(chan bool),
	}
	return s
}

func (s *Home) Stop() {
	close(s.ReqStop)
	<-s.Done
}

func (s *Home) Start() {
	go func() {
		for {
			select {
			case <-s.alphaArrivesHome:
				s.alphaHome = true
				s.lastHome = Alpha
				if s.betaHome {
					select {
					case s.tellBetaToGo <- true:
						s.betaHome = false
					default:
					}
				}

			case <-s.betaArrivesHome:
				s.betaHome = true
				s.lastHome = Beta
				if s.alphaHome {
					select {
					case s.tellAlphaToGo <- true:
						s.alphaHome = false
					default:
					}
				}

			case <-s.alphaDepartsHome:
				s.alphaHome = false
			case <-s.betaDepartsHome:
				s.betaHome = false

			case s.shouldAlphaGoNow <- s.shouldAlphaGo():
				if s.shouldAlphaGo() {
					s.alphaHome = false
				}
			case s.shouldBetaGoNow <- s.shouldBetaGo():
				if s.shouldBetaGo() {
					s.betaHome = false
				}
			case <-s.ReqStop:
				close(s.Done)
				return
			}
		}
	}()
}

// PRE: assumes alpha is home
func (s *Home) shouldAlphaGo() bool {
	if s.numHome() == 2 {
		return true
	}
	return false
}

// PRE: assumes beta is home
func (s *Home) shouldBetaGo() bool {
	if s.numHome() == 2 {
		// in case of tie, arbitrarily alpha goes first.
		return false
	}
	return true
}

func (s *Home) numHome() int {
	if s.alphaHome && s.betaHome {
		return 2
	}
	if s.alphaHome || s.betaHome {
		return 1
	}
	return 0
}
