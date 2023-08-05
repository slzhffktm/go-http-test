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

const baseURL = "http://localhost:3001"
const address = "localhost:3001"

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

	_, _, err = s.httpClient.Do(
		ctx,
		http.MethodGet,
		"/unknown",
		nil,
		nil,
		nil,
	)
	s.Error(err)
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

	server.RegisterHandler(path, http.MethodPost, func(w httptest.ResponseWriter, r *http.Request) {
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

	s.Equal(1, server.GetNCalls(path, http.MethodPost))
}

func (s *serverTestSuite) TestGet_WithQueryParams() {
	// Should only register the path without query params.
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(path, http.MethodGet, func(w httptest.ResponseWriter, r *http.Request) {
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

	s.Equal(1, server.GetNCalls(path, http.MethodGet))
}

func (s *serverTestSuite) TestGet_MultipleTimes_ResetNCalls() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(path, http.MethodGet, func(w httptest.ResponseWriter, r *http.Request) {
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

	s.Equal(10, server.GetNCalls(path, http.MethodGet))
	server.ResetNCalls()
	s.Equal(0, server.GetNCalls(path, http.MethodGet))
}

func (s *serverTestSuite) TestReregisterHandler_ShouldOverwrite() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path/subpath"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(path, http.MethodGet, func(w httptest.ResponseWriter, r *http.Request) {
		w.SetStatusCode(http.StatusOK)
		w.SetBodyBytes(expectedResBody)
	})

	server.RegisterHandler(path, http.MethodGet, func(w httptest.ResponseWriter, r *http.Request) {
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

	s.Equal(1, server.GetNCalls(path, http.MethodGet))
}

func (s *serverTestSuite) TestSimulateTimeout() {
	server, err := httptest.NewServer(address, httptest.ServerConfig{})
	s.NoError(err)
	defer server.Close()

	path := "/some-path"
	expectedResBody := []byte(`{"res":"ponse"}`)

	server.RegisterHandler(path, http.MethodGet, func(w httptest.ResponseWriter, r *http.Request) {
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

	s.Equal(1, server.GetNCalls(path, http.MethodGet))
}
