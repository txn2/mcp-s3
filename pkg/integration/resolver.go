// Package integration provides interfaces for cross-MCP server composition.
package integration

import (
	"context"
	"fmt"
	"regexp"
)

// ObjectReference represents a reference to an S3 object.
type ObjectReference struct {
	// Connection is the name of the S3 connection (empty for default).
	Connection string

	// Bucket is the S3 bucket name.
	Bucket string

	// Key is the object key (path).
	Key string
}

// String returns the object reference as a URI string.
func (r ObjectReference) String() string {
	if r.Connection != "" {
		return fmt.Sprintf("s3://%s@%s/%s", r.Connection, r.Bucket, r.Key)
	}
	return fmt.Sprintf("s3://%s/%s", r.Bucket, r.Key)
}

// ObjectResolver resolves object references from various formats.
// This interface allows other MCP servers to resolve S3 object references
// in a consistent manner.
type ObjectResolver interface {
	// ParseURI parses an S3 URI (e.g., "s3://bucket/key") into an ObjectReference.
	ParseURI(uri string) (*ObjectReference, error)

	// ParseARN parses an S3 ARN (e.g., "arn:aws:s3:::bucket/key") into an ObjectReference.
	ParseARN(arn string) (*ObjectReference, error)

	// Resolve resolves a string that could be a URI, ARN, or simple path.
	Resolve(ref string) (*ObjectReference, error)
}

// DefaultResolver is the default implementation of ObjectResolver.
type DefaultResolver struct {
	// DefaultConnection is used when no connection is specified.
	DefaultConnection string

	// DefaultBucket is used when only a key is provided.
	DefaultBucket string
}

// NewDefaultResolver creates a new default resolver.
func NewDefaultResolver(defaultConnection, defaultBucket string) *DefaultResolver {
	return &DefaultResolver{
		DefaultConnection: defaultConnection,
		DefaultBucket:     defaultBucket,
	}
}

var (
	// s3URIPattern matches S3 URIs: s3://[connection@]bucket/key
	s3URIPattern = regexp.MustCompile(`^s3://(?:([^@]+)@)?([^/]+)/(.+)$`)

	// s3ARNPattern matches S3 ARNs: arn:aws:s3:::bucket/key
	s3ARNPattern = regexp.MustCompile(`^arn:aws:s3:::([^/]+)/(.+)$`)
)

// ParseURI parses an S3 URI into an ObjectReference.
func (r *DefaultResolver) ParseURI(uri string) (*ObjectReference, error) {
	matches := s3URIPattern.FindStringSubmatch(uri)
	if matches == nil {
		return nil, fmt.Errorf("invalid S3 URI: %s", uri)
	}

	ref := &ObjectReference{
		Connection: matches[1],
		Bucket:     matches[2],
		Key:        matches[3],
	}

	if ref.Connection == "" {
		ref.Connection = r.DefaultConnection
	}

	return ref, nil
}

// ParseARN parses an S3 ARN into an ObjectReference.
func (r *DefaultResolver) ParseARN(arn string) (*ObjectReference, error) {
	matches := s3ARNPattern.FindStringSubmatch(arn)
	if matches == nil {
		return nil, fmt.Errorf("invalid S3 ARN: %s", arn)
	}

	return &ObjectReference{
		Connection: r.DefaultConnection,
		Bucket:     matches[1],
		Key:        matches[2],
	}, nil
}

// Resolve resolves a reference string that could be a URI, ARN, or simple path.
func (r *DefaultResolver) Resolve(ref string) (*ObjectReference, error) {
	// Try URI first
	if obj, err := r.ParseURI(ref); err == nil {
		return obj, nil
	}

	// Try ARN
	if obj, err := r.ParseARN(ref); err == nil {
		return obj, nil
	}

	// Treat as simple key with default bucket
	if r.DefaultBucket == "" {
		return nil, fmt.Errorf("cannot resolve %q: no default bucket configured", ref)
	}

	return &ObjectReference{
		Connection: r.DefaultConnection,
		Bucket:     r.DefaultBucket,
		Key:        ref,
	}, nil
}

// Ensure DefaultResolver implements ObjectResolver.
var _ ObjectResolver = (*DefaultResolver)(nil)

// ResolverMiddleware wraps an ObjectResolver with additional processing.
type ResolverMiddleware func(ObjectResolver) ObjectResolver

// WithConnectionAliases creates a middleware that maps connection aliases.
func WithConnectionAliases(aliases map[string]string) ResolverMiddleware {
	return func(resolver ObjectResolver) ObjectResolver {
		return &aliasResolver{
			inner:   resolver,
			aliases: aliases,
		}
	}
}

type aliasResolver struct {
	inner   ObjectResolver
	aliases map[string]string
}

func (r *aliasResolver) ParseURI(uri string) (*ObjectReference, error) {
	ref, err := r.inner.ParseURI(uri)
	if err != nil {
		return nil, err
	}
	if alias, ok := r.aliases[ref.Connection]; ok {
		ref.Connection = alias
	}
	return ref, nil
}

func (r *aliasResolver) ParseARN(arn string) (*ObjectReference, error) {
	return r.inner.ParseARN(arn)
}

func (r *aliasResolver) Resolve(ref string) (*ObjectReference, error) {
	obj, err := r.inner.Resolve(ref)
	if err != nil {
		return nil, err
	}
	if alias, ok := r.aliases[obj.Connection]; ok {
		obj.Connection = alias
	}
	return obj, nil
}

// Ensure aliasResolver implements ObjectResolver.
var _ ObjectResolver = (*aliasResolver)(nil)

// ObjectExists checks if an object exists at the given reference.
type ObjectExistsFunc func(ctx context.Context, ref *ObjectReference) (bool, error)
