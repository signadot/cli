// Package poll is an opinionated library for polling in the Signadot CLI.
package poll

import (
	"fmt"
	"time"
)

type PollingState int

const (
	pollDelay = 1 * time.Second

	KeepPolling PollingState = iota
	StopPolling
)

type Poll struct {
	delay       time.Duration
	timeout     time.Duration
	resetOnLoop bool
}

func NewPoll() *Poll {
	return &Poll{
		delay: pollDelay,
	}
}

func (p *Poll) WithTimeout(timeout time.Duration) *Poll {
	p.timeout = timeout
	return p
}

func (p *Poll) WithDelay(delay time.Duration) *Poll {
	p.delay = delay
	return p
}

func (p *Poll) WithResetOnLoop(resetOnLoop bool) *Poll {
	p.resetOnLoop = resetOnLoop
	return p
}

// UntilWithError polls until the given function returns true, or the timeout expires.
// But also can return error so can be pass down to handle proper errors
func (p *Poll) UntilWithError(fn func() (PollingState, error)) error {
	start := time.Now()

	for {
		if p.timeout > 0 && time.Since(start) >= p.timeout {
			return fmt.Errorf("timed out after %v", p.timeout)
		}

		state, err := fn()
		if state == StopPolling {
			return nil
		}

		if err != nil {
			return err
		}

		if p.resetOnLoop {
			start = time.Now()
		}

		time.Sleep(p.delay)
	}
}

// Until polls until the given function returns true, or the timeout expires.
func (p *Poll) Until(fn func() PollingState) error {
	return p.UntilWithError(func() (PollingState, error) {
		return fn(), nil
	})
}
