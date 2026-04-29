package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/golang-jwt/jwt/v5"
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

func ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("invalid signing method")
		}

		return []byte(os.Getenv("JWT_SECRET")), nil
	})
}

func GetUserIDFromToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("user id missing")
	}

	return userID, nil
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		if header == "" {
			WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
				Status:  "error",
				Message: "missing token; unauthorized",
			})
			return
		}

		parts := strings.Split(header, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
				Status:  "error",
				Message: "invalid auth header",
			})
			return
		}

		token, err := ValidateToken(parts[1])
		if err != nil || !token.Valid {
			log.Println("validate token err: ", err)
			WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
				Status:  "error",
				Message: "invalid token",
			})
			return
		}

		next.ServeHTTP(w, r)
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
