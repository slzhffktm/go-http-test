package httptest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
)

// Server is a mock http server for testing.
type Server struct {
	httpServer *http.Server
	engine     *gin.Engine
	// nCalls store map[method][path]count
	nCalls map[string]map[string]int
	// routes store map[method][path]handler
	routes map[string]map[string]ServerHandlerFunc
	calls  map[string]map[string][]RequestMade

	mu sync.Mutex
}

type Request struct {
	*http.Request
	Params Params
}

type RequestMade struct {
	Body    []byte
	Headers http.Header
	Query   url.Values
	Params  map[string]string
}

// ServerHandlerFunc is the interface of the handler function.
type ServerHandlerFunc func(w ResponseWriter, r *Request)

type ServerConfig struct {
	// Nothing here yet.
}

// NewServer creates and starts new http test server.
// address is the address to listen on, e.g. "localhost:3001".
func NewServer(address string, config ServerConfig) (*Server, error) {
	// Set gin to release mode to avoid unnecessary logs.
	gin.SetMode(gin.ReleaseMode)

	// Start listener first to make sure the address is available.
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}

	r := gin.Default()

	httpServer := &http.Server{
		Addr:    address,
		Handler: r.Handler(),
	}

	server := &Server{
		engine:     r,
		httpServer: httpServer,
		nCalls:     map[string]map[string]int{},
		routes:     map[string]map[string]ServerHandlerFunc{},
		calls:      map[string]map[string][]RequestMade{},
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

// GetNCalls returns the number of nCalls for a path.
func (s *Server) GetNCalls(method, path string) int {
	calls, ok := s.nCalls[method][path]
	if !ok {
		return 0
	}

	return calls
}

// ResetNCalls resets the number of nCalls for all paths.
func (s *Server) ResetNCalls() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for path := range s.nCalls {
		for method := range s.nCalls[path] {
			s.nCalls[path][method] = 0
		}
	}
}

// GetCalls returns the calls for a path.
func (s *Server) GetCalls(method, path string) []RequestMade {
	return s.calls[method][path]
}

// ResetCalls resets the calls & nCalls for all paths.
// It does not reset the handlers.
func (s *Server) ResetCalls() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ResetNCalls()
	for path := range s.calls {
		for method := range s.calls[path] {
			s.calls[path][method] = []RequestMade{}
		}
	}
}

// RegisterHandler registers handler of a path.
// Registering same path twice will overwrite the previous handler.
func (s *Server) RegisterHandler(method string, path string, handler ServerHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.nCalls[method] == nil {
		s.nCalls[method] = map[string]int{}
	}
	if s.routes[method] == nil {
		s.routes[method] = map[string]ServerHandlerFunc{}
	}
	if s.calls[method] == nil {
		s.calls[method] = map[string][]RequestMade{}
	}

	if s.routes[method] != nil && s.routes[method][path] != nil {
		// Regenerate the handler and re-register all handlers.
		s.engine = gin.Default()
		for m, p := range s.routes {
			for k, v := range p {
				// Skip same one, we'll register it below.
				if m == method && k == path {
					continue
				}
				s.engine.Handle(m, k, func(c *gin.Context) {
					s.incrNCalls(m, k)
					s.storeCall(m, k, c)
					v(ResponseWriter{w: c.Writer}, &Request{Request: c.Request, Params: Params{ginContext: c}})
				})
			}
		}
		s.httpServer.Handler = s.engine.Handler()
	}
	s.routes[method][path] = handler

	s.engine.Handle(method, path, func(c *gin.Context) {
		s.incrNCalls(method, path)
		s.storeCall(method, path, c)
		handler(ResponseWriter{w: c.Writer}, &Request{Request: c.Request, Params: Params{ginContext: c}})
	})

	s.httpServer.Handler = s.engine.Handler()
}

// incrNCalls increments the number of nCalls for a path.
func (s *Server) incrNCalls(method, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nCalls[method][path]++
}

// storeCall stores the call for a path.
func (s *Server) storeCall(method, path string, c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If body is not empty, read it into byte.
	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
		// Restore the body.
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	s.calls[method][path] = append(s.calls[method][path], RequestMade{
		Body:    body,
		Headers: c.Request.Header,
		Query:   c.Request.URL.Query(),
		Params:  s.getAllParams(c),
	})
}

func (s *Server) getAllParams(c *gin.Context) map[string]string {
	params := make(map[string]string)

	for _, param := range c.Params {
		params[param.Key] = param.Value
	}

	return params
}

// ResetAll resets all the nCalls, handlers, and calls.
func (s *Server) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.engine = gin.Default()
	s.httpServer.Handler = s.engine.Handler()
	s.nCalls = map[string]map[string]int{}
	s.routes = map[string]map[string]ServerHandlerFunc{}
	s.calls = map[string]map[string][]RequestMade{}
}
