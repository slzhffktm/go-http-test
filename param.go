package httptest

import "net/http"

// Params is a struct to get the request path param.
// It is still here just to keep backwards compatibility.
type Params struct {
	request *http.Request
}

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (p Params) ByName(name string) string {
	return p.request.PathValue(name)
}
