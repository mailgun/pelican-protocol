package main

import (
	"fmt"
	"time"
)

func (r *LittlePoll) NoteTmRecv() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastRecv = append(r.tmLastRecv, time.Now())
}
func (r *LittlePoll) NoteTmSent() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastSend = append(r.tmLastSend, time.Now())
}

func (r *Chaser) NoteTmRecv() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastRecv = append(r.tmLastRecv, time.Now())
}
func (r *Chaser) NoteTmSent() {
	r.mut.Lock()
	defer r.mut.Unlock()
	r.tmLastSend = append(r.tmLastSend, time.Now())
}

func (r *Chaser) ShowTmHistory() {
	r.mut.Lock()
	defer r.mut.Unlock()
	nr := len(r.tmLastRecv)
	ns := len(r.tmLastSend)
	min := nr
	if ns < min {
		min = ns
	}
	for i := 0; i < min; i++ {
		fmt.Printf("Chaser history: elap: '%s'    Send '%v'   Recv '%v'  \n", r.tmLastRecv[i].Sub(r.tmLastSend[i]), r.tmLastSend[i], r.tmLastRecv[i])

	}
	for i := 0; i < min-1; i++ {
		fmt.Printf("Chaser history: send-to-send elap: '%s'\n", r.tmLastSend[i+1].Sub(r.tmLastSend[i]))
	}
	for i := 0; i < min-1; i++ {
		fmt.Printf("Chaser history: recv-to-recv elap: '%s'\n", r.tmLastRecv[i+1].Sub(r.tmLastRecv[i]))
	}

}

func (r *LittlePoll) ShowTmHistory() {
	r.mut.Lock()
	defer r.mut.Unlock()
	nr := len(r.tmLastRecv)
	ns := len(r.tmLastSend)
	min := nr
	if ns < min {
		min = ns
	}
	for i := 0; i < min; i++ {
		fmt.Printf("LittlePoll history: elap: '%s'    Recv '%v'   Send '%v'  \n", r.tmLastSend[i].Sub(r.tmLastRecv[i]), r.tmLastRecv[i], r.tmLastSend[i])
	}

	for i := 0; i < min-1; i++ {
		fmt.Printf("LittlePoll history: send-to-send elap: '%s'\n", r.tmLastSend[i+1].Sub(r.tmLastSend[i]))
	}
	for i := 0; i < min-1; i++ {
		fmt.Printf("LittlePoll history: recv-to-recv elap: '%s'\n", r.tmLastRecv[i+1].Sub(r.tmLastRecv[i]))
	}

}
