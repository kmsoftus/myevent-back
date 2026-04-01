package middleware

import (
	"math"
	"net"
	nethttp "net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	apphttp "myevent-back/internal/http"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
	limit   int
	window  time.Duration
	hits    uint64
}

func NewIPRateLimit(limit int, window time.Duration) func(nethttp.Handler) nethttp.Handler {
	limiter := &ipRateLimiter{
		entries: make(map[string]rateLimitEntry),
		limit:   limit,
		window:  window,
	}

	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			allowed, retryAfter := limiter.allow(clientIP(r), time.Now().UTC())
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				apphttp.WriteErrorResponse(
					w,
					nethttp.StatusTooManyRequests,
					"Muitas tentativas em pouco tempo. Tente novamente em instantes.",
					"rate_limited",
					nil,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (l *ipRateLimiter) allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hits++
	if l.hits%256 == 0 {
		for existingKey, entry := range l.entries {
			if !entry.resetAt.After(now) {
				delete(l.entries, existingKey)
			}
		}
	}

	entry, ok := l.entries[key]
	if !ok || !entry.resetAt.After(now) {
		l.entries[key] = rateLimitEntry{
			count:   1,
			resetAt: now.Add(l.window),
		}
		return true, 0
	}

	if entry.count >= l.limit {
		retryAfter := int(math.Ceil(entry.resetAt.Sub(now).Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}

	entry.count++
	l.entries[key] = entry
	return true, 0
}

func clientIP(r *nethttp.Request) string {
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return "unknown"
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}

	return host
}
