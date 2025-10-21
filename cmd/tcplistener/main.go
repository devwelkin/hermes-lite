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
			log.Println("failed to accept connection:", err) // don't fatal
			continue
		}

		go handleConnection(conn) // handle each connection concurrently
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close() // <--- THIS IS THE BIG ONE. DEFER CLOSE.

	req, err := request.RequestFromReader(conn)
	if err != nil {
		// don't fatal, just log the error for this one connection
		log.Println("failed to parse request:", err)
		return
	}

	// your print statements
	fmt.Println("Request line:")
	fmt.Printf("- Method: %s\n", req.RequestLine.Method)
	fmt.Printf("- Target: %s\n", req.RequestLine.RequestTarget)
	fmt.Printf("- Version: %s\n", req.RequestLine.HTTPVersion)
	fmt.Println("Headers:")
	for key, value := range req.Headers {
		fmt.Printf("- %s: %s\n", key, value)
	}
	fmt.Println("Body:")
	fmt.Printf("%s", req.Body)
	fmt.Println("--- end of request ---")

	conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"))
}
