// Package poll is an opinionated library for polling in the Signadot CLI.
package poll

import (
	"context"
	"fmt"
	"time"
)

const (
	pollDelay = 1 * time.Second
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

type UntilWithErrorFunc func(ctx context.Context) (bool, error)

// UntilWithError polls until the given function returns true, or the timeout
// expires. But also can return error so can be pass down to handle proper
// errors
func (p *Poll) UntilWithError(ctx context.Context, fn UntilWithErrorFunc) error {
	start := time.Now()

	for {
		if p.timeout > 0 && time.Since(start) >= p.timeout {
			return fmt.Errorf("timed out after %v", p.timeout)
		}

		done, err := fn(ctx)
		if done {
			return nil
		}
		if err != nil {
			return err
		}

		if p.resetOnLoop {
			start = time.Now()
		}

		select {
		case <-time.After(p.delay):
			continue
		case <-ctx.Done():
			return fmt.Errorf("poll canceled during wait: %w", ctx.Err())
		}
	}
}

type UntilFunc func(ctx context.Context) bool

// Until polls until the given function returns true, or the timeout expires.
func (p *Poll) Until(ctx context.Context, fn UntilFunc) error {
	return p.UntilWithError(ctx, func(ctx context.Context) (bool, error) {
		return fn(ctx), nil
	})
}
