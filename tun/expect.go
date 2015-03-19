package main

import (
	"fmt"
	"time"
)

// poll up to 'within' time, every 20 milliseconds, for the
// predicate 'toHold'. If it does not hold, panic. Else return true.
func PollExpecting(desc string, toHold func() bool, within time.Duration) bool {
	sleep := time.Millisecond * 20
	tries := int(1 + int64(within)/20e6)
	//fmt.Printf("tries = %d\n", tries)
	t0 := time.Now()
	for i := 0; i < tries; i++ {
		if toHold() {
			fmt.Printf("Expect verified that '%s' on try %d; after %v\n", desc, i, time.Since(t0))
			return true
		}
		time.Sleep(sleep)
	}
	panic(fmt.Errorf("did not hold: '%s', within %d tries of %v each (total ~ %v)", desc, tries, sleep, within))
	return false
}
