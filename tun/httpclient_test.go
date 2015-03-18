package pelicantun

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHttpTimeout018(t *testing.T) {
	http.HandleFunc("/normal", func(w http.ResponseWriter, req *http.Request) {
		// Empirically, timeouts less than these seem to be flaky
		time.Sleep(100 * time.Millisecond)
		io.WriteString(w, "returning ok from /normal")
	})
	http.HandleFunc("/timeout", func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(250 * time.Millisecond)
		io.WriteString(w, "returning ok from /timeout")
	})
	ts := httptest.NewServer(http.DefaultServeMux)
	defer ts.Close()

	client := NewHttpClientWithTimeout(time.Millisecond * 200)

	addr := ts.URL

	SendTestRequest(t, client, "1st", addr, "normal")
	if client.NumDials() != 1 {
		t.Fatalf("Should only have 1 dial at this point.")
	}
	SendTestRequest(t, client, "2st", addr, "normal")
	if client.NumDials() != 1 {
		t.Fatalf("Should only have 1 dial at this point.")
	}
	SendTestRequest(t, client, "3st", addr, "timeout")
	if client.NumDials() != 1 {
		t.Fatalf("Should only have 1 dial at this point.")
	}
	SendTestRequest(t, client, "4st", addr, "normal")
	if client.NumDials() != 2 {
		t.Fatalf("Should have our 2nd dial.")
	}

	time.Sleep(time.Millisecond * 700)

	SendTestRequest(t, client, "5st", addr, "normal")
	if client.NumDials() != 2 {
		t.Fatalf("Should still only have 2 dials.")
	}
}

func SendTestRequest(t *testing.T, client *HttpClientWithTimeout, id, addr, path string) {
	req, err := http.NewRequest("GET", addr+"/"+path, nil)

	if err != nil {
		t.Fatalf("new request failed - %s", err)
	}

	req.Header.Add("Connection", "keep-alive")

	switch path {
	case "normal":
		//resp, err := client.Do(req)
		resp, err := client.Post(addr+"/"+path, "text/html", &bytes.Buffer{})
		if err != nil {
			t.Fatalf("%s request failed - %s", id, err)
		} else {
			result, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				t.Fatalf("%s response read failed - %s", id, err2)
			}
			resp.Body.Close()
			t.Logf("%s request - %s", id, result)
		}
	case "timeout":
		//_, err := client.Do(req)
		_, err := client.Post(addr+"/"+path, "text/html", &bytes.Buffer{})

		if err == nil {
			t.Fatalf("%s request not timeout", id)
		} else {
			t.Logf("%s request - %s", id, err)
		}
	}
}
