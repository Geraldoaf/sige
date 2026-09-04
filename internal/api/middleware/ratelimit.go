package middleware

import (
	"net"
	"net/http"
	"sige/internal/api/presenter"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type LimiterEntry struct {
	Limiter  *rate.Limiter
	LastSeen time.Time
}

// IPRateLimiter gerencia limites de taxa por endereço IP com suporte a proxy confiável.
type IPRateLimiter struct {
	mu             sync.Mutex
	limiters       map[string]*LimiterEntry
	trustedProxies []string
	rate           rate.Limit
	burst          int
	ttl            time.Duration
	stopCleanup    chan struct{}
}

func NewIPRateLimiter(r rate.Limit, burst int, trustedProxies []string) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters:       make(map[string]*LimiterEntry),
		trustedProxies: trustedProxies,
		rate:           r,
		burst:          burst,
		ttl:            5 * time.Minute,
		stopCleanup:    make(chan struct{}),
	}
	go limiter.cleanupLoop(1 * time.Minute)
	return limiter
}

func (l *IPRateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			for ip, entry := range l.limiters {
				if time.Since(entry.LastSeen) > l.ttl {
					delete(l.limiters, ip)
				}
			}
			l.mu.Unlock()
		case <-l.stopCleanup:
			return
		}
	}
}

func (l *IPRateLimiter) Close() {
	select {
	case <-l.stopCleanup:
	default:
		close(l.stopCleanup)
	}
}

func (l *IPRateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := l.extractIP(r)
			l.mu.Lock()
			entry, exists := l.limiters[ip]
			if !exists {
				entry = &LimiterEntry{
					Limiter: rate.NewLimiter(l.rate, l.burst),
				}
				l.limiters[ip] = entry
			}
			entry.LastSeen = time.Now()
			allowed := entry.Limiter.Allow()
			l.mu.Unlock()

			if !allowed {
				presenter.RenderError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "Taxa limite de requisições por segundo excedida", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (l *IPRateLimiter) extractIP(r *http.Request) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}
	for _, proxy := range l.trustedProxies {
		if remoteIP == strings.TrimSpace(proxy) {
			if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				return realIP
			}
		}
	}
	return remoteIP
}
