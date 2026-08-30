package services

import (
	"context"
	"sync"
	"time"
)

type RequestPacer struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
	now      func() time.Time
	after    func(time.Duration) <-chan time.Time
}

func NewRequestPacer(interval time.Duration) *RequestPacer {
	return &RequestPacer{
		interval: interval,
		now:      time.Now,
		after:    time.After,
	}
}

func (p *RequestPacer) Wait(ctx context.Context) error {
	if p == nil || p.interval <= 0 {
		return nil
	}

	for {
		wait := p.delayUntilNextSlot()
		if wait <= 0 {
			return nil
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.after(wait):
		}
	}
}

func (p *RequestPacer) delayUntilNextSlot() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.now()
	if p.next.IsZero() || !now.Before(p.next) {
		p.next = now.Add(p.interval)
		return 0
	}

	return p.next.Sub(now)
}
