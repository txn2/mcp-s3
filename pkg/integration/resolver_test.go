package integration

import (
	"testing"
)

func TestObjectReference_String(t *testing.T) {
	tests := []struct {
		name     string
		ref      ObjectReference
		expected string
	}{
		{
			name: "with connection",
			ref: ObjectReference{
				Connection: "prod",
				Bucket:     "my-bucket",
				Key:        "path/to/file.txt",
			},
			expected: "s3://prod@my-bucket/path/to/file.txt",
		},
		{
			name: "without connection",
			ref: ObjectReference{
				Connection: "",
				Bucket:     "my-bucket",
				Key:        "file.txt",
			},
			expected: "s3://my-bucket/file.txt",
		},
		{
			name: "empty key",
			ref: ObjectReference{
				Bucket: "bucket",
				Key:    "",
			},
			expected: "s3://bucket/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefaultResolver_ParseURI(t *testing.T) {
	resolver := NewDefaultResolver("default-conn", "default-bucket")

	tests := []struct {
		name       string
		uri        string
		wantConn   string
		wantBucket string
		wantKey    string
		wantErr    bool
	}{
		{
			name:       "simple URI",
			uri:        "s3://my-bucket/my-key.txt",
			wantConn:   "default-conn",
			wantBucket: "my-bucket",
			wantKey:    "my-key.txt",
			wantErr:    false,
		},
		{
			name:       "URI with connection",
			uri:        "s3://prod@my-bucket/path/to/file.txt",
			wantConn:   "prod",
			wantBucket: "my-bucket",
			wantKey:    "path/to/file.txt",
			wantErr:    false,
		},
		{
			name:       "URI with nested path",
			uri:        "s3://bucket/a/b/c/d/file.json",
			wantConn:   "default-conn",
			wantBucket: "bucket",
			wantKey:    "a/b/c/d/file.json",
			wantErr:    false,
		},
		{
			name:    "invalid URI - no bucket",
			uri:     "s3:///key",
			wantErr: true,
		},
		{
			name:    "invalid URI - wrong scheme",
			uri:     "http://bucket/key",
			wantErr: true,
		},
		{
			name:    "invalid URI - empty",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "invalid URI - no key",
			uri:     "s3://bucket",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := resolver.ParseURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ref.Connection != tt.wantConn {
				t.Errorf("Connection = %q, want %q", ref.Connection, tt.wantConn)
			}
			if ref.Bucket != tt.wantBucket {
				t.Errorf("Bucket = %q, want %q", ref.Bucket, tt.wantBucket)
			}
			if ref.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", ref.Key, tt.wantKey)
			}
		})
	}
}

func TestDefaultResolver_ParseARN(t *testing.T) {
	resolver := NewDefaultResolver("default-conn", "default-bucket")

	tests := []struct {
		name       string
		arn        string
		wantConn   string
		wantBucket string
		wantKey    string
		wantErr    bool
	}{
		{
			name:       "valid ARN",
			arn:        "arn:aws:s3:::my-bucket/my-key.txt",
			wantConn:   "default-conn",
			wantBucket: "my-bucket",
			wantKey:    "my-key.txt",
			wantErr:    false,
		},
		{
			name:       "ARN with nested path",
			arn:        "arn:aws:s3:::bucket/path/to/file.json",
			wantConn:   "default-conn",
			wantBucket: "bucket",
			wantKey:    "path/to/file.json",
			wantErr:    false,
		},
		{
			name:    "invalid ARN - wrong format",
			arn:     "arn:aws:ec2:::bucket/key",
			wantErr: true,
		},
		{
			name:    "invalid ARN - no key",
			arn:     "arn:aws:s3:::bucket",
			wantErr: true,
		},
		{
			name:    "invalid ARN - empty",
			arn:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := resolver.ParseARN(tt.arn)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ref.Connection != tt.wantConn {
				t.Errorf("Connection = %q, want %q", ref.Connection, tt.wantConn)
			}
			if ref.Bucket != tt.wantBucket {
				t.Errorf("Bucket = %q, want %q", ref.Bucket, tt.wantBucket)
			}
			if ref.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", ref.Key, tt.wantKey)
			}
		})
	}
}

