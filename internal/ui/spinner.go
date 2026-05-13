package ui

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
)

type spinnerState struct {
	s       spinner.Model
	done    chan struct{}
	running bool
}

func newSpinner() *spinnerState {
	s := spinner.New(spinner.WithSpinner(spinner.Line))
	return &spinnerState{s: s, done: make(chan struct{})}
}

func (sp *spinnerState) Start(msg string) {
	sp.running = true
	go func() {
		for {
			select {
			case <-sp.done:
				return
			default:
				sp.s, _ = sp.s.Update(spinner.TickMsg{})
				fmt.Printf("\r%s %s", sp.s.View(), InfoStyle.Render(msg))
			}
		}
	}()
}

func (sp *spinnerState) Stop() {
	if sp.running {
		sp.done <- struct{}{}
		sp.running = false
		fmt.Printf("\r                    \r")
	}
}

func PrintSpinner(msg string) func() {
	sp := newSpinner()
	sp.Start(msg)
	return sp.Stop
}