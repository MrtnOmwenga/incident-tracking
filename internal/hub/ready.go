package hub

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/MrtnOmwenga/lighthouse/internal/monitor"
)

// Readiness answers "is this demo up yet?" for launch pages. Asking also wakes a demo that sleeps
// when idle, since the health request is what wakes it.
//
// Anyone can ask, so answers are cached briefly and concurrent questions about one project share a
// single probe: a crowd of visitors (or a script) can't turn Lighthouse into a load generator
// against its own demos.
type Readiness struct {
	Prober *monitor.Prober
	TTL    time.Duration

	mu     sync.Mutex
	cache  map[string]answer
	flight singleflight.Group
}

type answer struct {
	ready bool
	at    time.Time
}

func NewReadiness(p *monitor.Prober) *Readiness {
	return &Readiness{Prober: p, TTL: 2 * time.Second, cache: map[string]answer{}}
}

func (r *Readiness) Ready(ctx context.Context, p Project) bool {
	if p.Health == "" {
		return true
	}
	r.mu.Lock()
	a, ok := r.cache[p.Slug]
	r.mu.Unlock()
	if ok && time.Since(a.at) < r.TTL {
		return a.ready
	}
	v, _, _ := r.flight.Do(p.Slug, func() (any, error) {
		// Not tied to one visitor's request: others may be waiting on the same probe.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 8*time.Second)
		defer cancel()
		res := r.Prober.HTTP(ctx, monitor.Target{
			URL: p.Health, Timeout: 8 * time.Second, StatusMin: 200, StatusMax: 399,
			// The catalog is the owner's configuration, and health addresses are usually inside
			// the cluster, so private addresses are allowed here (never for visitors' monitors).
			AllowPrivate: true,
		})
		r.mu.Lock()
		r.cache[p.Slug] = answer{ready: res.OK, at: time.Now()}
		r.mu.Unlock()
		return res.OK, nil
	})
	return v.(bool)
}
