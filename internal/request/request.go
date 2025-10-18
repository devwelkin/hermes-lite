// request.go

package request

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Custom errors
var (
	ErrInvalidRequestFormat = errors.New("invalid request line format")
	ErrUnsupportedHTTP      = errors.New("unsupported http version")
)

type Request struct {
	RequestLine RequestLine
}

type RequestLine struct {
	HTTPVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	r := bufio.NewReader(reader)

	line, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("could not read request line: %w", err)
	}

	rqline, err := parseRequestLine(line)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request line: %w", err)
	}

	req := &Request{
		RequestLine: *rqline,
	}

	return req, nil
}

func parseRequestLine(data string) (*RequestLine, error) {
	// trim \r\n
	firstline := strings.TrimSpace(data)

	parts := strings.Split(firstline, " ")

	// panic guard
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected 3 parts, got %d", ErrInvalidRequestFormat, len(parts))
	}

	method := parts[0]
	target := parts[1]
	versionRaw := parts[2]

	http, httpv, ok := strings.Cut(versionRaw, "/")

	if !ok || http != "HTTP" || (httpv != "1.1" && httpv != "1.0") {
		return nil, fmt.Errorf("%w: expected 'HTTP/1.1', got '%s'", ErrUnsupportedHTTP, versionRaw)
	}

	reqLine := RequestLine{
		Method:        method,
		RequestTarget: target,
		HTTPVersion:   httpv,
	}

	return &reqLine, nil
}
