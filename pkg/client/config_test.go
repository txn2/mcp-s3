package client

import (
	"os"
	"testing"
	"time"
)

func TestFromEnv(t *testing.T) {
	// Save and restore environment
	envVars := []string{
		"AWS_REGION",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_PROFILE",
		"S3_ENDPOINT",
		"S3_USE_PATH_STYLE",
		"S3_TIMEOUT",
		"S3_CONNECTION_NAME",
		"S3_DISABLE_SSL",
	}

	saved := make(map[string]string)
	for _, v := range envVars {
		saved[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range saved {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Clear all env vars
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	t.Run("defaults", func(t *testing.T) {
		cfg := FromEnv()

		if cfg.Region != DefaultRegion {
			t.Errorf("expected region %q, got %q", DefaultRegion, cfg.Region)
		}
		if cfg.Timeout != DefaultTimeout {
			t.Errorf("expected timeout %v, got %v", DefaultTimeout, cfg.Timeout)
		}
		if cfg.UsePathStyle {
			t.Error("expected UsePathStyle to be false by default")
		}
		if cfg.DisableSSL {
			t.Error("expected DisableSSL to be false by default")
		}
	})

	t.Run("custom values", func(t *testing.T) {
		os.Setenv("AWS_REGION", "eu-west-1")
		os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
		os.Setenv("AWS_SESSION_TOKEN", "test-token")
		os.Setenv("AWS_PROFILE", "test-profile")
		os.Setenv("S3_ENDPOINT", "http://localhost:9000")
		os.Setenv("S3_USE_PATH_STYLE", "true")
		os.Setenv("S3_TIMEOUT", "60s")
		os.Setenv("S3_CONNECTION_NAME", "test-conn")
		os.Setenv("S3_DISABLE_SSL", "true")
		defer func() {
			for _, v := range envVars {
				os.Unsetenv(v)
			}
		}()

		cfg := FromEnv()

		if cfg.Region != "eu-west-1" {
			t.Errorf("expected region %q, got %q", "eu-west-1", cfg.Region)
		}
		if cfg.AccessKeyID != "test-access-key" {
			t.Errorf("expected access key %q, got %q", "test-access-key", cfg.AccessKeyID)
		}
		if cfg.SecretAccessKey != "test-secret-key" {
			t.Errorf("expected secret key %q, got %q", "test-secret-key", cfg.SecretAccessKey)
		}
		if cfg.SessionToken != "test-token" {
			t.Errorf("expected session token %q, got %q", "test-token", cfg.SessionToken)
		}
		if cfg.Profile != "test-profile" {
			t.Errorf("expected profile %q, got %q", "test-profile", cfg.Profile)
		}
		if cfg.Endpoint != "http://localhost:9000" {
			t.Errorf("expected endpoint %q, got %q", "http://localhost:9000", cfg.Endpoint)
		}
		if !cfg.UsePathStyle {
			t.Error("expected UsePathStyle to be true")
		}
		if cfg.Timeout != 60*time.Second {
			t.Errorf("expected timeout 60s, got %v", cfg.Timeout)
		}
		if cfg.Name != "test-conn" {
			t.Errorf("expected name %q, got %q", "test-conn", cfg.Name)
		}
		if !cfg.DisableSSL {
			t.Error("expected DisableSSL to be true")
		}
	})

	t.Run("invalid bool defaults to false", func(t *testing.T) {
		os.Setenv("S3_USE_PATH_STYLE", "invalid")
		defer os.Unsetenv("S3_USE_PATH_STYLE")

		cfg := FromEnv()
		if cfg.UsePathStyle {
			t.Error("expected UsePathStyle to be false for invalid value")
		}
	})

	t.Run("invalid duration defaults", func(t *testing.T) {
		os.Setenv("S3_TIMEOUT", "invalid")
		defer os.Unsetenv("S3_TIMEOUT")

		cfg := FromEnv()
		if cfg.Timeout != DefaultTimeout {
			t.Errorf("expected timeout %v, got %v", DefaultTimeout, cfg.Timeout)
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("sets defaults", func(t *testing.T) {
		cfg := &Config{}
		err := cfg.Validate()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.Region != DefaultRegion {
			t.Errorf("expected region %q, got %q", DefaultRegion, cfg.Region)
		}
		if cfg.Timeout != DefaultTimeout {
			t.Errorf("expected timeout %v, got %v", DefaultTimeout, cfg.Timeout)
		}
	})

	t.Run("preserves custom values", func(t *testing.T) {
		cfg := &Config{
			Region:  "ap-southeast-1",
			Timeout: 120 * time.Second,
		}
		err := cfg.Validate()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.Region != "ap-southeast-1" {
			t.Errorf("expected region %q, got %q", "ap-southeast-1", cfg.Region)
		}
		if cfg.Timeout != 120*time.Second {
			t.Errorf("expected timeout 120s, got %v", cfg.Timeout)
		}
	})
}

func TestConfig_HasCredentials(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{
			name:     "no credentials",
			cfg:      Config{},
			expected: false,
		},
		{
			name: "access key only",
			cfg: Config{
				AccessKeyID: "test-key",
			},
			expected: false,
		},
		{
			name: "secret key only",
			cfg: Config{
				SecretAccessKey: "test-secret",
			},
			expected: false,
		},
		{
			name: "both credentials",
			cfg: Config{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.HasCredentials(); got != tt.expected {
				t.Errorf("HasCredentials() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_HasEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{
			name:     "no endpoint",
			cfg:      Config{},
			expected: false,
		},
		{
			name: "with endpoint",
			cfg: Config{
				Endpoint: "http://localhost:9000",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.HasEndpoint(); got != tt.expected {
				t.Errorf("HasEndpoint() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_Clone(t *testing.T) {
	original := &Config{
		Region:          "us-west-2",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		SessionToken:    "test-token",
		Profile:         "test-profile",
		UsePathStyle:    true,
		Timeout:         60 * time.Second,
		Name:            "test-conn",
		DisableSSL:      true,
	}

	clone := original.Clone()

	// Verify clone has same values
	if clone.Region != original.Region {
		t.Errorf("Region mismatch: got %q, expected %q", clone.Region, original.Region)
	}
	if clone.Endpoint != original.Endpoint {
		t.Errorf("Endpoint mismatch: got %q, expected %q", clone.Endpoint, original.Endpoint)
	}
	if clone.AccessKeyID != original.AccessKeyID {
		t.Errorf("AccessKeyID mismatch: got %q, expected %q", clone.AccessKeyID, original.AccessKeyID)
	}
	if clone.SecretAccessKey != original.SecretAccessKey {
		t.Errorf("SecretAccessKey mismatch: got %q, expected %q", clone.SecretAccessKey, original.SecretAccessKey)
	}
	if clone.SessionToken != original.SessionToken {
		t.Errorf("SessionToken mismatch: got %q, expected %q", clone.SessionToken, original.SessionToken)
	}
	if clone.Profile != original.Profile {
		t.Errorf("Profile mismatch: got %q, expected %q", clone.Profile, original.Profile)
	}
	if clone.UsePathStyle != original.UsePathStyle {
		t.Errorf("UsePathStyle mismatch: got %v, expected %v", clone.UsePathStyle, original.UsePathStyle)
	}
	if clone.Timeout != original.Timeout {
		t.Errorf("Timeout mismatch: got %v, expected %v", clone.Timeout, original.Timeout)
	}
	if clone.Name != original.Name {
		t.Errorf("Name mismatch: got %q, expected %q", clone.Name, original.Name)
	}
	if clone.DisableSSL != original.DisableSSL {
		t.Errorf("DisableSSL mismatch: got %v, expected %v", clone.DisableSSL, original.DisableSSL)
	}

	// Verify clone is independent
	clone.Region = "ap-northeast-1"
	if original.Region == clone.Region {
		t.Error("clone should be independent from original")
	}
}
