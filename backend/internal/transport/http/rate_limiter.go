package httptransport

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type clientWindow struct {
	startedAt time.Time
	requests  int
}

type rateLimiter struct {
	mutex         sync.Mutex
	perMinute     int
	perDay        int
	clients       map[string]clientWindow
	dailyDate     string
	dailyRequests int
	now           func() time.Time
}

func newRateLimiter(perMinute, perDay int) *rateLimiter {
	return &rateLimiter{
		perMinute: perMinute,
		perDay:    perDay,
		clients:   make(map[string]clientWindow),
		now:       time.Now,
	}
}

func (l *rateLimiter) allow(clientIP string) (bool, time.Duration) {
	now := l.now().UTC()
	l.mutex.Lock()
	defer l.mutex.Unlock()

	today := now.Format(time.DateOnly)
	if l.dailyDate != today {
		l.dailyDate = today
		l.dailyRequests = 0
	}
	if l.dailyRequests >= l.perDay {
		untilTomorrow := now.Truncate(24 * time.Hour).Add(24 * time.Hour).Sub(now)
		return false, untilTomorrow
	}

	window := l.clients[clientIP]
	if window.startedAt.IsZero() || now.Sub(window.startedAt) >= time.Minute {
		window = clientWindow{startedAt: now}
	}
	if window.requests >= l.perMinute {
		return false, time.Minute - now.Sub(window.startedAt)
	}

	window.requests++
	l.clients[clientIP] = window
	l.dailyRequests++
	return true, 0
}

func (h *Handler) limitRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		allowed, retryAfter := h.rateLimiter.allow(clientAddress(request))
		if !allowed {
			seconds := max(1, int(retryAfter.Round(time.Second).Seconds()))
			response.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeAPIError(response, http.StatusTooManyRequests, "rate_limited", "Request limit reached. Please try again later.")
			return
		}
		next.ServeHTTP(response, request)
	})
}

func clientAddress(request *http.Request) string {
	if forwardedFor := request.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		addresses := strings.Split(forwardedFor, ",")
		if first := strings.TrimSpace(addresses[0]); first != "" {
			return first
		}
	}
	if realIP := strings.TrimSpace(request.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr
}
