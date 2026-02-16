package integration

import (
	"context"
	"time"
)

// ObjectMetadata contains metadata about an S3 object.
type ObjectMetadata struct {
	// Reference is the object reference.
	Reference *ObjectReference

	// Size is the object size in bytes.
	Size int64

	// ContentType is the MIME type of the object.
	ContentType string

	// LastModified is when the object was last modified.
	LastModified time.Time

	// ETag is the entity tag (typically an MD5 hash).
	ETag string

	// CustomMetadata contains user-defined metadata.
	CustomMetadata map[string]string

	// StorageClass is the S3 storage class.
	StorageClass string

	// VersionID is the version ID (if versioning is enabled).
	VersionID string
}

// MetadataProvider provides access to S3 object metadata.
// This interface allows other MCP servers to access S3 metadata
// without directly depending on the S3 client.
type MetadataProvider interface {
	// GetMetadata retrieves metadata for an S3 object.
	GetMetadata(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error)

	// Exists checks if an object exists.
	Exists(ctx context.Context, ref *ObjectReference) (bool, error)

	// GetETag retrieves the ETag of an object.
	GetETag(ctx context.Context, ref *ObjectReference) (string, error)
}

// ListProvider provides listing capabilities.
type ListProvider interface {
	// ListObjects lists objects in a bucket with optional prefix.
	ListObjects(ctx context.Context, connection, bucket, prefix string, maxKeys int) ([]ObjectMetadata, error)

	// ListBuckets lists all accessible buckets.
	ListBuckets(ctx context.Context, connection string) ([]string, error)
}

// SimpleMetadataProvider wraps functions as a MetadataProvider.
type SimpleMetadataProvider struct {
	getMetadata func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error)
}

// NewSimpleMetadataProvider creates a simple metadata provider.
func NewSimpleMetadataProvider(
	getMetadata func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error),
) *SimpleMetadataProvider {
	return &SimpleMetadataProvider{
		getMetadata: getMetadata,
	}
}

// GetMetadata implements MetadataProvider.
func (p *SimpleMetadataProvider) GetMetadata(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
	return p.getMetadata(ctx, ref)
}

// Exists implements MetadataProvider.
func (p *SimpleMetadataProvider) Exists(ctx context.Context, ref *ObjectReference) (bool, error) {
	_, err := p.getMetadata(ctx, ref)
	if err != nil {
		return false, nil // Treat errors as not existing
	}
	return true, nil
}

// GetETag implements MetadataProvider.
func (p *SimpleMetadataProvider) GetETag(ctx context.Context, ref *ObjectReference) (string, error) {
	meta, err := p.getMetadata(ctx, ref)
	if err != nil {
		return "", err
	}
	return meta.ETag, nil
}

// Ensure SimpleMetadataProvider implements MetadataProvider.
var _ MetadataProvider = (*SimpleMetadataProvider)(nil)

// CachedMetadataProvider wraps a MetadataProvider with caching.
type CachedMetadataProvider struct {
	inner MetadataProvider
	cache map[string]*ObjectMetadata
	ttl   time.Duration
	times map[string]time.Time
}

// NewCachedMetadataProvider creates a cached metadata provider.
func NewCachedMetadataProvider(inner MetadataProvider, ttl time.Duration) *CachedMetadataProvider {
	return &CachedMetadataProvider{
		inner: inner,
		cache: make(map[string]*ObjectMetadata),
		ttl:   ttl,
		times: make(map[string]time.Time),
	}
}

// GetMetadata implements MetadataProvider with caching.
func (p *CachedMetadataProvider) GetMetadata(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
	key := ref.String()

	// Check cache
	if meta, ok := p.cache[key]; ok {
		if time.Since(p.times[key]) < p.ttl {
			return meta, nil
		}
		// Expired, remove from cache
		delete(p.cache, key)
		delete(p.times, key)
	}

	// Fetch from inner provider
	meta, err := p.inner.GetMetadata(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Cache the result
	p.cache[key] = meta
	p.times[key] = time.Now()

	return meta, nil
}

// Exists implements MetadataProvider.
func (p *CachedMetadataProvider) Exists(ctx context.Context, ref *ObjectReference) (bool, error) {
	_, err := p.GetMetadata(ctx, ref)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetETag implements MetadataProvider.
func (p *CachedMetadataProvider) GetETag(ctx context.Context, ref *ObjectReference) (string, error) {
	meta, err := p.GetMetadata(ctx, ref)
	if err != nil {
		return "", err
	}
	return meta.ETag, nil
}

// ClearCache clears the metadata cache.
func (p *CachedMetadataProvider) ClearCache() {
	p.cache = make(map[string]*ObjectMetadata)
	p.times = make(map[string]time.Time)
}

// Ensure CachedMetadataProvider implements MetadataProvider.
var _ MetadataProvider = (*CachedMetadataProvider)(nil)

// CompositeProvider combines multiple providers into one.
type CompositeProvider struct {
	ContentProvider
	MetadataProvider
	ListProvider
	ObjectResolver
}

// NewCompositeProvider creates a composite provider.
func NewCompositeProvider(
	content ContentProvider,
	metadata MetadataProvider,
	list ListProvider,
	resolver ObjectResolver,
) *CompositeProvider {
	return &CompositeProvider{
		ContentProvider:  content,
		MetadataProvider: metadata,
		ListProvider:     list,
		ObjectResolver:   resolver,
	}
}
