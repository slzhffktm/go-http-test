package httptest

import (
	"github.com/gin-gonic/gin"
)

// Params is a struct to get the request path param.
// It is still here just to keep backwards compatibility.
type Params struct {
	ginContext *gin.Context
}

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (p Params) ByName(name string) string {
	return p.ginContext.Param(name)
}

func (p Params) All() map[string]string {
	params := make(map[string]string)

	for _, param := range p.ginContext.Params {
		params[param.Key] = param.Value
	}

	return params
}
