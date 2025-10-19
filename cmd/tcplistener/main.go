package main

import (
	"fmt"
	"log"
	"net"

	"github.com/devwelkin/hermes-lite/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	for {

		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("connection has accepted\n")

		req, err := request.RequestFromReader(conn)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(`Request Line:
- Method: %s
- Target: %s
- Version: %s`, req.RequestLine.Method, req.RequestLine.RequestTarget, req.RequestLine.HTTPVersion)
	}
}
