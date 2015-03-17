package pelicantun

// ideas from gist: https://gist.github.com/seantalts/11266762

import (
	"fmt"
	"net/http"
	"time"
)

type TimeoutClient struct {
	http.Client
	trans *TimeoutTransport
}

func NewTimeoutClient(roundTo time.Duration) *TimeoutClient {
	trans := NewTimeoutTransport(roundTo)
	return &TimeoutClient{
		http.Client{
			Transport: trans,
		},
		trans,
	}
}

func NewTimeoutTransport(roundTo time.Duration) *TimeoutTransport {
	return &TimeoutTransport{RoundTripTimeout: roundTo}
}

type TimeoutTransport struct {
	http.Transport
	RoundTripTimeout time.Duration
}

type respAndErr struct {
	resp *http.Response
	err  error
}

type netTimeoutError struct {
	error
}

func (ne netTimeoutError) Timeout() bool { return true }

// If you don't set RoundTrip on TimeoutTransport, this will always timeout at 0
func (t *TimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	timeout := time.After(t.RoundTripTimeout)
	resp := make(chan respAndErr, 1)

	go func() {
		r, e := t.Transport.RoundTrip(req)
		resp <- respAndErr{
			resp: r,
			err:  e,
		}
	}()

	select {
	case <-timeout: // A round trip timeout has occurred.
		t.Transport.CancelRequest(req)
		return nil, netTimeoutError{
			error: fmt.Errorf("timed out after %s", t.RoundTripTimeout),
		}
	case r := <-resp: // Success!
		return r.resp, r.err
	}
}
