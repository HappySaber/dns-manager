package cluster

import (
	"context"
	"time"
)

// RunLoop calls fn every interval until ctx is canceled. It is used for the
// periodic reconciliation with the peer.
func RunLoop(ctx context.Context, interval time.Duration, fn func(context.Context)) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn(ctx)
		}
	}
}
