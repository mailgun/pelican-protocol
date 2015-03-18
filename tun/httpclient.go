package pelicantun

// ideas from gist: https://gist.github.com/seantalts/11266762

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type HttpClientWithTimeout struct {
	http.Client
	trans *TimeoutTransport
}

func NewHttpClientWithTimeout(roundTo time.Duration) *HttpClientWithTimeout {

	trans := NewTimeoutTransport(roundTo)

	return &HttpClientWithTimeout{
		http.Client{
			Transport: trans,
		},
		trans,
	}
}

var defaultDialerWithTimeout = &net.Dialer{Timeout: 1000 * time.Millisecond}

// could be a global, but put on http client to be clear what it affects.
func (s *HttpClientWithTimeout) SetConnectTimeout(duration time.Duration) {
	defaultDialerWithTimeout.Timeout = duration
}

func (s *HttpClientWithTimeout) CloseIdleConnections() {
	s.trans.CloseIdleConnections()
}

func (s *HttpClientWithTimeout) Post(url string, contentType string, body *bytes.Buffer) ([]byte, error) {

	req, err := http.NewRequest("POST", url, body)

	if err != nil {
		return []byte{}, fmt.Errorf("HttpClientWithTimeout::Post() NewRequest() to '%s' failed with error: '%s'", url, err)
	}

	req.Header.Add("Connection", "keep-alive")

	response, err := s.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("HttpClientWithTimeout::Post() client.Do(req) failed with error: '%s'", err)
	}

	resp, err2 := ioutil.ReadAll(response.Body)
	if err2 != nil {
		return []byte{}, fmt.Errorf("HttpClientWithTimeout::Post() ReadAll(resp.Body) failed: '%s'", err2)
	}
	response.Body.Close()

	return resp, nil
}

func NewTimeoutTransport(roundTo time.Duration) *TimeoutTransport {
	return &TimeoutTransport{
		http.Transport{
			Dial:  defaultDialerWithTimeout.Dial,
			Proxy: http.ProxyFromEnvironment,
		},
		roundTo,
	}
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
