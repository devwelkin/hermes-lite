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
	stateStatus   writerState = iota // can write status
	stateHeaders                     // can write headers
	stateBody                        // can write body
	stateTrailers                    // can write trailers
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
	if w.state != stateBody {
		return 0, errors.New("WriteBody called in wrong state")
	}
	return w.w.Write(p)
}

// WriteChunkedBody writes a chunk of data for a chunked response.
// It writes the chunk size in hex, followed by the data, and a CRLF.
func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.state != stateBody {
		return 0, errors.New("WriteChunkedBody called in wrong state")
	}

	// Don't write empty chunks unless it's the final one.
	if len(p) == 0 {
		return 0, nil
	}

	// Format: <chunk size in hex>\r\n<chunk data>\r\n
	chunkHeader := fmt.Sprintf("%x\r\n", len(p))
	chunkTrailer := "\r\n"

	totalWritten := 0

	n, err := w.w.Write([]byte(chunkHeader))
	totalWritten += n
	if err != nil {
		return totalWritten, err
	}

	n, err = w.w.Write(p)
	totalWritten += n
	if err != nil {
		return totalWritten, err
	}

	n, err = w.w.Write([]byte(chunkTrailer))
	totalWritten += n
	if err != nil {
		return totalWritten, err
	}

	return totalWritten, nil
}

// WriteChunkedBodyDone writes the zero-length chunk to signal the end
// of a chunked response body, and prepares for writing trailers.
func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.state != stateBody {
		return 0, errors.New("WriteChunkedBodyDone called in wrong state")
	}

	n, err := w.w.Write([]byte("0\r\n"))
	w.state = stateTrailers
	return n, err
}

// WriteTrailers writes the trailers. Must be called after WriteChunkedBodyDone.
func (w *Writer) WriteTrailers(h headers.Headers) error {
	if w.state != stateTrailers {
		return errors.New("WriteTrailers called in wrong state")
	}

	for key, val := range h {
		line := fmt.Sprintf("%s: %s\r\n", key, val)
		if _, err := w.w.Write([]byte(line)); err != nil {
			return err
		}
	}

	// final crlf to terminate the response
	_, err := w.w.Write([]byte("\r\n"))
	return err
}

// GetDefaultHeaders is still a useful helper for the handler.
func GetDefaultHeaders(contentLen int) headers.Headers {
	return headers.Headers{
		"Content-Type":   "text/plain",
		"Connection":     "close",
		"Content-Length": strconv.Itoa(contentLen),
	}
}
