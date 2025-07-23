package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetJoke(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Test case 1: Successful retrieval of a joke
	rows := sqlmock.NewRows([]string{"id", "entry_date", "author", "joke_text"}).
		AddRow(1, time.Now(), "Test Author", "Test Joke")
	mock.ExpectQuery("SELECT id, entry_date, author, joke_text FROM jokes ORDER BY RANDOM() LIMIT 1").WillReturnRows(rows)

	req, err := http.NewRequest("GET", "/random", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getJoke(db, w, r)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedJoke := Joke{Id: 1, Author: "Test Author", Text: "Test Joke"}
	var actualJoke Joke
	err = json.NewDecoder(rr.Body).Decode(&actualJoke)
	if err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	// Compare only relevant fields as entry_date will differ
	if actualJoke.Id != expectedJoke.Id || actualJoke.Author != expectedJoke.Author || actualJoke.Text != expectedJoke.Text {
		t.Errorf("handler returned unexpected body: got %v want %v",
			actualJoke, expectedJoke)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 2: No jokes found (sql.ErrNoRows)
	mock.ExpectQuery("SELECT id, entry_date, author, joke_text FROM jokes ORDER BY RANDOM() LIMIT 1").WillReturnError(sql.ErrNoRows)

	req, err = http.NewRequest("GET", "/random", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code for no rows: got %v want %v",
			status, http.StatusNotFound)
	}

	expectedErrorResponse := map[string]string{"message": "No jokes found in the database."}
	var actualErrorResponse map[string]string
	err = json.NewDecoder(rr.Body).Decode(&actualErrorResponse)
	if err != nil {
		t.Fatalf("could not decode error response: %v", err)
	}
	if actualErrorResponse["message"] != expectedErrorResponse["message"] {
		t.Errorf("handler returned unexpected error message: got %v want %v",
			actualErrorResponse["message"], expectedErrorResponse["message"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 3: Database error
	mock.ExpectQuery("SELECT id, entry_date, author, joke_text FROM jokes ORDER BY RANDOM() LIMIT 1").WillReturnError(errors.New("database connection error"))

	req, err = http.NewRequest("GET", "/random", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code for database error: got %v want %v",
			status, http.StatusInternalServerError)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSaveJoke(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Test case 1: Successful joke submission
	joke := Joke{Author: "New Author", Text: "New Joke Text"}
	jsonJoke, _ := json.Marshal(joke)

	mock.ExpectExec("INSERT INTO jokes (author, joke_text) VALUES ($1, $2)").WithArgs(joke.Author, joke.Text).WillReturnResult(sqlmock.NewResult(1, 1))

	req, err := http.NewRequest("POST", "/write", bytes.NewBuffer(jsonJoke))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saveJoke(db, w, r)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}

	var actualJoke Joke
	err = json.NewDecoder(rr.Body).Decode(&actualJoke)
	if err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if actualJoke.Author != joke.Author || actualJoke.Text != joke.Text {
		t.Errorf("handler returned unexpected body: got %v want %v",
			actualJoke, joke)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 2: Invalid JSON payload
	req, err = http.NewRequest("POST", "/write", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid JSON: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Test case 3: Database error during insert
	joke = Joke{Author: "Another Author", Text: "Another Joke Text"}
	jsonJoke, _ = json.Marshal(joke)

	mock.ExpectExec("INSERT INTO jokes (author, joke_text) VALUES ($1, $2)").WithArgs(joke.Author, joke.Text).WillReturnError(errors.New("database insert error"))

	req, err = http.NewRequest("POST", "/write", bytes.NewBuffer(jsonJoke))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code for database insert error: got %v want %v",
			status, http.StatusInternalServerError)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Create a handler that will be wrapped by the middleware
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the rate limit middleware
	testHandler := rateLimitMiddleware(nextHandler)

	// Create a request
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"

	// Create a response recorder
	rr := httptest.NewRecorder()

	// First request should be allowed
	testHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	// Second and third requests should be allowed
	rr = httptest.NewRecorder()
	testHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	testHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	// Fourth request should be rate limited
	rr = httptest.NewRecorder()
	testHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status Too Many Requests, got %d", rr.Code)
	}
}

func TestSaveJokeInputValidation(t *testing.T) {
	tests := []struct {
		name           string
		author         string
		jokeText       string
		expectedStatus int
		expectedMessage string
	}{
		{
			name:           "Empty Author",
			author:         "",
			jokeText:       "Valid joke text.",
			expectedStatus: http.StatusBadRequest,
			expectedMessage: "Author cannot be empty.",
		},
		{
			name:           "Author Too Long",
			author:         string(make([]byte, 256)), // 256 characters
			jokeText:       "Valid joke text.",
			expectedStatus: http.StatusBadRequest,
			expectedMessage: "Author exceeds maximum length of 255 characters.",
		},
		{
			name:           "Empty Joke Text",
			author:         "Valid Author",
			jokeText:       "",
			expectedStatus: http.StatusBadRequest,
			expectedMessage: "Joke text cannot be empty.",
		},
		{
			name:           "Joke Text Too Long",
			author:         "Valid Author",
			jokeText:       string(make([]byte, 2001)), // 2001 characters
			expectedStatus: http.StatusBadRequest,
			expectedMessage: "Joke text exceeds maximum length of 2000 characters.",
		},
		{
			name:           "Valid Input",
			author:         "Valid Author",
			jokeText:       "This is a valid joke.",
			expectedStatus: http.StatusCreated,
			expectedMessage: "", // No error message for success
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new db and mock for each subtest
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				saveJoke(db, w, r)
			})

			joke := Joke{Author: tt.author, Text: tt.jokeText}
			jsonJoke, _ := json.Marshal(joke)

			req, err := http.NewRequest("POST", "/write", bytes.NewBuffer(jsonJoke))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Set up mock expectation for valid input BEFORE serving the request
			if tt.name == "Valid Input" {
				mock.ExpectExec("INSERT INTO jokes (author, joke_text) VALUES ($1, $2)").WithArgs(tt.author, tt.jokeText).WillReturnResult(sqlmock.NewResult(1, 1))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.expectedMessage != "" {
				var actualErrorResponse map[string]string
				err = json.NewDecoder(rr.Body).Decode(&actualErrorResponse)
				if err != nil {
					t.Fatalf("could not decode error response: %v", err)
				}
				if actualErrorResponse["message"] != tt.expectedMessage {
					t.Errorf("handler returned unexpected error message: got %q want %q",
						actualErrorResponse["message"], tt.expectedMessage)
				}
			}

			// Verify expectations were met for this subtest
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
