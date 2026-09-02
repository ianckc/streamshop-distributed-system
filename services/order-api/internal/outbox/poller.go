package outbox

import (
	"context"
	"log/slog"
	"time"
)

const DefaultInterval = 300 * time.Millisecond

// Poller drains unpublished outbox rows on a fixed interval.
type Poller struct {
	Store    Store
	Publish  PublishFunc
	Interval time.Duration
}

func (p *Poller) Run(ctx context.Context) {
	interval := p.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.drain(ctx)
		}
	}
}

func (p *Poller) drain(ctx context.Context) {
	for {
		processed, err := p.Store.ProcessNext(ctx, p.Publish)
		if err != nil {
			slog.ErrorContext(ctx, "outbox process failed", "error", err)
			return
		}
		if !processed {
			return
		}
	}
}
