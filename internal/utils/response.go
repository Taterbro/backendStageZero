package utils

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Taterbro/backendStageZero/internal/model"
	"golang.org/x/time/rate"
)

type StatusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func init() {
	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
}
func (r *StatusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.statusCode = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *StatusRecorder) Write(b []byte) (int, error) {
	// If WriteHeader wasn't explicitly called, default to 200
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func WriteJson(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &StatusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		startTime := time.Now()
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.statusCode, time.Since(startTime))
	})
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}
type RateLimiterConfig struct {
	Requests int
	Window   time.Duration
}

var (
	mu      sync.Mutex
	clients = make(map[string]*client)
)

func getIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func ClearBucket() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, c := range clients {
			if time.Since(c.lastSeen) > 3*time.Minute {
				delete(clients, ip)
			}
		}
		mu.Unlock()
	}

}

func NewRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			mu.Lock()
			defer mu.Unlock()

			c, exists := clients[ip]
			if !exists {
				limiter := rate.NewLimiter(
					rate.Every(cfg.Window/time.Duration(cfg.Requests)),
					cfg.Requests,
				)

				c = &client{limiter: limiter}
				clients[ip] = c
			}

			c.lastSeen = time.Now()

			if !c.limiter.Allow() {
				WriteJson(w, http.StatusTooManyRequests, model.ErrorResponse{
					Status:  "error",
					Message: "Too many requests",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

var AuthLimiter = NewRateLimiter(RateLimiterConfig{
	Requests: 10,
	Window:   time.Minute,
})

var GeneralLimiter = NewRateLimiter(RateLimiterConfig{
	Requests: 60,
	Window:   time.Minute,
})
