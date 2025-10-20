// request.go

package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/devwelkin/hermes-lite/internal/headers"
)

// Custom errors
var (
	ErrInvalidRequestFormat = errors.New("invalid request line format")
	ErrUnsupportedHTTP      = errors.New("unsupported http version")
)

const (
	stateRequestLine = iota // 0
	stateHeaders            // 1
	stateBody               // 2
	stateDone               // 3
)

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	state       int
}

type RequestLine struct {
	HTTPVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := &Request{
		state:   stateRequestLine,
		Headers: headers.NewHeaders(),
	}
	var accumulatedData []byte

	readBuf := make([]byte, 1024)

	for req.state != stateDone {
		n, err := reader.Read(readBuf)

		if n > 0 {
			accumulatedData = append(accumulatedData, readBuf[:n]...)
		}

		// --- BEGIN FIX ---
		// you have to keep parsing the buffer until it's empty
		// or you're done.
		for {
			consumed, pErr := req.parse(accumulatedData)
			if pErr != nil {
				return nil, pErr
			}

			if consumed == 0 {
				// not enough data in the buffer to parse a full line.
				// break the *inner* loop to read more data.
				break
			}

			accumulatedData = accumulatedData[consumed:]

			if req.state == stateDone {
				// we're done parsing, break the *inner* loop.
				break
			}
		}
		// --- END FIX ---

		if req.state == stateDone {
			// we're done, break the *outer* loop.
			break
		}

		if err == io.EOF {
			if req.state != stateDone {
				return nil, io.ErrUnexpectedEOF
			}

			break
		}

		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 {
		return nil, 0, nil
	}

	parts := strings.Split(string(data[:idx]), " ")
	// panic guard
	if len(parts) != 3 {
		return nil, 0, fmt.Errorf("%w: expected 3 parts, got %d", ErrInvalidRequestFormat, len(parts))
	}

	method := parts[0]
	target := parts[1]
	versionRaw := parts[2]

	http, httpv, ok := strings.Cut(versionRaw, "/")

	if !ok || http != "HTTP" || (httpv != "1.1" && httpv != "1.0") {
		return nil, 0, fmt.Errorf("%w: expected 'HTTP/1.1', got '%s'", ErrUnsupportedHTTP, versionRaw)
	}

	reqLine := RequestLine{
		Method:        method,
		RequestTarget: target,
		HTTPVersion:   httpv,
	}

	return &reqLine, idx + 2, nil
}

func (r *Request) parse(data []byte) (int, error) {
	switch r.state {
	case stateRequestLine:

		reqLine, consumed, err := parseRequestLine(data)
		if err != nil {
			return 0, fmt.Errorf("failed to parse request line: %w", err)
		}

		if consumed == 0 {
			return 0, nil
		}

		r.RequestLine = *reqLine
		r.state = stateHeaders

		return consumed, nil
	case stateHeaders:

		consumed, done, err := r.Headers.Parse(data)
		if done {
			r.state = stateDone
			return consumed, nil
		}
		if err != nil {
			return 0, err
		}

		if consumed == 0 {
			return 0, nil
		}

		return consumed, nil

	case stateDone:
		return 0, nil

	default:

		return 0, errors.New("invalid parser state")
	}
}
