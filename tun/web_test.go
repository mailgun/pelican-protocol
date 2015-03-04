package pelicantun_test

import (
	"strings"
	"testing"

	. "github.com/mailgun/pelican-protocol/tun"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWebServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Web Server Suite")
}

var _ = Describe("Web Server Suite", func() {

	s := NewWebServer(WebServerConfig{})
	s.Start()

	Describe("NewWebServer functions", func() {
		Context("Start() should bring up a debug web-server", func() {
			It("and should bind an unbound port automatically, and be servring debug/pprof", func() {

				Expect(PortIsBound(s.Cfg.Addr)).To(Equal(true))

				by, err := FetchUrl("http://" + s.Cfg.Addr + "/debug/pprof")

				Expect(err == nil).To(Equal(true))
				//fmt.Printf("by:'%s'\n", string(by))
				Expect(strings.HasPrefix(string(by), `<html>
<head>
<title>/debug/pprof/</title>
</head>
/debug/pprof/<br>
<br>`)).To(Equal(true))
			})
			It("Stop() should halt the web server.", func() {
				s.Stop()
				Expect(PortIsBound(s.Cfg.Addr)).To(Equal(false))
			})
		})
	})
})
