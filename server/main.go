package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB
var redisClient *redis.Client

type URLRecord struct {
	ShortCode   string    `json:"shortCode"`
	OriginalURL string    `json:"originalUrl"`
	CreatedAt   time.Time `json:"createdAt"`
}

func initDB() {
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbName)
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
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
}

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

	// Handle collision by regenerating if the short code already exists
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

func generateShortCode(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal("Error generating random string:", err)
	}
	for i := 0; i < length; i++ {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

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