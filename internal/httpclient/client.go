package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type HttpClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func New(baseUrl string, client *http.Client) *HttpClient {
	u, err := url.Parse(baseUrl)
	if err != nil {
		panic(err)
	}

	if client == nil {
		client = http.DefaultClient
	}
	return &HttpClient{baseURL: u, httpClient: client}
}

func (c *HttpClient) Do(
	ctx context.Context,
	method string,
	path string,
	headers map[string]string,
	reqBody []byte,
	queryParameters url.Values,
) (*http.Response, []byte, error) {
	requestUrl, err := c.buildURL(path)
	if err != nil {
		return nil, nil, err
	}

	if queryParameters != nil {
		requestUrl.RawQuery = queryParameters.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, requestUrl.String(), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	httpRes, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer httpRes.Body.Close()

	respBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return nil, nil, err
	}

	return httpRes, respBody, nil
}

func (c *HttpClient) buildURL(path string) (*url.URL, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("url.Parse: %s", err)
	}

	return c.baseURL.ResolveReference(rel), nil
}
