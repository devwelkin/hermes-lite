package response

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/devwelkin/hermes-lite/internal/headers"
)

// StatusCode and related consts remain the same.
type StatusCode int

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

type writerState int

const (
	stateStatus  writerState = iota // can write status
	stateHeaders                    // can write headers
	stateBody                       // can write body
)

// Writer is a stateful writer for constructing an http response.
type Writer struct {
	w     io.Writer   // connection
	state writerState // state machine
}

// NewWriter creates a new response Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:     w,
		state: stateStatus,
	}
}

// WriteStatusLine writes the status line. can only be called once, and first.
func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != stateStatus {
		return errors.New("WriteStatusLine called in wrong state")
	}
	reason, ok := reasonPhrases[statusCode]
	if !ok {
		reason = ""
	}
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, reason)

	if _, err := w.w.Write([]byte(statusLine)); err != nil {
		return err
	}
	w.state = stateHeaders
	return nil
}

// WriteHeaders writes the headers. must be called after status and before body.
func (w *Writer) WriteHeaders(h headers.Headers) error {
	if w.state != stateHeaders {
		return errors.New("WriteHeaders called in wrong state")
	}

	for key, val := range h {
		line := fmt.Sprintf("%s: %s\r\n", key, val)
		if _, err := w.w.Write([]byte(line)); err != nil {
			return err
		}
	}

	// final crlf to separate headers from body
	if _, err := w.w.Write([]byte("\r\n")); err != nil {
		return err
	}

	w.state = stateBody
	return nil
}

// WriteBody writes to the response body. can be called multiple times, but
// only after headers have been written.
func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state < stateHeaders {
		return 0, errors.New("WriteBody called before headers")
	}
	if w.state == stateHeaders {
		if _, err := w.w.Write([]byte("\r\n")); err != nil {
			return 0, err
		}
		w.state = stateBody
	}
	return w.w.Write(p)
}

// GetDefaultHeaders is still a useful helper for the handler.
func GetDefaultHeaders(contentLen int) headers.Headers {
	return headers.Headers{
		"Content-Type":   "text/plain",
		"Connection":     "close",
		"Content-Length": strconv.Itoa(contentLen),
	}
}
