package integration

import (
	"context"
	"io"
)

// ContentProvider provides access to S3 object content.
// This interface allows other MCP servers to access S3 content
// without directly depending on the S3 client.
type ContentProvider interface {
	// GetContent retrieves the content of an S3 object.
	GetContent(ctx context.Context, ref *ObjectReference) ([]byte, error)

	// GetContentStream retrieves the content as a stream for large objects.
	GetContentStream(ctx context.Context, ref *ObjectReference) (io.ReadCloser, error)

	// GetContentType retrieves only the content type without downloading.
	GetContentType(ctx context.Context, ref *ObjectReference) (string, error)

	// GetSize retrieves the object size in bytes.
	GetSize(ctx context.Context, ref *ObjectReference) (int64, error)
}

// ContentWriter provides write access to S3 objects.
type ContentWriter interface {
	// PutContent uploads content to an S3 object.
	PutContent(ctx context.Context, ref *ObjectReference, content []byte, contentType string) error

	// PutContentStream uploads content from a stream.
	PutContentStream(ctx context.Context, ref *ObjectReference, reader io.Reader, size int64, contentType string) error
}

// ContentProviderFunc is a function type for simple content providers.
type ContentProviderFunc func(ctx context.Context, ref *ObjectReference) ([]byte, error)

// SimpleContentProvider wraps a function as a ContentProvider.
type SimpleContentProvider struct {
	getContent     func(ctx context.Context, ref *ObjectReference) ([]byte, error)
	getContentType func(ctx context.Context, ref *ObjectReference) (string, error)
	getSize        func(ctx context.Context, ref *ObjectReference) (int64, error)
}

// NewSimpleContentProvider creates a simple content provider from functions.
func NewSimpleContentProvider(
	getContent func(ctx context.Context, ref *ObjectReference) ([]byte, error),
	getContentType func(ctx context.Context, ref *ObjectReference) (string, error),
	getSize func(ctx context.Context, ref *ObjectReference) (int64, error),
) *SimpleContentProvider {
	return &SimpleContentProvider{
		getContent:     getContent,
		getContentType: getContentType,
		getSize:        getSize,
	}
}

// GetContent implements ContentProvider.
func (p *SimpleContentProvider) GetContent(ctx context.Context, ref *ObjectReference) ([]byte, error) {
	return p.getContent(ctx, ref)
}

// GetContentStream implements ContentProvider by loading the full content.
func (p *SimpleContentProvider) GetContentStream(ctx context.Context, ref *ObjectReference) (io.ReadCloser, error) {
	content, err := p.getContent(ctx, ref)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(newBytesReader(content)), nil
}

// GetContentType implements ContentProvider.
func (p *SimpleContentProvider) GetContentType(ctx context.Context, ref *ObjectReference) (string, error) {
	if p.getContentType == nil {
		return "application/octet-stream", nil
	}
	return p.getContentType(ctx, ref)
}

// GetSize implements ContentProvider.
func (p *SimpleContentProvider) GetSize(ctx context.Context, ref *ObjectReference) (int64, error) {
	if p.getSize == nil {
		content, err := p.getContent(ctx, ref)
		if err != nil {
			return 0, err
		}
		return int64(len(content)), nil
	}
	return p.getSize(ctx, ref)
}

// bytesReader is a simple io.Reader that reads from a byte slice.
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// Ensure SimpleContentProvider implements ContentProvider.
var _ ContentProvider = (*SimpleContentProvider)(nil)

// CachedContentProvider wraps a ContentProvider with a simple cache.
type CachedContentProvider struct {
	inner ContentProvider
	cache map[string][]byte
}

// NewCachedContentProvider creates a cached content provider.
func NewCachedContentProvider(inner ContentProvider) *CachedContentProvider {
	return &CachedContentProvider{
		inner: inner,
		cache: make(map[string][]byte),
	}
}

// GetContent implements ContentProvider with caching.
func (p *CachedContentProvider) GetContent(ctx context.Context, ref *ObjectReference) ([]byte, error) {
	key := ref.String()
	if content, ok := p.cache[key]; ok {
		return content, nil
	}

	content, err := p.inner.GetContent(ctx, ref)
	if err != nil {
		return nil, err
	}

	p.cache[key] = content
	return content, nil
}

// GetContentStream implements ContentProvider.
func (p *CachedContentProvider) GetContentStream(ctx context.Context, ref *ObjectReference) (io.ReadCloser, error) {
	content, err := p.GetContent(ctx, ref)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(newBytesReader(content)), nil
}

// GetContentType implements ContentProvider.
func (p *CachedContentProvider) GetContentType(ctx context.Context, ref *ObjectReference) (string, error) {
	return p.inner.GetContentType(ctx, ref)
}

// GetSize implements ContentProvider.
func (p *CachedContentProvider) GetSize(ctx context.Context, ref *ObjectReference) (int64, error) {
	return p.inner.GetSize(ctx, ref)
}

// ClearCache clears the content cache.
func (p *CachedContentProvider) ClearCache() {
	p.cache = make(map[string][]byte)
}

// Ensure CachedContentProvider implements ContentProvider.
var _ ContentProvider = (*CachedContentProvider)(nil)
