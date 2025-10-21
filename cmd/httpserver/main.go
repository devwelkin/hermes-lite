package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/devwelkin/hermes-lite/internal/headers"
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

func proxyHandler(w *response.Writer, req *request.Request) {
	path := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin")
	targetURL := "https://httpbin.org" + path
	log.Printf("proxying to %s", targetURL)

	// Make the request to the target server
	resp, err := http.Get(targetURL)
	if err != nil {
		log.Printf("error making request to httpbin: %v", err)
		// Send an error response back to the client
		body := []byte(htmlInternalError)
		h := response.GetDefaultHeaders(len(body))
		h.Set("Content-Type", "text/html")
		_ = w.WriteStatusLine(response.StatusInternalServerError)
		_ = w.WriteHeaders(h)
		_, _ = w.WriteBody(body)
		return
	}
	defer resp.Body.Close()

	// Prepare headers for the chunked response
	h := headers.NewHeaders()
	// Copy headers from httpbin response, but skip Content-Length and Transfer-Encoding
	for key, values := range resp.Header {
		lowerKey := strings.ToLower(key)
		if lowerKey != "content-length" && lowerKey != "transfer-encoding" {
			h.Set(key, strings.Join(values, ", "))
		}
	}
	h.Set("Transfer-Encoding", "chunked")
	h.Set("Connection", "close") // good practice for simple servers

	// Write status line and headers
	if err := w.WriteStatusLine(response.StatusCode(resp.StatusCode)); err != nil {
		log.Printf("error writing proxy status line: %v", err)
		return
	}
	if err := w.WriteHeaders(h); err != nil {
		log.Printf("error writing proxy headers: %v", err)
		return
	}

	// Stream the body chunk by chunk
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			log.Printf("read %d bytes from httpbin, writing as chunk", n)
			if _, writeErr := w.WriteChunkedBody(buf[:n]); writeErr != nil {
				log.Printf("error writing chunked body: %v", writeErr)
				break // Stop streaming if we can't write to the client
			}
		}
		if err == io.EOF {
			break // End of body
		}
		if err != nil {
			log.Printf("error reading from httpbin body: %v", err)
			break
		}
	}

	// Signal end of chunked response
	if _, err := w.WriteChunkedBodyDone(); err != nil {
		log.Printf("error writing chunked body done: %v", err)
	}
}

func myHandler(w *response.Writer, req *request.Request) {
	// Route to proxy if the path matches
	if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin/") {
		proxyHandler(w, req)
		return
	}

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