func TestDefaultResolver_Resolve(t *testing.T) {
	t.Run("resolves URI first", func(t *testing.T) {
		resolver := NewDefaultResolver("default", "default-bucket")
		ref, err := resolver.Resolve("s3://bucket/key.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.Bucket != "bucket" {
			t.Errorf("Bucket = %q, want %q", ref.Bucket, "bucket")
		}
		if ref.Key != "key.txt" {
			t.Errorf("Key = %q, want %q", ref.Key, "key.txt")
		}
	})

	t.Run("resolves ARN if not URI", func(t *testing.T) {
		resolver := NewDefaultResolver("default", "default-bucket")
		ref, err := resolver.Resolve("arn:aws:s3:::bucket/key.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.Bucket != "bucket" {
			t.Errorf("Bucket = %q, want %q", ref.Bucket, "bucket")
		}
		if ref.Key != "key.txt" {
			t.Errorf("Key = %q, want %q", ref.Key, "key.txt")
		}
	})

	t.Run("uses default bucket for simple key", func(t *testing.T) {
		resolver := NewDefaultResolver("conn", "my-default-bucket")
		ref, err := resolver.Resolve("path/to/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.Connection != "conn" {
			t.Errorf("Connection = %q, want %q", ref.Connection, "conn")
		}
		if ref.Bucket != "my-default-bucket" {
			t.Errorf("Bucket = %q, want %q", ref.Bucket, "my-default-bucket")
		}
		if ref.Key != "path/to/file.txt" {
			t.Errorf("Key = %q, want %q", ref.Key, "path/to/file.txt")
		}
	})

	t.Run("errors without default bucket", func(t *testing.T) {
		resolver := NewDefaultResolver("conn", "")
		_, err := resolver.Resolve("just-a-key.txt")
		if err == nil {
			t.Error("expected error for simple key without default bucket")
		}
	})
}

func TestWithConnectionAliases(t *testing.T) {
	inner := NewDefaultResolver("default", "bucket")
	aliases := map[string]string{
		"prod":    "production-account",
		"staging": "staging-account",
	}

	resolver := WithConnectionAliases(aliases)(inner)

	t.Run("resolves alias in URI", func(t *testing.T) {
		ref, err := resolver.ParseURI("s3://prod@bucket/key.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.Connection != "production-account" {
			t.Errorf("Connection = %q, want %q", ref.Connection, "production-account")
		}
	})

	t.Run("keeps non-aliased connection", func(t *testing.T) {
		ref, err := resolver.ParseURI("s3://other@bucket/key.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.Connection != "other" {
			t.Errorf("Connection = %q, want %q", ref.Connection, "other")
		}
	})

	t.Run("resolves alias in Resolve", func(t *testing.T) {
		ref, err := resolver.Resolve("s3://staging@bucket/key.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.Connection != "staging-account" {
			t.Errorf("Connection = %q, want %q", ref.Connection, "staging-account")
		}
	})

	t.Run("ParseARN uses default connection", func(t *testing.T) {
		ref, err := resolver.ParseARN("arn:aws:s3:::bucket/key.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ref.Connection != "default" {
			t.Errorf("Connection = %q, want %q", ref.Connection, "default")
		}
	})
}

func TestNewDefaultResolver(t *testing.T) {
	resolver := NewDefaultResolver("my-conn", "my-bucket")

	if resolver.DefaultConnection != "my-conn" {
		t.Errorf("DefaultConnection = %q, want %q", resolver.DefaultConnection, "my-conn")
	}
	if resolver.DefaultBucket != "my-bucket" {
		t.Errorf("DefaultBucket = %q, want %q", resolver.DefaultBucket, "my-bucket")
	}
}
