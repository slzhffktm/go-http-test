package httptest

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
)

// Server is a mock http server for testing.
type Server struct {
	server  *httptest.Server
	handler map[string]ServerHandlerFunc
	calls   map[string]int
}

type ServerHandlerFunc func(w ResponseWriter, r *http.Request)

type ServerConfig struct {
	EnableHTTP2 bool
	UseTLS      bool
}

// NewServer creates and starts new http test server.
// address is the address to listen on, e.g. "localhost:3001".
func NewServer(address string, config ServerConfig) (*Server, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	server := &Server{
		handler: map[string]ServerHandlerFunc{},
		calls:   map[string]int{},
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.calls[r.URL.Path]++
		server.handler[r.URL.Path](ResponseWriter{w}, r)
	}))

	// httptest.NewUnstartedServer creates a listener.
	// Close that listener and replace with the one we created.
	if err = ts.Listener.Close(); err != nil {
		return nil, fmt.Errorf("default listener.Close(): %w", err)
	}
	ts.Listener = l

	if config.EnableHTTP2 {
		ts.EnableHTTP2 = true
	}

	if config.UseTLS {
		ts.StartTLS()
	} else {
		ts.Start()
	}

	server.server = ts

	return server, nil
}

// Close closes the server.
func (s *Server) Close() {
	s.server.Close()
}

// GetNCalls returns the number of calls for a path.
func (s *Server) GetNCalls(path string) int {
	return s.calls[path]
}

// ResetNCalls resets the number of calls for all paths.
func (s *Server) ResetNCalls() {
	for k := range s.calls {
		s.calls[k] = 0
	}
}

// RegisterHandler registers handler of a path.
// Registering same path twice will overwrite the previous handler.
func (s *Server) RegisterHandler(path string, handler ServerHandlerFunc) {
	s.handler[path] = handler
}

// RemoveHandler removes registered handler of a path.
func (s *Server) RemoveHandler(path string) {
	delete(s.handler, path)
}
