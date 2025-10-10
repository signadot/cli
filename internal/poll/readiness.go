package poll

import (
	"context"
	"sync"
	"time"
)

type Readiness interface {
	Warn() error
	Fatal() error
	Ready() bool
	Stop()
	Stopped() <-chan struct{}
}

func (p *Poll) Readiness(ctx context.Context, interval time.Duration, fn func() (ready bool, warn, fatal error)) Readiness {
	res := &readiness{
		done:     make(chan struct{}),
		doneAck:  make(chan struct{}),
		warnC:    make(chan error, 100),
		fatalC:   make(chan error, 1),
		interval: interval,
		fn:       fn,
	}
	go res.run(ctx)
	return res
}

type readiness struct {
	sync.RWMutex
	warnC  chan error
	fatalC chan error
	ready  bool

	done, doneAck chan struct{}

	interval time.Duration
	fn       func() (ready bool, warn, fatal error)
}

func (r *readiness) run(ctx context.Context) {
	defer close(r.doneAck)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		ready, warn, fatal := r.fn()
		if fatal != nil {
			r.fatalC <- fatal
			select {
			case <-r.done:
				return
			default:
				close(r.done)
			}
			return
		}
		r.add(ready, warn)
		select {
		case <-r.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *readiness) add(ready bool, err error) {
	if err != nil {
		select {
		case r.warnC <- err:
		default:
		}
	}
	r.Lock()
	defer r.Unlock()
	r.ready = ready
}

func (r *readiness) Warn() error {
	select {
	case e := <-r.warnC:
		return e
	default:
		return nil
	}
}

func (r *readiness) Fatal() error {
	select {
	case e := <-r.fatalC:
		return e
	default:
		return nil
	}
}

func (r *readiness) Ready() bool {
	r.RLock()
	defer r.RUnlock()
	return r.ready
}

func (r *readiness) Stop() {
	select {
	case <-r.done:
	default:
		close(r.done)
	}
	<-r.doneAck
}

func (r *readiness) Stopped() <-chan struct{} {
	return r.doneAck
}
