// Package poll is an opinionated library for polling in the Signadot CLI.
package poll

import (
	"fmt"
	"time"
)

const (
	pollDelay = 1 * time.Second
)

// Until polls until the given function returns true, or the timeout expires.
func Until(timeout time.Duration, fn func() bool) error {
	start := time.Now()

	for {
		if time.Since(start) >= timeout {
			return fmt.Errorf("timed out after %v", timeout)
		}

		if fn() {
			return nil
		}

		time.Sleep(pollDelay)
	}
}
