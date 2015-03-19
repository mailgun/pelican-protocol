package pelicantun

import (
	"fmt"
	"testing"
)

func TestFullRoundtripAllCanShutdown009(t *testing.T) {

	web, rev, fwd, err := StartTestSystemWithPing()
	panicOn(err)
	defer web.Stop()
	defer rev.Stop()
	defer fwd.Stop()

	fmt.Printf("\n done with Test Full Roundtrip All Can Shutdown 009()\n")
}
