package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/devwelkin/hermes-lite/internal/request"
	"github.com/devwelkin/hermes-lite/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
	listener net.Listener
	handler  Handler // güncellenmiş handler tipi
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
		handler:  handler,
	}

	go s.listen()

	return s, nil
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

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
		go s.handle(conn, s.handler)
	}
}

func (s *Server) handle(conn net.Conn, handler Handler) {
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("error parsing request: %v", err)

		w := response.NewWriter(conn) // yeni writer'ı oluştur

		_ = w.WriteStatusLine(response.StatusBadRequest)
		h := response.GetDefaultHeaders(len("Bad Request\n")) // varsayılan header'lar
		_ = w.WriteHeaders(h)
		_, _ = w.WriteBody([]byte("Bad Request\n")) // body
		return
	}

	resWriter := response.NewWriter(conn)

	handler(resWriter, req)
}
