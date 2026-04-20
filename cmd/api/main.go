package main

import (
	"log"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/handler"
)


func main() {
	database.Connect()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/classify", handler.ClassifyHandler)
	mux.HandleFunc("POST /api/profiles", handler.CreateUser)
	mux.HandleFunc("GET /api/profiles/{id}", handler.FindUser)
	mux.HandleFunc("GET /api/profiles", handler.GetAllUsers)
	mux.HandleFunc("DELETE /api/profiles/{id}", handler.DeleteUser)


	server := &http.Server{
		Addr:    ":8080",
		Handler: cors(mux),
	}

	log.Println("Server running on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
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