package pelicantun

import (
	"math/rand"
	"time"
)

// prototype of the ping-pong/apha-beta
// long polling implementation

func example_main() {
	c := &Chaser{
		ReqStop: make(chan bool),
		Done:    make(chan bool),

		alphaDone: make(chan bool),
		betaDone:  make(chan bool),

		incoming:    make(chan int),
		alphaIsHome: make(chan bool),
		betaIsHome:  make(chan bool),
	}

	c.Start()

	for i := 0; i < 100; i++ {
		c.incoming <- i
		rsleep()
	}

}

type Chaser struct {
	ReqStop chan bool
	Done    chan bool

	incoming    chan int
	alphaIsHome chan bool
	betaIsHome  chan bool

	alphaDone chan bool
	betaDone  chan bool
}

func (s *Chaser) Start() {
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
	close(s.Done)
}

// alpha and beta are an odd-couple of room-mates
// who hate to be home together.
// If alpha arrives home and beta is present,
// alpha kicks out beta and beta goes on a data
// retrieval mission. When beta gets back if
// alpha is home, alpha is forced to go himself
// on a data retrieval mission.
//
// In this way we implement the ping-pong of
// long-polling. Within the constraints of only
// having two http connections open, each party
// can send whenever they so desire.
//
func (s *Chaser) StartAlpha() {
	go func() {
		var work int
		for {
			work = 0

			select {
			// if both are home, it is random who
			// gets the work off of c.Incoming
			case work = <-s.incoming:
				// launch with the data in work
			case <-s.betaIsHome:
				// launch without data
			case <-s.ReqStop:
				close(s.alphaDone)
				return
			}

			// send request to server
			po("Alpha got work %d, and now is away delivering that work\n", work)
			rsleep()

			po("Alpha back\n")

			// if Beta is here, tell him to head out.
			select {
			case s.alphaIsHome <- true:
			default:
				// he's not home, no worries then.
			}

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
			select {
			// if both are home, it is random who
			// gets the work off of c.Incoming
			case work = <-s.incoming:
				// launch with the data in work
			case <-s.alphaIsHome:
				// launch without data
			case <-s.ReqStop:
				close(s.betaDone)
				return
			}

			// send request to server
			po("Beta got work %d, and now is away delivering that work\n", work)
			rsleep()

			po("Beta back.\n")
			// if Alpha is here, tell him to head out.
			select {
			case s.betaIsHome <- true:
			default:
				// he's not home, no worries then.
			}

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
