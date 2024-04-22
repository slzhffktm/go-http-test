package httptest

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sync"
)

// Server is a mock http server for testing.
type Server struct {
	httpServer *http.Server
	// calls store map[method][path]count
	calls map[string]map[string]int
	// routes store map[method][path]handler
	routes map[string]map[string]ServerHandlerFunc

	mu sync.Mutex
}

type Request struct {
	*http.Request
	Params Params
}

// ServerHandlerFunc is the interface of the handler function.
type ServerHandlerFunc func(w ResponseWriter, r *Request)

type ServerConfig struct {
	// Nothing here yet.
}

// NewServer creates and starts new http test server.
// address is the address to listen on, e.g. "localhost:3001".
func NewServer(address string, config ServerConfig) (*Server, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	httpServer := &http.Server{
		Addr:    address,
		Handler: http.NewServeMux(),
	}

	server := &Server{
		httpServer: httpServer,
		calls:      map[string]map[string]int{},
		routes:     map[string]map[string]ServerHandlerFunc{},
	}

	go func() {
		if err = httpServer.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	return server, nil
}

// Close closes the server.
func (s *Server) Close() error {
	return s.httpServer.Close()
}

// GetNCalls returns the number of calls for a path.
func (s *Server) GetNCalls(method, path string) int {
	calls, ok := s.calls[method][path]
	if !ok {
		return 0
	}

	return calls
}

// ResetNCalls resets the number of calls for all paths.
func (s *Server) ResetNCalls() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for path := range s.calls {
		for method := range s.calls[path] {
			s.calls[path][method] = 0
		}
	}
}

// RegisterHandler registers handler of a path.
// Registering same path twice will overwrite the previous handler.
func (s *Server) RegisterHandler(method string, path string, handler ServerHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.calls[method] == nil {
		s.calls[method] = map[string]int{}
	}
	if s.routes[method] == nil {
		s.routes[method] = map[string]ServerHandlerFunc{}
	}

	if s.routes[method] != nil && s.routes[method][path] != nil {
		// Regenerate the handler and re-register all handlers.
		serveMux := http.NewServeMux()
		for m, p := range s.routes {
			for k, v := range p {
				if m == method && k == path {
					continue
				}
				serveMux.HandleFunc(k, func(w http.ResponseWriter, r *http.Request) {
					s.IncrNCalls(m, k)
					v(ResponseWriter{w: w}, &Request{Request: r, Params: Params{request: r}})
				})
			}
		}
		s.httpServer.Handler = serveMux
	}
	s.routes[method][path] = handler

	serveMux := s.httpServer.Handler.(*http.ServeMux)
	serveMux.HandleFunc(s.combinePath(method, path), func(w http.ResponseWriter, r *http.Request) {
		s.IncrNCalls(method, path)
		handler(ResponseWriter{w: w}, &Request{Request: r, Params: Params{request: r}})
	})

	s.httpServer.Handler = serveMux
}

// IncrNCalls increments the number of calls for a path.
func (s *Server) IncrNCalls(method, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[method][path]++
}

// ResetAll resets all the calls and handlers.
func (s *Server) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.httpServer.Handler = http.NewServeMux()
	s.calls = map[string]map[string]int{}
	s.routes = map[string]map[string]ServerHandlerFunc{}
}

// combinePath combines method and path & converts the path.
// Example: combinePath(GET, /path/:param) -> GET /path/{param}
func (s *Server) combinePath(method, path string) string {
	return fmt.Sprintf("%s %s", method, s.convertPathParams(path))
}

// convertPathParams converts from "/path/:param" to "/path/{param}" to comply with the standard library.
func (s *Server) convertPathParams(input string) string {
	pattern := `:([^/]+)`
	re := regexp.MustCompile(pattern)
	converted := re.ReplaceAllString(input, "{$1}")

	return converted
}
