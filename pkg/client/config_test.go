package client

import (
	"os"
	"testing"
	"time"
)

func TestFromEnv(t *testing.T) {
	envVars := []string{
		"AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN", "AWS_PROFILE", "S3_ENDPOINT",
		"S3_USE_PATH_STYLE", "S3_TIMEOUT", "S3_CONNECTION_NAME", "S3_DISABLE_SSL",
	}

	saved := saveEnv(envVars)
	defer restoreEnv(saved)
	clearEnv(envVars)

	t.Run("defaults", func(t *testing.T) {
		cfg := FromEnv()
		assertString(t, "Region", DefaultRegion, cfg.Region)
		assertDuration(t, "Timeout", DefaultTimeout, cfg.Timeout)
		assertBool(t, "UsePathStyle", false, cfg.UsePathStyle)
		assertBool(t, "DisableSSL", false, cfg.DisableSSL)
	})

	t.Run("custom values", func(t *testing.T) {
		setEnvVars(map[string]string{
			"AWS_REGION":            "eu-west-1",
			"AWS_ACCESS_KEY_ID":     "test-access-key",
			"AWS_SECRET_ACCESS_KEY": "test-secret-key",
			"AWS_SESSION_TOKEN":     "test-token",
			"AWS_PROFILE":           "test-profile",
			"S3_ENDPOINT":           "http://localhost:9000",
			"S3_USE_PATH_STYLE":     "true",
			"S3_TIMEOUT":            "60s",
			"S3_CONNECTION_NAME":    "test-conn",
			"S3_DISABLE_SSL":        "true",
		})
		defer clearEnv(envVars)

		cfg := FromEnv()
		assertString(t, "Region", "eu-west-1", cfg.Region)
		assertString(t, "AccessKeyID", "test-access-key", cfg.AccessKeyID)
		assertString(t, "SecretAccessKey", "test-secret-key", cfg.SecretAccessKey)
		assertString(t, "SessionToken", "test-token", cfg.SessionToken)
		assertString(t, "Profile", "test-profile", cfg.Profile)
		assertString(t, "Endpoint", "http://localhost:9000", cfg.Endpoint)
		assertBool(t, "UsePathStyle", true, cfg.UsePathStyle)
		assertDuration(t, "Timeout", 60*time.Second, cfg.Timeout)
		assertString(t, "Name", "test-conn", cfg.Name)
		assertBool(t, "DisableSSL", true, cfg.DisableSSL)
	})

	t.Run("invalid bool defaults to false", func(t *testing.T) {
		os.Setenv("S3_USE_PATH_STYLE", "invalid")
		defer os.Unsetenv("S3_USE_PATH_STYLE")
		cfg := FromEnv()
		assertBool(t, "UsePathStyle", false, cfg.UsePathStyle)
	})

	t.Run("invalid duration defaults", func(t *testing.T) {
		os.Setenv("S3_TIMEOUT", "invalid")
		defer os.Unsetenv("S3_TIMEOUT")
		cfg := FromEnv()
		assertDuration(t, "Timeout", DefaultTimeout, cfg.Timeout)
	})

	t.Run("unresolved template variables treated as empty", func(t *testing.T) {
		setEnvVars(map[string]string{
			"AWS_PROFILE":           "${user_config.aws_profile}",
			"S3_ENDPOINT":           "${user_config.s3_endpoint}",
			"AWS_ACCESS_KEY_ID":     "${user_config.aws_access_key_id}",
			"AWS_SECRET_ACCESS_KEY": "${user_config.aws_secret_access_key}",
			"AWS_SESSION_TOKEN":     "${user_config.aws_session_token}",
			"S3_CONNECTION_NAME":    "${user_config.connection_name}",
		})
		defer clearEnv(envVars)

		cfg := FromEnv()
		assertString(t, "Profile", "", cfg.Profile)
		assertString(t, "Endpoint", "", cfg.Endpoint)
		assertString(t, "AccessKeyID", "", cfg.AccessKeyID)
		assertString(t, "SecretAccessKey", "", cfg.SecretAccessKey)
		assertString(t, "SessionToken", "", cfg.SessionToken)
		assertString(t, "Name", "", cfg.Name)
	})
}

func TestIsUnresolvedTemplateVar(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"${user_config.aws_profile}", true},
		{"${some_var}", true},
		{"${}", true},
		{"normal-value", false},
		{"", false},
		{"$missing_braces", false},
		{"{missing_dollar}", false},
		{"prefix${var}suffix", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isUnresolvedTemplateVar(tt.input); got != tt.expected {
				t.Errorf("isUnresolvedTemplateVar(%q) = %v, expected %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("sets defaults", func(t *testing.T) {
		cfg := &Config{}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assertString(t, "Region", DefaultRegion, cfg.Region)
		assertDuration(t, "Timeout", DefaultTimeout, cfg.Timeout)
	})

	t.Run("preserves custom values", func(t *testing.T) {
		cfg := &Config{Region: "ap-southeast-1", Timeout: 120 * time.Second}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assertString(t, "Region", "ap-southeast-1", cfg.Region)
		assertDuration(t, "Timeout", 120*time.Second, cfg.Timeout)
	})
}

func TestConfig_HasCredentials(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{"no credentials", Config{}, false},
		{"access key only", Config{AccessKeyID: "test-key"}, false},
		{"secret key only", Config{SecretAccessKey: "test-secret"}, false},
		{"both credentials", Config{AccessKeyID: "test-key", SecretAccessKey: "test-secret"}, true},
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
		{"no endpoint", Config{}, false},
		{"with endpoint", Config{Endpoint: "http://localhost:9000"}, true},
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
	assertConfigEqual(t, original, clone)

	// Verify clone is independent
	clone.Region = "ap-northeast-1"
	if original.Region == clone.Region {
		t.Error("clone should be independent from original")
	}
}

// Test helpers

func saveEnv(vars []string) map[string]string {
	saved := make(map[string]string)
	for _, v := range vars {
		saved[v] = os.Getenv(v)
	}
	return saved
}

func restoreEnv(saved map[string]string) {
	for k, v := range saved {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
}

func clearEnv(vars []string) {
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func setEnvVars(vars map[string]string) {
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func assertString(t *testing.T, field, expected, got string) {
	t.Helper()
	if got != expected {
		t.Errorf("%s: expected %q, got %q", field, expected, got)
	}
}

func assertDuration(t *testing.T, field string, expected, got time.Duration) {
	t.Helper()
	if got != expected {
		t.Errorf("%s: expected %v, got %v", field, expected, got)
	}
}

func assertBool(t *testing.T, field string, expected, got bool) {
	t.Helper()
	if got != expected {
		t.Errorf("%s: expected %v, got %v", field, expected, got)
	}
}

func assertConfigEqual(t *testing.T, expected, got *Config) {
	t.Helper()
	assertString(t, "Region", expected.Region, got.Region)
	assertString(t, "Endpoint", expected.Endpoint, got.Endpoint)
	assertString(t, "AccessKeyID", expected.AccessKeyID, got.AccessKeyID)
	assertString(t, "SecretAccessKey", expected.SecretAccessKey, got.SecretAccessKey)
	assertString(t, "SessionToken", expected.SessionToken, got.SessionToken)
	assertString(t, "Profile", expected.Profile, got.Profile)
	assertBool(t, "UsePathStyle", expected.UsePathStyle, got.UsePathStyle)
	assertDuration(t, "Timeout", expected.Timeout, got.Timeout)
	assertString(t, "Name", expected.Name, got.Name)
	assertBool(t, "DisableSSL", expected.DisableSSL, got.DisableSSL)
}
