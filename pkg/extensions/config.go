// Package extensions provides built-in middleware, interceptors, and transformers for mcp-s3.
package extensions

import (
	"os"
	"strconv"
)

// Config holds configuration for all built-in extensions.
type Config struct {
	// ReadOnly enables read-only mode, blocking all write operations.
	ReadOnly bool

	// SizeLimit enables size limit enforcement.
	SizeLimit bool

	// MaxGetSize is the maximum size in bytes for object retrieval (0 = unlimited).
	MaxGetSize int64

	// MaxPutSize is the maximum size in bytes for object uploads (0 = unlimited).
	MaxPutSize int64

	// Logging enables structured logging of operations.
	Logging bool

	// Audit enables audit logging.
	Audit bool

	// PrefixACL enables prefix-based access control.
	PrefixACL bool

	// AllowedPrefixes is a list of prefixes that are allowed when PrefixACL is enabled.
	AllowedPrefixes []string

	// DeniedPrefixes is a list of prefixes that are denied when PrefixACL is enabled.
	DeniedPrefixes []string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ReadOnly:   true,
		SizeLimit:  true,
		MaxGetSize: 10 * 1024 * 1024,  // 10MB
		MaxPutSize: 100 * 1024 * 1024, // 100MB
		Logging:    false,
		Audit:      false,
		PrefixACL:  false,
	}
}

// FromEnv creates a Config populated from environment variables.
//
// Environment variables:
//   - MCP_S3_EXT_READONLY: Enable read-only mode (default: true)
//   - MCP_S3_EXT_SIZELIMIT: Enable size limits (default: true)
//   - MCP_S3_MAX_GET_SIZE: Max bytes for GET (default: 10MB)
//   - MCP_S3_MAX_PUT_SIZE: Max bytes for PUT (default: 100MB)
//   - MCP_S3_EXT_LOGGING: Enable logging (default: false)
//   - MCP_S3_EXT_AUDIT: Enable audit logging (default: false)
//   - MCP_S3_EXT_PREFIX_ACL: Enable prefix-based ACL (default: false)
func FromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("MCP_S3_EXT_READONLY"); v != "" {
		cfg.ReadOnly = parseBool(v, true)
	}

	if v := os.Getenv("MCP_S3_EXT_SIZELIMIT"); v != "" {
		cfg.SizeLimit = parseBool(v, true)
	}

	if v := os.Getenv("MCP_S3_MAX_GET_SIZE"); v != "" {
		cfg.MaxGetSize = parseSize(v, cfg.MaxGetSize)
	}

	if v := os.Getenv("MCP_S3_MAX_PUT_SIZE"); v != "" {
		cfg.MaxPutSize = parseSize(v, cfg.MaxPutSize)
	}

	if v := os.Getenv("MCP_S3_EXT_LOGGING"); v != "" {
		cfg.Logging = parseBool(v, false)
	}

	if v := os.Getenv("MCP_S3_EXT_AUDIT"); v != "" {
		cfg.Audit = parseBool(v, false)
	}

	if v := os.Getenv("MCP_S3_EXT_PREFIX_ACL"); v != "" {
		cfg.PrefixACL = parseBool(v, false)
	}

	return cfg
}

// parseBool parses a boolean from a string, returning defaultValue on error.
func parseBool(s string, defaultValue bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return defaultValue
	}
	return v
}

// sizeSuffixes maps size suffixes to their multipliers and lengths.
var sizeSuffixes = []struct {
	suffix     string
	multiplier int64
}{
	{"KB", 1024},
	{"MB", 1024 * 1024},
	{"GB", 1024 * 1024 * 1024},
	{"TB", 1024 * 1024 * 1024 * 1024},
	{"K", 1024},
	{"M", 1024 * 1024},
	{"G", 1024 * 1024 * 1024},
	{"T", 1024 * 1024 * 1024 * 1024},
}

// parseSize parses a size value (e.g., "10MB", "1GB") from a string.
// Returns defaultValue for invalid or negative values.
func parseSize(s string, defaultValue int64) int64 {
	if v, err := strconv.ParseInt(s, 10, 64); err == nil && v >= 0 {
		return v
	}
	return parseSizeWithSuffix(s, defaultValue)
}

// parseSizeWithSuffix handles size strings with unit suffixes.
func parseSizeWithSuffix(s string, defaultValue int64) int64 {
	if len(s) < 2 {
		return defaultValue
	}

	upper := toUpper(s)
	for _, entry := range sizeSuffixes {
		if hasSuffix(upper, entry.suffix) {
			numPart := s[:len(s)-len(entry.suffix)]
			if num, err := strconv.ParseInt(numPart, 10, 64); err == nil && num >= 0 {
				// Check for overflow before multiplication
				if num > (1<<63-1)/entry.multiplier {
					return defaultValue
				}
				return num * entry.multiplier
			}
			return defaultValue
		}
	}
	return defaultValue
}

// toUpper converts a string to uppercase without importing strings package.
func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// hasSuffix checks if string s ends with suffix.
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
