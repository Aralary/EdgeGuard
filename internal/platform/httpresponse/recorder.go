package httpresponse

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// Recorder wraps an HTTP response writer and records the response status and size.
// Unwrap allows Echo and other middleware to reach the original response writer.
type Recorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func NewRecorder(writer http.ResponseWriter) *Recorder {
	return &Recorder{
		ResponseWriter: writer,
		statusCode:     http.StatusOK,
	}
}

func (r *Recorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.statusCode = statusCode
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *Recorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	size, err := r.ResponseWriter.Write(data)
	r.bytesWritten += size
	return size, err
}

func (r *Recorder) StatusCode() int {
	return r.statusCode
}

func (r *Recorder) BytesWritten() int {
	return r.bytesWritten
}

func (r *Recorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *Recorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *Recorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("underlying response writer does not support hijacking")
	}
	return hijacker.Hijack()
}
