package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/devwelkin/hermes-lite/internal/request"
	"github.com/devwelkin/hermes-lite/internal/response"
	"github.com/devwelkin/hermes-lite/internal/server"
)

const port = 42069

// define the html responses
const (
	htmlOK = `<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body></html>`

	htmlBadRequest = `<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body></html>`

	htmlInternalError = `<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body></html>`
)

func myHandler(w *response.Writer, req *request.Request) {
	var body string
	var statusCode response.StatusCode

	switch req.RequestLine.RequestTarget {
	case "/yourproblem":
		statusCode = response.StatusBadRequest
		body = htmlBadRequest
	case "/myproblem":
		statusCode = response.StatusInternalServerError
		body = htmlInternalError
	default:
		statusCode = response.StatusOK
		body = htmlOK
	}

	bodyBytes := []byte(body)
	h := response.GetDefaultHeaders(len(bodyBytes))
	h.Set("Content-Type", "text/html")

	if err := w.WriteStatusLine(statusCode); err != nil {
		log.Printf("error writing status line: %v", err)
		return
	}
	if err := w.WriteHeaders(h); err != nil {
		log.Printf("error writing headers: %v", err)
		return
	}
	if _, err := w.WriteBody(bodyBytes); err != nil {
		log.Printf("error writing body: %v", err)
	}
}

func main() {
	server, err := server.Serve(port, myHandler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
