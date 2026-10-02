package poll

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Stop racing with run closing done on a fatal error must not panic
// ("close of closed channel").
func TestReadinessStopRacesFatal(t *testing.T) {
	for i := 0; i < 200000; i++ {
		r := NewPoll().Readiness(context.Background(), time.Hour, func() (bool, error, error) {
			return false, nil, errors.New("fatal")
		})
		r.Stop()
		if err := r.Fatal(); err == nil {
			t.Fatal("expected fatal error")
		}
	}
}
