// Package client provides an S3 client wrapper with configuration management.
package client

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Default values for configuration.
const (
	DefaultRegion  = "us-east-1"
	DefaultTimeout = 30 * time.Second
)

// Config holds the configuration for connecting to an S3-compatible storage service.
type Config struct {
	// Region is the AWS region for the S3 service.
	Region string

	// Endpoint is an optional custom endpoint URL for S3-compatible services (SeaweedFS, LocalStack, etc.).
	Endpoint string

	// AccessKeyID is the AWS access key ID. If empty, the SDK credential chain is used.
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string

	// SessionToken is an optional session token for temporary credentials.
	SessionToken string

	// Profile is an optional AWS profile name to use from shared credentials/config.
	Profile string

	// UsePathStyle enables path-style addressing instead of virtual-hosted style.
	// Required for some S3-compatible services like SeaweedFS.
	UsePathStyle bool

	// Timeout is the timeout for S3 operations.
	Timeout time.Duration

	// Name is an optional identifier for this connection (used in multi-connection setups).
	Name string

	// DisableSSL disables SSL/TLS for the connection (useful for local development).
	DisableSSL bool
}

// FromEnv creates a Config populated from environment variables.
//
// Environment variables:
//   - AWS_REGION: AWS region (default: us-east-1)
//   - AWS_ACCESS_KEY_ID: Access key ID
//   - AWS_SECRET_ACCESS_KEY: Secret access key
//   - AWS_SESSION_TOKEN: Session token (optional)
//   - AWS_PROFILE: Profile name (optional)
//   - S3_ENDPOINT: Custom endpoint URL (optional)
//   - S3_USE_PATH_STYLE: Use path-style URLs (default: false)
//   - S3_TIMEOUT: Operation timeout (default: 30s)
//   - S3_CONNECTION_NAME: Connection name (optional)
//   - S3_DISABLE_SSL: Disable SSL (default: false)
func FromEnv() Config {
	cfg := Config{
		Region:          getEnvOrDefault("AWS_REGION", DefaultRegion),
		Endpoint:        getEnvSanitized("S3_ENDPOINT"),
		AccessKeyID:     getEnvSanitized("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: getEnvSanitized("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    getEnvSanitized("AWS_SESSION_TOKEN"),
		Profile:         getEnvSanitized("AWS_PROFILE"),
		UsePathStyle:    getEnvBool("S3_USE_PATH_STYLE", false),
		Timeout:         getEnvDuration("S3_TIMEOUT", DefaultTimeout),
		Name:            getEnvSanitized("S3_CONNECTION_NAME"),
		DisableSSL:      getEnvBool("S3_DISABLE_SSL", false),
	}

	return cfg
}

// Validate checks if the configuration is valid.
// It returns an error if required fields are missing or invalid.
func (c *Config) Validate() error {
	if c.Region == "" {
		c.Region = DefaultRegion
	}

	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}

	return nil
}

// HasCredentials returns true if explicit credentials are configured.
func (c *Config) HasCredentials() bool {
	return c.AccessKeyID != "" && c.SecretAccessKey != ""
}

// HasEndpoint returns true if a custom endpoint is configured.
func (c *Config) HasEndpoint() bool {
	return c.Endpoint != ""
}

// Clone creates a deep copy of the configuration.
func (c *Config) Clone() *Config {
	return &Config{
		Region:          c.Region,
		Endpoint:        c.Endpoint,
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
		Profile:         c.Profile,
		UsePathStyle:    c.UsePathStyle,
		Timeout:         c.Timeout,
		Name:            c.Name,
		DisableSSL:      c.DisableSSL,
	}
}

// getEnvOrDefault returns the value of an environment variable or a default value.
// It treats unresolved template variables (from mcpb) as empty strings.
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" || isUnresolvedTemplateVar(value) {
		return defaultValue
	}
	return value
}

// getEnvBool returns the boolean value of an environment variable or a default value.
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

// getEnvDuration returns the duration value of an environment variable or a default value.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

// isUnresolvedTemplateVar checks if a string appears to be an unresolved
// template variable (e.g., "${user_config.some_value}").
// This can occur when mcpb doesn't resolve optional configuration fields.
func isUnresolvedTemplateVar(s string) bool {
	return strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}")
}

// getEnvSanitized returns the value of an environment variable,
// treating unresolved template variables as empty strings.
// This handles cases where mcpb passes through template variables
// that weren't configured by the user.
func getEnvSanitized(key string) string {
	value := os.Getenv(key)
	if isUnresolvedTemplateVar(value) {
		return ""
	}
	return value
}

// SanitizeAWSEnvVars clears AWS environment variables that contain
// unresolved template variables. This prevents the AWS SDK from using
// invalid values when mcpb passes through unresolved templates.
// This function must be called before config.LoadDefaultConfig() as the
// AWS SDK reads these environment variables directly.
func SanitizeAWSEnvVars() {
	envVars := []string{
		"AWS_PROFILE",
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
	}
	for _, key := range envVars {
		if isUnresolvedTemplateVar(os.Getenv(key)) {
			_ = os.Unsetenv(key)
		}
	}
}
