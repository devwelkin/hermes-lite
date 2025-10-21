// request.go

package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
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
	Body        []byte
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

		// keep parsing the buffer until it's empty
		for {
			consumed, pErr := req.parse(accumulatedData)
			if pErr != nil {
				return nil, pErr
			}

			if consumed == 0 && req.state != stateDone {
				// not enough data in the buffer to parse a full line.
				// break the *inner* loop to read more data.
				break
			}

			accumulatedData = accumulatedData[consumed:]

			if req.state == stateDone {
				// break the *inner* loop.
				break
			}
		}

		if req.state == stateDone {
			// break the *outer* loop.
			break
		}

		if err == io.EOF {
			// If we hit EOF but we are not done parsing, it's an unexpected EOF.
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
		if err != nil {
			return 0, err
		}

		if consumed > 0 {
			if done {
				r.state = stateBody
			}
			return consumed, nil
		}

		return 0, nil

	case stateBody:
		value, ok := r.Headers["content-length"]
		if !ok {
			// No content-length
			r.state = stateDone
			return 0, nil
		}

		contentLength, err := strconv.Atoi(value)
		if err != nil {
			// A malformed content-length is a client error.
			return 0, fmt.Errorf("invalid content-length value: %q", value)
		}

		if contentLength == 0 {
			r.state = stateDone
			return 0, nil
		}

		// How much of the body we still need to read.
		needed := contentLength - len(r.Body)

		// How much we can actually read from the current data buffer.
		canRead := len(data)

		// We will consume the smaller of the two values.
		toConsume := needed
		if canRead < needed {
			toConsume = canRead
		}

		// Append the consumed part to the body.
		r.Body = append(r.Body, data[:toConsume]...)

		// If we've read the entire body, we're done.
		if len(r.Body) == contentLength {
			r.state = stateDone
		}

		// Return the number of bytes we actually consumed from the 'data' buffer.
		return toConsume, nil

	case stateDone:
		return 0, nil

	default:
		return 0, errors.New("invalid parser state")
	}
}
