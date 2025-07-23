package main

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
)

type Joke struct {
    Id     int    `json:"id"`
    Date   string `json:"entry_date"`
    Author string `json:"author"`
    Text   string `json:"joke_text"`
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

    router.HandleFunc("/random", getJoke).Methods("GET")
    router.HandleFunc("/write", saveJoke).Methods("POST")

    log.Fatal(http.ListenAndServe(":3000", router))
}

func getJoke(response http.ResponseWriter, request *http.Request) {
    var joke Joke
    err := db.QueryRow("SELECT id, entry_date, author, joke_text FROM jokes ORDER BY RANDOM() LIMIT 1").Scan(&joke.Id, &joke.Date, &joke.Author, &joke.Text)
    if err != nil {
        if err == sql.ErrNoRows {
            response.WriteHeader(http.StatusNotFound)
            json.NewEncoder(response).Encode(map[string]string{"message": "No jokes found in the database."})
            return
        }
        http.Error(response, err.Error(), http.StatusInternalServerError)
        return
    }

    response.Header().Set("Content-Type", "application/json")
    json.NewEncoder(response).Encode(joke)
}

func saveJoke(response http.ResponseWriter, request *http.Request) {
    var joke Joke
    err := json.NewDecoder(request.Body).Decode(&joke)
    if err != nil {
        http.Error(response, err.Error(), http.StatusBadRequest)
        return
    }

    _, err = db.Exec("INSERT INTO jokes (author, joke_text) VALUES ($1, $2)", joke.Author, joke.Text)
    if err != nil {
        http.Error(response, err.Error(), http.StatusInternalServerError)
        return
    }

    response.WriteHeader(http.StatusCreated)
    response.Header().Set("Content-Type", "application/json")
    json.NewEncoder(response).Encode(joke)
}
