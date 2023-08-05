package httptest

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ResponseWriter is a struct that handles the response writing.
type ResponseWriter struct {
	w http.ResponseWriter
}

// SetBodyBytes sets the response body.
func (r *ResponseWriter) SetBodyBytes(b []byte) (int, error) {
	return r.w.Write(b)
}

// SetBodyJSON marshals s to JSON and sets it as the response body, and
// sets the Content-Type header to application/json.
func (r *ResponseWriter) SetBodyJSON(s any) (int, error) {
	r.w.Header().Set("Content-Type", "application/json")

	b, err := json.Marshal(&s)
	if err != nil {
		return 0, fmt.Errorf("json.Marshal: %w", err)
	}

	return r.w.Write(b)
}

// SetStatusCode sets the response status code.
func (r *ResponseWriter) SetStatusCode(statusCode int) {
	r.w.WriteHeader(statusCode)
}

// Header returns the response headers.
func (r *ResponseWriter) Header() http.Header {
	return r.w.Header()
}
