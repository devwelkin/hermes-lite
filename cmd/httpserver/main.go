package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/devwelkin/hermes-lite/internal/request"
	"github.com/devwelkin/hermes-lite/internal/response"
	"github.com/devwelkin/hermes-lite/internal/server"
)

const port = 42069

// this is our new handler logic
func myHandler(w io.Writer, req *request.Request) *server.HandlerError {
	// check the path
	switch req.RequestLine.RequestTarget {
	case "/yourproblem":
		// return a 400 error
		return &server.HandlerError{
			StatusCode: response.StatusBadRequest,
			Message:    "Your problem is not my problem\n",
		}
	case "/myproblem":
		// return a 500 error
		return &server.HandlerError{
			StatusCode: response.StatusInternalServerError,
			Message:    "Woopsie, my bad\n",
		}
	default:
		// success. write the body to the buffer (w)
		_, err := w.Write([]byte("All good, frfr\n"))
		if err != nil {
			// this is an error writing to the *buffer*,
			// which is weird. return a 500.
			return &server.HandlerError{
				StatusCode: response.StatusInternalServerError,
				Message:    fmt.Sprintf("error writing to buffer: %v\n", err),
			}
		}
		// return nil for success
		return nil
	}
}

func main() {
	// pass our new handler to server.Serve
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
