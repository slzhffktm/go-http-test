package httptest

import (
	"errors"
	"net"
	"net/http"

	"github.com/slzhffktm/httprouter"
)

// Server is a mock http server for testing.
type Server struct {
	httpServer *http.Server
	// calls store map[method][path]count
	calls  map[string]map[string]int
	router *httprouter.Router
}

type Request struct {
	*http.Request
	Params httprouter.Params
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

	router := httprouter.New()

	httpServer := &http.Server{
		Addr:    address,
		Handler: router,
	}

	server := &Server{
		httpServer: httpServer,
		calls:      map[string]map[string]int{},
		router:     router,
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
	for path := range s.calls {
		for method := range s.calls[path] {
			s.calls[path][method] = 0
		}
	}
}

// RegisterHandler registers handler of a path.
// Registering same path twice will overwrite the previous handler.
func (s *Server) RegisterHandler(method string, path string, handler ServerHandlerFunc) {
	if s.calls[method] == nil {
		s.calls[method] = map[string]int{}
	}

	s.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		s.calls[method][path]++
		handler(ResponseWriter{w: w}, &Request{
			Request: r,
			Params:  params,
		})
	})
}

// ResetAll resets all the calls and handlers.
func (s *Server) ResetAll() {
	s.router.ClearHandlers()
	s.calls = map[string]map[string]int{}
}
