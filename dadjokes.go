package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/time/rate"
)

type Joke struct {
	Id     int       `json:"id"`
	Date   time.Time `json:"entry_date"`
	Author string    `json:"author"`
	Text   string    `json:"joke_text"`
}

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the the map to hold values of the type visitor.
var visitors = make(map[string]*visitor)
var mu sync.Mutex

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	go cleanupVisitors()
}

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	db, err = sql.Open("postgres", os.Getenv("DB_CONN_STRING"))
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	router := mux.NewRouter()

	router.HandleFunc("/random", func(w http.ResponseWriter, r *http.Request) {
		getJoke(db, w, r)
	}).Methods("GET")
	router.Handle("/write", rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saveJoke(db, w, r)
	}))).Methods("POST")

	log.Fatal(http.ListenAndServe(":3000", router))
}

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 3)
		// Include the current time when creating a new visitor.
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()

		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}

		mu.Unlock()
	}
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		limiter := getVisitor(ip)
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getJoke(db *sql.DB, response http.ResponseWriter, request *http.Request) {
    var joke Joke
    err := db.QueryRow("SELECT id, entry_date, author, joke_text FROM jokes ORDER BY RANDOM() LIMIT 1").Scan(&joke.Id, &joke.Date, &joke.Author, &joke.Text)
    if err != nil {
        if err == sql.ErrNoRows {
            response.WriteHeader(http.StatusNotFound)
            json.NewEncoder(response).Encode(map[string]string{"message": "No jokes found in the database."})
            return
        }
        log.Printf("Error getting joke: %v", err)
        response.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(response).Encode(map[string]string{"message": "An internal server error occurred."})
        return
    }

    response.Header().Set("Content-Type", "application/json")
    json.NewEncoder(response).Encode(joke)
}

func saveJoke(db *sql.DB, response http.ResponseWriter, request *http.Request) {
    var joke Joke
    err := json.NewDecoder(request.Body).Decode(&joke)
    if err != nil {
        response.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(response).Encode(map[string]string{"message": err.Error()})
        return
    }

    // Input validation
    if joke.Author == "" {
        response.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(response).Encode(map[string]string{"message": "Author cannot be empty."})
        return
    }
    if len(joke.Author) > 255 {
        response.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(response).Encode(map[string]string{"message": "Author exceeds maximum length of 255 characters."})
        return
    }
    if joke.Text == "" {
        response.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(response).Encode(map[string]string{"message": "Joke text cannot be empty."})
        return
    }
    if len(joke.Text) > 2000 {
        response.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(response).Encode(map[string]string{"message": "Joke text exceeds maximum length of 2000 characters."})
        return
    }

    _, err = db.Exec("INSERT INTO jokes (author, joke_text) VALUES ($1, $2)", joke.Author, joke.Text)
    if err != nil {
        log.Printf("Error saving joke: %v", err)
        response.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(response).Encode(map[string]string{"message": "An internal server error occurred."})
        return
    }

    response.WriteHeader(http.StatusCreated)
    response.Header().Set("Content-Type", "application/json")
    json.NewEncoder(response).Encode(joke)
}
