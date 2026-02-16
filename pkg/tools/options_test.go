package tools

import (
	"context"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWithMiddleware(t *testing.T) {
	tk := &Toolkit{
		middleware: NewMiddlewareChain(),
	}

	called := false
	mw := NewMiddlewareFunc("test",
		func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			called = true
			return ctx, nil
		},
		func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			return result, err
		},
	)

	opt := WithMiddleware(mw)
	opt(tk)

	if len(tk.middleware.All()) != 1 {
		t.Error("middleware not added")
	}

	// Verify middleware works
	tc := NewToolContext(ToolListBuckets, "")
	_, _ = tk.middleware.Before(context.Background(), tc)

	if !called {
		t.Error("middleware was not invoked")
	}
}

func TestWithInterceptor(t *testing.T) {
	tk := &Toolkit{
		interceptors: NewInterceptorChain(),
	}

	interceptor := NewRequestInterceptorFunc("test", func(ctx context.Context, tc *ToolContext, req *mcp.CallToolRequest) InterceptResult {
		return Blocked("blocked")
	})

	opt := WithInterceptor(interceptor)
	opt(tk)

	if len(tk.interceptors.All()) != 1 {
		t.Error("interceptor not added")
	}
}

func TestWithTransformer(t *testing.T) {
	tk := &Toolkit{
		transformers: NewTransformerChain(),
	}

	transformer := NewResultTransformerFunc("test", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		return result, nil
	})

	opt := WithTransformer(transformer)
	opt(tk)

	if len(tk.transformers.All()) != 1 {
		t.Error("transformer not added")
	}
}

func TestWithLogger(t *testing.T) {
	tk := &Toolkit{}

	logger := slog.Default()
	opt := WithLogger(logger)
	opt(tk)

	if tk.logger != logger {
		t.Error("logger not set")
	}
}

func TestWithClientProvider(t *testing.T) {
	tk := &Toolkit{}

	provider := func(name string) (S3Client, error) {
		return nil, nil
	}

	opt := WithClientProvider(provider)
	opt(tk)

	if tk.clientProvider == nil {
		t.Error("client provider not set")
	}
}

func TestWithDescriptions(t *testing.T) {
	tk := &Toolkit{}

	descs := map[ToolName]string{
		ToolListBuckets: "custom list buckets",
		ToolGetObject:   "custom get object",
	}

	opt := WithDescriptions(descs)
	opt(tk)

	if len(tk.descriptions) != 2 {
		t.Errorf("expected 2 descriptions, got %d", len(tk.descriptions))
	}
	if tk.descriptions[ToolListBuckets] != "custom list buckets" {
		t.Errorf("unexpected description: %s", tk.descriptions[ToolListBuckets])
	}

	// Verify it's a copy, not a reference.
	descs[ToolListBuckets] = "modified"
	if tk.descriptions[ToolListBuckets] == "modified" {
		t.Error("descriptions map should be copied, not referenced")
	}
}

func TestWithDescription(t *testing.T) {
	cfg := &toolConfig{}

	opt := WithDescription("per-tool override")
	opt(cfg)

	if cfg.description == nil {
		t.Fatal("description not set")
	}
	if *cfg.description != "per-tool override" {
		t.Errorf("unexpected description: %s", *cfg.description)
	}
}

func TestWithAnnotations(t *testing.T) {
	tk := &Toolkit{}

	anns := map[ToolName]*mcp.ToolAnnotations{
		ToolListBuckets: {ReadOnlyHint: true},
		ToolPutObject:   {DestructiveHint: boolPtr(false)},
	}

	opt := WithAnnotations(anns)
	opt(tk)

	if len(tk.annotations) != 2 {
		t.Errorf("expected 2 annotations, got %d", len(tk.annotations))
	}
	if !tk.annotations[ToolListBuckets].ReadOnlyHint {
		t.Error("expected ReadOnlyHint to be true for ToolListBuckets")
	}

	// Verify it's a copy, not a reference.
	anns[ToolListBuckets] = nil
	if tk.annotations[ToolListBuckets] == nil {
		t.Error("annotations map should be copied, not referenced")
	}
}

func TestWithAnnotation(t *testing.T) {
	cfg := &toolConfig{}

	ann := &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
	}

	opt := WithAnnotation(ann)
	opt(cfg)

	if cfg.annotations == nil {
		t.Fatal("annotations not set")
	}
	if !cfg.annotations.ReadOnlyHint {
		t.Error("expected ReadOnlyHint to be true")
	}
	if !cfg.annotations.IdempotentHint {
		t.Error("expected IdempotentHint to be true")
	}
}
