package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

func shortenURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LongUrl string `json:"longUrl"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.LongUrl == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	shortCode := generateShortCode(6)

	for {
		err := db.QueryRow("SELECT short_code FROM urls WHERE short_code = $1", shortCode).Scan(&shortCode)
		if err == sql.ErrNoRows {
			break
		}
		shortCode = generateShortCode(6)
	}

	_, err := db.Exec("INSERT INTO urls (short_code, original_url) VALUES ($1, $2)", shortCode, req.LongUrl)
	if err != nil {
		http.Error(w, "Failed to store URL", http.StatusInternalServerError)
		return
	}

	redisClient.Set(r.Context(), shortCode, req.LongUrl, 0)

	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	json.NewEncoder(w).Encode(map[string]string{"shortCode": shortCode})
}

func redirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	cachedURL, err := redisClient.Get(r.Context(), shortCode).Result()
	if err == redis.Nil {
		var originalURL string
		err = db.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&originalURL)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		redisClient.Set(r.Context(), shortCode, originalURL, 0)
		http.Redirect(w, r, originalURL, http.StatusFound)
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, cachedURL, http.StatusFound)
}
