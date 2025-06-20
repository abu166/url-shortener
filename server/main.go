package main

import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	initDB()
	initRedis()
	defer db.Close()
	defer redisClient.Close()

	r := mux.NewRouter()
	r.HandleFunc("/api/shorten", shortenURL).Methods("POST", "OPTIONS")
	r.HandleFunc("/{shortCode}", redirectURL).Methods("GET")

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:3000"}),
		handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		handlers.AllowCredentials(),
	)
	r.Use(corsHandler)

	log.Fatal(http.ListenAndServe(":8080", r))
}
