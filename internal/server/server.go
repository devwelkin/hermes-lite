package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"

	"github.com/devwelkin/hermes-lite/internal/request"
	"github.com/devwelkin/hermes-lite/internal/response"
)

// HandlerError is a structured error for http handlers
type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

// Handler is the function signature for handling requests.
type Handler func(w io.Writer, req *request.Request) *HandlerError

// Server holds the state for our http server
type Server struct {
	listener net.Listener
	handler  Handler // the user-provided handler
	closed   atomic.Bool
}

func Serve(port int, handler Handler) (*Server, error) {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: listener,
		handler:  handler, // store the handler
	}

	go s.listen()

	return s, nil
}

// Close gracefully shuts down the server
func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

// listen is the main accept loop
func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				log.Println("listener closed, server shutting down.")
				return
			}
			log.Printf("error accepting connection: %v", err)
			continue
		}
		// pass the handler to the connection handler
		go s.handle(conn, s.handler)
	}
}

// writeErrorResponse is our new dry helper for sending error pages
func (s *Server) writeErrorResponse(conn io.Writer, handlerErr *HandlerError) {
	body := []byte(handlerErr.Message)
	h := response.GetDefaultHeaders(len(body))

	// 1. write status
	if err := response.WriteStatusLine(conn, handlerErr.StatusCode); err != nil {
		log.Printf("error writing error status line: %v", err)
		return
	}

	// 2. write headers
	if err := response.WriteHeaders(conn, h); err != nil {
		log.Printf("error writing error headers: %v", err)
		return
	}

	// 3. write body
	if _, err := conn.Write(body); err != nil {
		log.Printf("error writing error body: %v", err)
	}
}

func (s *Server) handle(conn net.Conn, handler Handler) {
	defer conn.Close()

	// 1. parse the request
	req, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("error parsing request: %v", err)
		s.writeErrorResponse(conn, &HandlerError{
			StatusCode: response.StatusBadRequest,
			Message:    "Bad Request\n",
		})
		return
	}

	// 2. create a buffer for the handler to write its body to
	bodyBuf := new(bytes.Buffer)

	// 3. call the user's handler
	handlerErr := handler(bodyBuf, req)

	// 4. if handler errors, write the error response
	if handlerErr != nil {
		s.writeErrorResponse(conn, handlerErr)
		return
	}

	// 5. if handler succeeds, write the successful response
	h := response.GetDefaultHeaders(bodyBuf.Len())

	// 5b. write status line (200 OK)
	if err := response.WriteStatusLine(conn, response.StatusOK); err != nil {
		log.Printf("error writing success status line: %v", err)
		return
	}

	// 5c. write headers
	if err := response.WriteHeaders(conn, h); err != nil {
		log.Printf("error writing success headers: %v", err)
		return
	}

	// 5d. write body (from the buffer)
	if _, err := bodyBuf.WriteTo(conn); err != nil {
		log.Printf("error writing success body: %v", err)
	}
}
