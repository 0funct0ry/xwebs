package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"golang.org/x/time/rate"
)

// SecurityManager handles IP filtering and rate limiting.
type SecurityManager struct {
	allowedIPs     []net.IPNet
	deniedIPs      []net.IPNet
	allowedOrigins []string

	globalLimiter *rate.Limiter

	mu           sync.RWMutex
	clientLimits map[string]*clientLimiter

	perClientRate  rate.Limit
	perClientBurst int

	verbose bool
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewSecurityManager creates a new security manager with the given options.
func NewSecurityManager(opts *Options) (*SecurityManager, error) {
	sm := &SecurityManager{
		clientLimits:   make(map[string]*clientLimiter),
		allowedOrigins: opts.AllowedOrigins,
		verbose:        opts.Verbose,
	}

	// Parse allowed IPs
	for _, ipStr := range opts.AllowIPs {
		cidr := ipStr
		if !strings.Contains(ipStr, "/") {
			cidr = ipStr + "/32"
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid allow-ip %s: %w", ipStr, err)
		}
		sm.allowedIPs = append(sm.allowedIPs, *ipNet)
	}

	// Parse denied IPs
	for _, ipStr := range opts.DenyIPs {
		cidr := ipStr
		if !strings.Contains(ipStr, "/") {
			cidr = ipStr + "/32"
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid deny-ip %s: %w", ipStr, err)
		}
		sm.deniedIPs = append(sm.deniedIPs, *ipNet)
	}

	// Parse Rate Limit
	if opts.RateLimit != "" {
		parts := strings.Split(opts.RateLimit, ",")
		if len(parts) > 0 && parts[0] != "" {
			// Per-client limit
			perS, burst, err := handler.ParseRateLimit(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid per-client rate limit: %w", err)
			}
			sm.perClientRate = rate.Limit(perS)
			sm.perClientBurst = burst
		}
		if len(parts) > 1 && parts[1] != "" {
			// Global limit
			perS, burst, err := handler.ParseRateLimit(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid global rate limit: %w", err)
			}
			sm.globalLimiter = rate.NewLimiter(rate.Limit(perS), burst)
		}
	}

	return sm, nil
}

// IsIPAllowed checks if an IP address is allowed to connect.
func (sm *SecurityManager) IsIPAllowed(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr // Fallback if no port
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check denied list first
	for _, ipNet := range sm.deniedIPs {
		if ipNet.Contains(ip) {
			return false
		}
	}

	// If allowed list is empty, all are allowed (except denied)
	if len(sm.allowedIPs) == 0 {
		return true
	}

	// Check allowed list
	for _, ipNet := range sm.allowedIPs {
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// CheckRateLimit checks if the request exceeds the rate limits.
func (sm *SecurityManager) CheckRateLimit(remoteAddr string) bool {
	// Global Rate Limit
	if sm.globalLimiter != nil && !sm.globalLimiter.Allow() {
		if sm.verbose {
			fmt.Printf("[security] global rate limit exceeded\n")
		}
		return false
	}

	// Per-Client Rate Limit
	if sm.perClientRate > 0 {
		host, _, _ := net.SplitHostPort(remoteAddr)
		if host == "" {
			host = remoteAddr
		}

		sm.mu.Lock()
		cl, ok := sm.clientLimits[host]
		if !ok {
			cl = &clientLimiter{
				limiter: rate.NewLimiter(sm.perClientRate, sm.perClientBurst),
			}
			sm.clientLimits[host] = cl
		}
		cl.lastSeen = time.Now()
		sm.mu.Unlock()

		if !cl.limiter.Allow() {
			if sm.verbose {
				fmt.Printf("[security] per-client rate limit exceeded: %s\n", host)
			}
			return false
		}
	}

	return true
}

// IsOriginAllowed checks if the origin is allowed for WebSocket connections.
func (sm *SecurityManager) IsOriginAllowed(r *http.Request) bool {
	if len(sm.allowedOrigins) == 0 {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		// If Origin is missing, it's not a browser cross-origin request.
		// Usually we allow if missing, but if Origins are restricted,
		// maybe we should be strict? gorilla/websocket handles this.
		// We'll follow gorilla's lead and return true if Origin matches Host.
		// But here we are just checking against our allowed list.
		return true
	}

	for _, o := range sm.allowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}

	if sm.verbose {
		fmt.Printf("[security] origin denied: %s\n", origin)
	}
	return false
}

// Middleware returns an HTTP middleware that performs security checks.
func (sm *SecurityManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. IP Check
		if !sm.IsIPAllowed(r.RemoteAddr) {
			if sm.verbose {
				fmt.Printf("[security] IP denied: %s\n", r.RemoteAddr)
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// 2. Rate Limit Check
		if !sm.CheckRateLimit(r.RemoteAddr) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Cleanup garbage collects old client limiters.
func (sm *SecurityManager) Cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	now := time.Now()
	for host, cl := range sm.clientLimits {
		if now.Sub(cl.lastSeen) > 1*time.Hour {
			delete(sm.clientLimits, host)
		}
	}
}
