package main

import (
	"fmt"
	"log"
	"net"

	"github.com/devwelkin/hermes-lite/util"
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

		lines := util.GetLinesChannel(conn)
		for line := range lines {
			fmt.Printf("%s\n", line)
		}
	}
}
