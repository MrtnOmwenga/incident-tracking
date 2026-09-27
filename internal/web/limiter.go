package web

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// limiter is a token bucket per client. Idle clients are forgotten, so the map stays small.
type limiter struct {
	mu      sync.Mutex
	every   rate.Limit
	burst   int
	clients map[string]*client
	swept   time.Time
}

type client struct {
	bucket *rate.Limiter
	seen   time.Time
}

func newLimiter(every rate.Limit, burst int) *limiter {
	return &limiter{every: every, burst: burst, clients: map[string]*client{}, swept: time.Now()}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.swept) > time.Minute {
		for k, c := range l.clients {
			// A bucket idle long enough to have refilled completely is the same as a new one.
			if now.Sub(c.seen) > time.Duration(float64(l.burst)/float64(l.every)*float64(time.Second)) {
				delete(l.clients, k)
			}
		}
		l.swept = now
	}
	c, ok := l.clients[key]
	if !ok {
		c = &client{bucket: rate.NewLimiter(l.every, l.burst)}
		l.clients[key] = c
	}
	c.seen = now
	return c.bucket.AllowN(now, 1)
}
