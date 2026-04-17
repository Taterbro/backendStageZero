package main

import (
	"log"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/classify", handler.ClassifyHandler)
	mux.HandleFunc("POST /api/profiles", handler.CreateUser)
	mux.HandleFunc("GET /api/profiles/{id}", handler.FindUser)
	mux.HandleFunc("GET /api/profiles", handler.GetAllUsers)
	mux.HandleFunc("DELETE /api/profiles/{id}", handler.DeleteUser)


	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server running on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}