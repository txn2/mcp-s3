package multiserver

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func FuzzUnmarshalJSON(f *testing.F) {
	// Valid JSON configs
	f.Add(`{"connections":[]}`)
	f.Add(`{"default":"main","connections":[{"name":"main","region":"us-east-1"}]}`)
	f.Add(`{"connections":[{"name":"test","endpoint":"http://localhost:9000"}]}`)

	// Invalid JSON
	f.Add(`{}`)
	f.Add(`{"connections":null}`)
	f.Add(`not json at all`)
	f.Add(`{"connections":[{"name":""}]}`)
	f.Add(``)

	f.Fuzz(func(t *testing.T, jsonData string) {
		var cfg MultiConfig
		// Should never panic
		err := json.Unmarshal([]byte(jsonData), &cfg)

		// If no error, config should be accessible
		if err == nil {
			_ = cfg.ConnectionNames()
		}
	})
}

func FuzzUnmarshalYAML(f *testing.F) {
	// Valid YAML configs
	f.Add("connections: []")
	f.Add("default: main\nconnections:\n  - name: main\n    region: us-east-1")
	f.Add("connections:\n  - name: test\n    endpoint: http://localhost:9000")

	// Invalid YAML
	f.Add("")
	f.Add("not: [valid: yaml")
	f.Add("connections: null")

	f.Fuzz(func(t *testing.T, yamlData string) {
		var cfg MultiConfig
		// Should never panic
		err := yaml.Unmarshal([]byte(yamlData), &cfg)

		// If no error, config should be accessible
		if err == nil {
			_ = cfg.ConnectionNames()
		}
	})
}

func FuzzConnectionConfigToClientConfig(f *testing.F) {
	f.Add("test", "us-east-1", "http://localhost:9000", "key", "secret", "", "", true, false)
	f.Add("", "", "", "", "", "", "", false, false)
	f.Add("prod", "eu-west-1", "", "AKIA...", "secret123", "token", "profile", false, true)

	f.Fuzz(func(t *testing.T, name, region, endpoint, accessKey, secretKey, token, profile string, pathStyle, disableSSL bool) {
		cfg := ConnectionConfig{
			Name:            name,
			Region:          region,
			Endpoint:        endpoint,
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			SessionToken:    token,
			Profile:         profile,
			UsePathStyle:    pathStyle,
			DisableSSL:      disableSSL,
		}

		// Should never panic
		clientCfg := cfg.ToClientConfig()

		// Validate conversion
		if clientCfg.Name != name {
			t.Errorf("Name mismatch: got %q, expected %q", clientCfg.Name, name)
		}
		if clientCfg.Region != region {
			t.Errorf("Region mismatch: got %q, expected %q", clientCfg.Region, region)
		}
	})
}
