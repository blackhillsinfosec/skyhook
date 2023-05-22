package server

import (
	"bytes"
	"net/http"
)

// BuffResponseWriter is a custom structure to allow us to overwrite buffers before sending.
type BuffResponseWriter struct {
	Buff *bytes.Buffer
}

// Header returns an empty http.Header map
func (brw BuffResponseWriter) Header() http.Header {
	return http.Header{}
}

// Write uses the default Buff Write function and returns the result
func (brw BuffResponseWriter) Write(b []byte) (int, error) {
	return brw.Buff.Write(b)
}

// WriteHeader accepts a status code and does nothing
func (brw BuffResponseWriter) WriteHeader(statusCode int) {}
