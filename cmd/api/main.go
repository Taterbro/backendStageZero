package main

import (
	"log"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/handler"
	"github.com/Taterbro/backendStageZero/internal/utils"
)

func main() {
	log.Println("Starting server... Running DB next")
	database.Connect()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/classify", handler.ClassifyHandler)
	mux.HandleFunc("POST /api/profiles", handler.CreateUser)
	mux.HandleFunc("GET /api/seed", handler.Seed)
	mux.HandleFunc("GET /api/profiles", handler.GetAllUsers)
	mux.HandleFunc("GET /api/profiles/search", handler.GetAllUsers)
	mux.HandleFunc("GET /api/profiles/{id}", handler.FindUser)
	mux.HandleFunc("GET /api/dev", handler.DevQuery)
	mux.HandleFunc("DELETE /api/profiles/{id}", handler.DeleteUser)

	// Auth endpoints
	mux.HandleFunc("GET /auth/github", handler.GitHubOAuthHandler)
	mux.HandleFunc("GET /auth/github/callback", handler.GitHubCallbackHandler)
	mux.HandleFunc("POST /auth/refresh", handler.RefreshTokenHandler)
	mux.HandleFunc("POST /auth/logout", handler.LogoutHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: utils.RequestLogger(cors(mux)),
	}

	log.Println("Server running on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
	utils.ClearBucket()

}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
