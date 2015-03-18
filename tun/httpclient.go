package pelicantun

import (
	"bytes"
	"fmt"
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
		Client: http.Client{
			Transport: trans,
		},
		trans: trans,
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

func (s *HttpClientWithTimeout) CancelAllReq() {
	s.trans.ReqCancel <- true
}

func (s *HttpClientWithTimeout) NumDials() int64 {
	return s.trans.NumDials
}
func (s *HttpClientWithTimeout) Post(url string, contentType string, body *bytes.Buffer) (*http.Response, error) {

	req, err := http.NewRequest("POST", url, body)

	if err != nil {
		return nil, fmt.Errorf("HttpClientWithTimeout::Post() NewRequest() to '%s' failed with error: '%s'", url, err)
	}

	req.Header.Add("Connection", "keep-alive")

	response, err := s.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HttpClientWithTimeout::Post() client.Do(req) failed with error: '%s'", err)
	}

	return response, err
}

func NewTimeoutTransport(roundTo time.Duration) *TimeoutTransport {
	s := &TimeoutTransport{
		RoundTripTimeout: roundTo,
		ReqCancel:        make(chan bool, 100),
	}

	s.Transport = http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			po("TimeoutTransport: dial to %s://%s", netw, addr)
			s.NumDials++
			return defaultDialerWithTimeout.Dial(netw, addr)
		},
		Proxy: http.ProxyFromEnvironment,
	}
	return s
}

type TimeoutTransport struct {
	http.Transport
	RoundTripTimeout time.Duration
	ReqCancel        chan bool
	NumDials         int64
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
	case <-timeout:
		t.Transport.CancelRequest(req)
		return nil, netTimeoutError{
			error: fmt.Errorf("timed out after %s", t.RoundTripTimeout),
		}
	case <-t.ReqCancel:
		t.Transport.CancelRequest(req)
		return nil, netTimeoutError{
			error: fmt.Errorf("ReqCancel: request cancelled at user behest"),
		}
	case r := <-resp: // response received
		return r.resp, r.err
	}
}
