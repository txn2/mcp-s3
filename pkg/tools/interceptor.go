package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// InterceptResult represents the result of a request interception.
type InterceptResult struct {
	// Allow indicates whether the request should proceed.
	Allow bool

	// Reason provides an explanation if the request was blocked.
	Reason string

	// ModifiedRequest is an optional modified version of the request.
	// If nil, the original request is used.
	ModifiedRequest *mcp.CallToolRequest
}

// Allowed returns an InterceptResult that allows the request to proceed.
func Allowed() InterceptResult {
	return InterceptResult{Allow: true}
}

// AllowedWithModification returns an InterceptResult that allows with a modified request.
func AllowedWithModification(req *mcp.CallToolRequest) InterceptResult {
	return InterceptResult{
		Allow:           true,
		ModifiedRequest: req,
	}
}

// Blocked returns an InterceptResult that blocks the request.
func Blocked(reason string) InterceptResult {
	return InterceptResult{
		Allow:  false,
		Reason: reason,
	}
}

// RequestInterceptor allows inspection and modification of requests before execution.
// Unlike middleware, interceptors focus on request validation and access control.
type RequestInterceptor interface {
	// Name returns a unique identifier for this interceptor.
	Name() string

	// Intercept examines the request and returns an InterceptResult.
	// Called before the tool handler executes.
	Intercept(ctx context.Context, tc *ToolContext, request *mcp.CallToolRequest) InterceptResult
}

// RequestInterceptorFunc is a function type that implements RequestInterceptor.
type RequestInterceptorFunc struct {
	name        string
	interceptFn func(context.Context, *ToolContext, *mcp.CallToolRequest) InterceptResult
}

// NewRequestInterceptorFunc creates a new RequestInterceptorFunc.
func NewRequestInterceptorFunc(name string, fn func(context.Context, *ToolContext, *mcp.CallToolRequest) InterceptResult) *RequestInterceptorFunc {
	return &RequestInterceptorFunc{
		name:        name,
		interceptFn: fn,
	}
}

// Name returns the interceptor name.
func (i *RequestInterceptorFunc) Name() string {
	return i.name
}

// Intercept calls the underlying function.
func (i *RequestInterceptorFunc) Intercept(ctx context.Context, tc *ToolContext, request *mcp.CallToolRequest) InterceptResult {
	return i.interceptFn(ctx, tc, request)
}

// InterceptorChain manages a collection of request interceptors.
type InterceptorChain struct {
	interceptors []RequestInterceptor
}

// NewInterceptorChain creates a new interceptor chain.
func NewInterceptorChain() *InterceptorChain {
	return &InterceptorChain{
		interceptors: make([]RequestInterceptor, 0),
	}
}

// Add adds an interceptor to the chain.
func (c *InterceptorChain) Add(i RequestInterceptor) {
	c.interceptors = append(c.interceptors, i)
}

// Intercept runs all interceptors in order.
// Returns the first blocking result, or allows if all pass.
func (c *InterceptorChain) Intercept(ctx context.Context, tc *ToolContext, request *mcp.CallToolRequest) InterceptResult {
	currentReq := request

	for _, interceptor := range c.interceptors {
		result := interceptor.Intercept(ctx, tc, currentReq)
		if !result.Allow {
			return result
		}
		// Apply any modifications
		if result.ModifiedRequest != nil {
			currentReq = result.ModifiedRequest
		}
	}

	// All interceptors passed - check if request was modified
	if currentReq != request {
		return AllowedWithModification(currentReq)
	}

	return Allowed()
}

// All returns all registered interceptors.
func (c *InterceptorChain) All() []RequestInterceptor {
	result := make([]RequestInterceptor, len(c.interceptors))
	copy(result, c.interceptors)
	return result
}

// Clear removes all interceptors from the chain.
func (c *InterceptorChain) Clear() {
	c.interceptors = make([]RequestInterceptor, 0)
}
