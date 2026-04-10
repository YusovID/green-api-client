package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// client хранит rate-limiter и время последней активности для одного IP-адреса.
type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit возвращает HTTP-middleware, ограничивающий запросы по IP с помощью token-bucket алгоритма.
// Фоновая горутина каждые 10 минут удаляет неактивных клиентов из map.
func RateLimit(rps int, burst int) func(http.Handler) http.Handler {
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(10 * time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			mu.Lock()
			if _, exists := clients[ip]; !exists {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(rps), burst),
				}
			}
			clients[ip].lastSeen = time.Now()
			limiter := clients[ip].limiter
			mu.Unlock()

			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
