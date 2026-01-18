// Package multiserver provides support for managing multiple S3 connections.
package multiserver

import (
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/txn2/mcp-s3/pkg/client"
)

// ConnectionConfig represents configuration for a single S3 connection.
type ConnectionConfig struct {
	// Name is the unique identifier for this connection.
	Name string `json:"name" yaml:"name"`

	// Region is the AWS region.
	Region string `json:"region,omitempty" yaml:"region,omitempty"`

	// Endpoint is an optional custom endpoint URL.
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

	// AccessKeyID is the AWS access key ID.
	AccessKeyID string `json:"access_key_id,omitempty" yaml:"access_key_id,omitempty"`

	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string `json:"secret_access_key,omitempty" yaml:"secret_access_key,omitempty"`

	// SessionToken is an optional session token.
	SessionToken string `json:"session_token,omitempty" yaml:"session_token,omitempty"`

	// Profile is an optional AWS profile name.
	Profile string `json:"profile,omitempty" yaml:"profile,omitempty"`

	// UsePathStyle enables path-style addressing.
	UsePathStyle bool `json:"use_path_style,omitempty" yaml:"use_path_style,omitempty"`

	// DisableSSL disables SSL/TLS.
	DisableSSL bool `json:"disable_ssl,omitempty" yaml:"disable_ssl,omitempty"`
}

// ToClientConfig converts a ConnectionConfig to a client.Config.
func (c *ConnectionConfig) ToClientConfig() *client.Config {
	return &client.Config{
		Name:            c.Name,
		Region:          c.Region,
		Endpoint:        c.Endpoint,
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
		Profile:         c.Profile,
		UsePathStyle:    c.UsePathStyle,
		DisableSSL:      c.DisableSSL,
	}
}

// MultiConfig holds configuration for multiple S3 connections.
type MultiConfig struct {
	// DefaultConnection is the name of the default connection.
	DefaultConnection string `json:"default_connection,omitempty" yaml:"default_connection,omitempty"`

	// Connections is a list of connection configurations.
	Connections []ConnectionConfig `json:"connections" yaml:"connections"`
}

// FromEnvJSON loads multi-connection configuration from the S3_ADDITIONAL_CONNECTIONS
// environment variable, which should contain a JSON object mapping connection names
// to connection configurations.
func FromEnvJSON() (*MultiConfig, error) {
	jsonStr := os.Getenv("S3_ADDITIONAL_CONNECTIONS")
	if jsonStr == "" {
		return nil, nil
	}

	// Try to parse as a map of connection configs
	var connMap map[string]ConnectionConfig
	if err := json.Unmarshal([]byte(jsonStr), &connMap); err != nil {
		return nil, err
	}

	cfg := &MultiConfig{
		DefaultConnection: os.Getenv("S3_CONNECTION_NAME"),
		Connections:       make([]ConnectionConfig, 0, len(connMap)),
	}

	for name, conn := range connMap {
		conn.Name = name
		cfg.Connections = append(cfg.Connections, conn)
	}

	return cfg, nil
}

// FromYAMLFile loads multi-connection configuration from a YAML file.
func FromYAMLFile(path string) (*MultiConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg MultiConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// FromJSONFile loads multi-connection configuration from a JSON file.
func FromJSONFile(path string) (*MultiConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg MultiConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetConnection returns the configuration for a specific connection by name.
func (c *MultiConfig) GetConnection(name string) *ConnectionConfig {
	for i := range c.Connections {
		if c.Connections[i].Name == name {
			return &c.Connections[i]
		}
	}
	return nil
}

// ConnectionNames returns a list of all connection names.
func (c *MultiConfig) ConnectionNames() []string {
	names := make([]string, 0, len(c.Connections))
	for _, conn := range c.Connections {
		names = append(names, conn.Name)
	}
	return names
}
