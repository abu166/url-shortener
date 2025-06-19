package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"github.com/gorilla/handlers"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/go-redis/redis/v8"
)

var db *sql.DB
var redisClient *redis.Client

type URLRecord struct {
	ShortCode  string    `json:"shortCode"`
	OriginalURL string `json:"originalUrl"`
	CreatedAt  time.Time `json:"createdAt"`
}

func initDB() {
	connStr := "user=postgres password=admin dbname=url_shortener sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		short_code VARCHAR(10) PRIMARY KEY,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal(err)
	}
}

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LongUrl string `json:"longUrl"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Simple validation
	if req.LongUrl == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Generate short code (simplified)
	shortCode := generateShortCode(6)

	// Store in DB
	_, err := db.Exec("INSERT INTO urls (short_code, original_url) VALUES ($1, $2)", shortCode, req.LongUrl)
	if err != nil {
		http.Error(w, "Failed to store URL", http.StatusInternalServerError)
		return
	}

	// Cache in Redis
	redisClient.Set(r.Context(), shortCode, req.LongUrl, 0)

	json.NewEncoder(w).Encode(map[string]string{"shortCode": shortCode})
}

func redirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	// Check Redis cache
	cachedURL, err := redisClient.Get(r.Context(), shortCode).Result()
	if err == redis.Nil {
		// Cache miss, check DB
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

func generateShortCode(length int) string {
	// Simplified: Generate a random string (in production, use a better method)
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[i%len(chars)]
	}
	return string(b)
}

func main() {
	initDB()
	initRedis()
	defer db.Close()
	defer redisClient.Close()

	r := mux.NewRouter()
	r.HandleFunc("/api/shorten", shortenURL).Methods("POST")
	r.HandleFunc("/{shortCode}", redirectURL).Methods("GET")

	r.Use(handlers.CORS(handlers.AllowedOrigins([]string{"http://localhost:3001"})))
	log.Fatal(http.ListenAndServe(":8081", r))
}