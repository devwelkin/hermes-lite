package response

import (
	"fmt"
	"io"
	"strconv"

	"github.com/devwelkin/hermes-lite/internal/headers"
)

// StatusCode is our "enum" for http status codes.
type StatusCode int

// our supported status codes
const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

var reasonPhrases = map[StatusCode]string{
	StatusOK:                  "OK",
	StatusBadRequest:          "Bad Request",
	StatusInternalServerError: "Internal Server Error",
}

// WriteStatusLine writes the http status line (e.g., HTTP/1.1 200 OK)
func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	reason, ok := reasonPhrases[statusCode]
	if !ok {
		// rfc 9112: "a server must send the space that separates the
		// status-code from the reason-phrase even when the
		// reason-phrase is absent"
		reason = ""
	}

	// e.g., "HTTP/1.1 200 OK\r\n"
	// or "HTTP/1.1 404 \r\n" if 404 wasn't in our map
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, reason)

	_, err := w.Write([]byte(statusLine))
	return err
}

// GetDefaultHeaders returns the headers we (almost) always want.
func GetDefaultHeaders(contentLen int) headers.Headers {
	return headers.Headers{
		// prompt says text/plain, no charset.
		"Content-Type":   "text/plain",
		"Connection":     "close", // we don't do keep-alive yet
		"Content-Length": strconv.Itoa(contentLen),
	}
}

// WriteHeaders writes all headers to the writer, followed by the
// terminating crlf.
func WriteHeaders(w io.Writer, h headers.Headers) error {
	// write each header line
	for key, val := range h {
		line := fmt.Sprintf("%s: %s\r\n", key, val)
		if _, err := w.Write([]byte(line)); err != nil {
			return err
		}
	}

	// write the final crlf to separate headers from body
	_, err := w.Write([]byte("\r\n"))
	return err
}
