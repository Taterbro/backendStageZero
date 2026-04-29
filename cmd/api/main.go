package main

import (
	"log"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/handler"
	"github.com/Taterbro/backendStageZero/internal/utils"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting server... Running DB next")
	err := godotenv.Load()
	if err != nil {
		//log.Fatal("Error loading .env file")
		log.Println("couldn't load .env; proceeding since in prod")
	}

	database.Connect()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/classify", handler.ClassifyHandler)
	mux.Handle("POST /api/profiles", utils.AuthMiddleware(utils.GeneralLimiter(http.HandlerFunc(handler.CreateUser))))
	mux.HandleFunc("GET /api/seed", handler.Seed)
	mux.Handle("GET /api/profiles", utils.AuthMiddleware(utils.GeneralLimiter(http.HandlerFunc(handler.GetAllUsers))))
	mux.Handle("GET /api/profiles/search", utils.AuthMiddleware(utils.GeneralLimiter(http.HandlerFunc(handler.GetAllUsers))))
	mux.HandleFunc("GET /api/profiles/{id}", handler.FindUser)
	mux.HandleFunc("GET /api/dev", handler.DevQuery)
	mux.HandleFunc("DELETE /api/profiles/{id}", handler.DeleteUser)
	mux.Handle("GET /api/auth/github", utils.AuthLimiter(http.HandlerFunc(handler.GitHubAuth)))
	mux.Handle("GET /api/auth/github/callback", utils.AuthLimiter(http.HandlerFunc(handler.GitHubCallback)))
	mux.Handle("GET /api/auth/refresh", utils.AuthLimiter(http.HandlerFunc(handler.Refresh)))
	mux.Handle("GET /api/auth/logout", utils.AuthMiddleware(utils.AuthLimiter(http.HandlerFunc(handler.Logout))))

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
