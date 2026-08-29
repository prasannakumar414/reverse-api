package helper

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
	KeyFunc  func(*http.Request) string
	Now      func() time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	requests int
	window   time.Duration
	keyFunc  func(*http.Request) string
	now      func() time.Time
	clients  map[string]clientWindow
}

type clientWindow struct {
	count    int
	resetsAt time.Time
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.Requests < 0 {
		config.Requests = 0
	}
	if config.KeyFunc == nil {
		config.KeyFunc = ClientIP
	}
	if config.Now == nil {
		config.Now = time.Now
	}

	return &RateLimiter{
		requests: config.Requests,
		window:   config.Window,
		keyFunc:  config.KeyFunc,
		now:      config.Now,
		clients:  make(map[string]clientWindow),
	}
}

func (l *RateLimiter) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		retryAfter, ok := l.Allow(r)
		if !ok {
			seconds := int((retryAfter + time.Second - time.Nanosecond) / time.Second)
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			WriteError(w, http.StatusTooManyRequests, "rate limit exceeded; retry after "+strconv.Itoa(seconds)+" seconds")
			return
		}

		next(w, r)
	}
}

func (l *RateLimiter) Allow(r *http.Request) (time.Duration, bool) {
	if l == nil || l.requests <= 0 || l.window <= 0 {
		return 0, true
	}

	key := strings.TrimSpace(l.keyFunc(r))
	if key == "" {
		key = "unknown"
	}

	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.clients[key]
	if window.resetsAt.IsZero() || !now.Before(window.resetsAt) {
		l.clients[key] = clientWindow{
			count:    1,
			resetsAt: now.Add(l.window),
		}
		return 0, true
	}

	if window.count >= l.requests {
		return window.resetsAt.Sub(now), false
	}

	window.count++
	l.clients[key] = window
	return 0, true
}

func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		first, _, _ := strings.Cut(forwardedFor, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
