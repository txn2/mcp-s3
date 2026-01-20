package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResultTransformer allows modification of tool results after execution.
// Transformers can add metadata, filter content, or format output.
type ResultTransformer interface {
	// Name returns a unique identifier for this transformer.
	Name() string

	// Transform modifies the result and returns the transformed version.
	Transform(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error)
}

// ResultTransformerFunc is a function type that implements ResultTransformer.
type ResultTransformerFunc struct {
	name        string
	transformFn func(context.Context, *ToolContext, *mcp.CallToolResult) (*mcp.CallToolResult, error)
}

// NewResultTransformerFunc creates a new ResultTransformerFunc.
func NewResultTransformerFunc(name string, fn func(context.Context, *ToolContext, *mcp.CallToolResult) (*mcp.CallToolResult, error)) *ResultTransformerFunc {
	return &ResultTransformerFunc{
		name:        name,
		transformFn: fn,
	}
}

// Name returns the transformer name.
func (t *ResultTransformerFunc) Name() string {
	return t.name
}

// Transform calls the underlying function.
func (t *ResultTransformerFunc) Transform(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	return t.transformFn(ctx, tc, result)
}

// TransformerChain manages a collection of result transformers.
type TransformerChain struct {
	transformers []ResultTransformer
}

// NewTransformerChain creates a new transformer chain.
func NewTransformerChain() *TransformerChain {
	return &TransformerChain{
		transformers: make([]ResultTransformer, 0),
	}
}

// Add adds a transformer to the chain.
func (c *TransformerChain) Add(t ResultTransformer) {
	c.transformers = append(c.transformers, t)
}

// Transform applies all transformers in order.
func (c *TransformerChain) Transform(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	current := result

	for _, transformer := range c.transformers {
		transformed, err := transformer.Transform(ctx, tc, current)
		if err != nil {
			return nil, err
		}
		current = transformed
	}

	return current, nil
}

// All returns all registered transformers.
func (c *TransformerChain) All() []ResultTransformer {
	result := make([]ResultTransformer, len(c.transformers))
	copy(result, c.transformers)
	return result
}

// Clear removes all transformers from the chain.
func (c *TransformerChain) Clear() {
	c.transformers = make([]ResultTransformer, 0)
}
