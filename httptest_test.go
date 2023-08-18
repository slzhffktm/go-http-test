package httptest_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	httptest "github.com/slzhffktm/go-http-test"
)

const baseURL = "http://127.0.0.1:3001"
const address = "127.0.0.1:3001"

var (
	ctx = context.Background()
)

type serverTestSuite struct {
	suite.Suite
	httpClient *HttpClient
}

func TestServerTestSuite(t *testing.T) {
	httpClient := http.DefaultClient
	httpClient.Timeout = 500 * time.Millisecond
	suite.Run(t, &serverTestSuite{
		httpClient: NewHttpClient(baseURL, httpClient),
	})
}

func (s *serverTestSuite) TestNotRegisteredPath() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	res, _, err := s.httpClient.Do(
		ctx,
		http.MethodGet,
		"/unknown",
		nil,
		nil,
		nil,
	)
	s.Equal(http.StatusNotFound, res.StatusCode)
}

func (s *serverTestSuite) TestPost_ResponseJSON() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path"
	expectedReqBody := []byte(`{"test":"osterone"}`)
	expectedResBody := []byte(`{"abcd":"abcd","efgh":1}`)

	type resStruct struct {
		Abcd string `json:"abcd"`
		Efgh int    `json:"efgh"`
	}

	server.RegisterHandler(http.MethodPost, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal(path, r.URL.Path)

		reqBody, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(expectedReqBody, reqBody)

		w.SetStatusCode(http.StatusOK)
		w.SetBodyJSON(resStruct{
			Abcd: "abcd",
			Efgh: 1,
		})
	})

	res, resBody, err := s.httpClient.Do(
		ctx,
		http.MethodPost,
		path,
		map[string]string{
			"Content-Type": "application/json",
		},
		expectedReqBody,
		nil,
	)
	s.NoError(err)

	s.Equal(200, res.StatusCode)
	s.Equal(expectedResBody, resBody)

	s.Equal(1, server.GetNCalls(http.MethodPost, path))
}

func (s *serverTestSuite) TestGet_WithQueryParams() {
	// Should only register the path without query params.
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(http.MethodGet, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		s.Equal("param", r.FormValue("query"))
		s.Equal(http.MethodGet, r.Method)
		s.Equal(path, r.URL.Path)

		w.SetStatusCode(http.StatusOK)
		w.SetBodyBytes(expectedResBody)
	})

	qParams := url.Values{}
	qParams.Add("query", "param")
	qParams.Add("test", "osterone")

	res, resBody, err := s.httpClient.Do(
		ctx,
		http.MethodGet,
		path,
		nil,
		nil,
		qParams,
	)
	s.NoError(err)

	s.Equal(200, res.StatusCode)
	s.Equal(expectedResBody, resBody)

	s.Equal(1, server.GetNCalls(http.MethodGet, path))
}

func (s *serverTestSuite) TestGet_WithPathParam() {
	// Should only register the path without query params.
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path/:id/subpath/:subid"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(http.MethodGet, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		s.Equal("1d", r.Params.ByName("id"))
		s.Equal("5ub1d", r.Params.ByName("subid"))
		s.Equal(http.MethodGet, r.Method)

		w.SetStatusCode(http.StatusOK)
		w.SetBodyBytes(expectedResBody)
	})

	reqPath := "/some-path/1d/subpath/5ub1d"

	res, resBody, err := s.httpClient.Do(
		ctx,
		http.MethodGet,
		reqPath,
		nil,
		nil,
		nil,
	)
	s.NoError(err)

	s.Equal(200, res.StatusCode)
	s.Equal(expectedResBody, resBody)

	s.Equal(1, server.GetNCalls(http.MethodGet, path))
}

func (s *serverTestSuite) TestGet_MultipleTimes_ResetNCalls() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(http.MethodGet, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		w.SetStatusCode(http.StatusOK)
		w.SetBodyBytes(expectedResBody)
	})

	qParams := url.Values{}
	qParams.Add("query", "param")
	qParams.Add("test", "osterone")

	for i := 0; i < 10; i++ {
		res, resBody, err := s.httpClient.Do(
			ctx,
			http.MethodGet,
			path,
			nil,
			nil,
			qParams,
		)
		s.NoError(err)
		s.Equal(200, res.StatusCode)
		s.Equal(expectedResBody, resBody)
	}

	s.Equal(10, server.GetNCalls(http.MethodGet, path))
	server.ResetNCalls()
	s.Equal(0, server.GetNCalls(http.MethodGet, path))
}

func (s *serverTestSuite) TestReregisterHandler_ShouldOverwrite() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path/subpath"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(http.MethodGet, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		w.SetStatusCode(http.StatusOK)
		w.SetBodyBytes(expectedResBody)
	})

	server.RegisterHandler(http.MethodGet, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		w.Header().Set("some", "header")
		w.SetStatusCode(http.StatusNotFound)
	})

	res, _, err := s.httpClient.Do(
		ctx,
		http.MethodGet,
		path,
		nil,
		nil,
		nil,
	)
	s.NoError(err)
	s.Equal(404, res.StatusCode)
	s.Equal("header", res.Header.Get("some"))

	s.Equal(1, server.GetNCalls(http.MethodGet, path))
}

func (s *serverTestSuite) TestSimulateTimeout() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(http.MethodGet, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		time.Sleep(1 * time.Second)
		w.SetStatusCode(http.StatusOK)
		w.SetBodyBytes(expectedResBody)
	})

	_, _, err = s.httpClient.Do(
		ctx,
		http.MethodGet,
		path,
		nil,
		nil,
		nil,
	)
	s.Error(err)
	s.True(os.IsTimeout(err))

	s.Equal(1, server.GetNCalls(http.MethodGet, path))
}

func (s *serverTestSuite) TestResetAll() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path/subpath"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(http.MethodGet, path, func(w httptest.ResponseWriter, r *httptest.Request) {
		w.SetStatusCode(http.StatusOK)
		w.SetBodyBytes(expectedResBody)
	})

	server.ResetAll()

	res, _, err := s.httpClient.Do(
		ctx,
		http.MethodGet,
		path,
		nil,
		nil,
		nil,
	)
	s.NoError(err)
	s.Equal(http.StatusNotFound, res.StatusCode)

	s.Equal(0, server.GetNCalls(http.MethodGet, path))
}
