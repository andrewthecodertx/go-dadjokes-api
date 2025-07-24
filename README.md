# Go Dad Jokes API

A simple and fun REST API service written in Go that serves and stores dad
jokes. The API allows you to fetch random dad jokes and submit new ones to the
collection.

[CHECK IT OUT HERE](https://dadjokes.developersandbox.xyz/api/v2/random)

## Features

- Fetch random dad jokes
- Submit new dad jokes
- IP-based rate limiting for POST requests
- Input validation for joke submission
- PostgreSQL database integration
- Secure HTTPS support via Nginx
- Environment-based configuration

## Prerequisites

- Go 1.x or higher
- PostgreSQL database
- Nginx (for production deployment)
- Let's Encrypt SSL certificates (for HTTPS)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/andrewthecodertx/go-dadjokes-api.git
cd go-dadjokes-api
```

1. Install dependencies:

```bash
go mod init go-dadjokes-api
go get github.com/gorilla/mux
go get github.com/joho/godotenv
go get github.com/lib/pq
```

1. Create a `.env` file in the project root with your database configuration:

```env
DB_CONN_STRING="postgres://user:password@host:5432/database_name?sslmode=disable"
```

1. Set up the PostgreSQL database:

```sql
CREATE TABLE jokes (
    id SERIAL PRIMARY KEY,
    entry_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    author VARCHAR(255),
    joke_text TEXT
);
```

## Running the Application

### Development

```bash
go run main.go
```

The server will start on port 3000.

### Production

1. Build the binary:

```bash
go build -o dadjokes-api
```

2. Configure Nginx using the provided configuration:

```nginx
server {
    ...

    location /api/v2/random {
        proxy_pass http://localhost:8080/random;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /api/v2/submit {
        proxy_pass http://localhost:8080/write;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_method POST;
        proxy_pass_request_headers on;
    }
}
```

## API Endpoints

### Get Random Joke

```http
GET /api/v2/random
```

Response:

```json
{
    "id": 1,
    "entry_date": "2024-01-06T12:00:00Z",
    "author": "John Doe",
    "joke_text": "Why don't eggs tell jokes? They'd crack up!"
}
```

### Submit New Joke

```http
POST /api/v2/submit
Content-Type: application/json

{
    "author": "Jane Doe",
    "joke_text": "Why don't programmers like nature? It has too many bugs!"
}
```

Response:

```json
{
    "id": 2,
    "entry_date": "2024-01-06T12:01:00Z",
    "author": "Jane Doe",
    "joke_text": "Why don't programmers like nature? It has too many bugs!"
}
```

## Security Considerations

- The API uses HTTPS encryption in production
- Nginx acts as a reverse proxy
- Database credentials are stored in environment variables (use a dedicated, least-privilege user in production)
- Input validation is implemented for joke submission
- IP-based rate limiting is implemented for POST requests
- Generic error messages are returned to clients to prevent sensitive information leakage, with detailed errors logged internally
- HTTP Security Headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Strict-Transport-Security) are recommended for Nginx configuration
- Regular dependency updates and vulnerability scanning (e.g., using `govulncheck`) are crucial for maintaining security

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Thanks to all contributors who add their dad jokes
- Built with Go, PostgreSQL, and Nginx
