package pelicantun

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestDroppedConnectionsCloseBothEnds022(t *testing.T) {

	srv := NewBcastServer(Addr{})
	srv.Start()
	defer srv.Stop()

	rev := NewReverseProxy(ReverseProxyConfig{Dest: srv.Listen})
	rev.Start()
	defer rev.Stop()

	// whichever end does the active close on the tcp socket is the end that ends up in TIME_WAIT.
	// Hence: on the fwd side, Chaser.rw that manages the connection upstream should never initiate active close.
	// And: on the rev side, the LongPoller

	cv.Convey("Given a reverse proxy or a forward proxy, if we are hit with a IsPortBound()/WaitUntilServerUp() client that opens and closes the connection immedidately, we should waste no resources after the IsPortBound() closes without sending any data. PelicanSocksProxy::Start() does this to itself.", t, func() {

	})
}
