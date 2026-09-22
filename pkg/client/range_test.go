package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// rangeStore is an S3 stand-in serving GetObject over HTTP, so GetObjectRange
// runs through the real SDK: the Range header it builds, the Content-Range and
// Content-Length the SDK deserializes, and the error type the SDK wraps a 416
// in. A mock S3API sees none of that.
type rangeStore struct {
	objects     map[string][]byte
	ignoreRange bool

	mu        sync.Mutex
	gotRanges []string
}

func (s *rangeStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Path-style addressing: /<bucket>/<key>.
	_, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	obj, ok := s.objects[key]
	if !ok {
		writeS3Error(w, http.StatusNotFound, "NoSuchKey")
		return
	}
	s.mu.Lock()
	s.gotRanges = append(s.gotRanges, r.Header.Get("Range"))
	s.mu.Unlock()

	rng := r.Header.Get("Range")
	if s.ignoreRange || rng == "" {
		w.Header().Set("Content-Length", strconv.Itoa(len(obj)))
		_, _ = w.Write(obj)
		return
	}

	var start, end int64
	if _, err := fmt.Sscanf(rng, "bytes=%d-%d", &start, &end); err != nil {
		writeS3Error(w, http.StatusBadRequest, "InvalidArgument")
		return
	}
	size := int64(len(obj))
	// S3 answers a range starting at or past the end, including any range of a
	// zero-byte object, with 416 InvalidRange.
	if start >= size {
		w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(size, 10))
		writeS3Error(w, http.StatusRequestedRangeNotSatisfiable, "InvalidRange")
		return
	}
	end = min(end, size-1)
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	w.WriteHeader(http.StatusPartialContent)
	_, _ = w.Write(obj[start : end+1])
}

func writeS3Error(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>%s</Message></Error>`, code, code)
}

// newRangeStoreClient builds a Client with New, pointed at store.
func newRangeStoreClient(t *testing.T, store *rangeStore) *Client {
	t.Helper()
	srv := httptest.NewServer(store)
	t.Cleanup(srv.Close)

	// Keep the developer's own AWS config out of the client under test.
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_PROFILE", "")

	c, err := New(context.Background(), &Config{
		Region:          "us-east-1",
		Endpoint:        srv.URL,
		UsePathStyle:    true,
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestGetObjectRange_ThroughSDK(t *testing.T) {
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 251)
	}
	objects := map[string][]byte{"data.bin": data, "empty.bin": {}}

	t.Run("returns exactly the requested bytes and the whole object's size", func(t *testing.T) {
		store := &rangeStore{objects: objects}
		c := newRangeStoreClient(t, store)

		got, err := c.GetObjectRange(context.Background(), "b", "data.bin", 992, 8)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(got.Body, data[992:1000]) {
			t.Errorf("body = %v, want %v", got.Body, data[992:1000])
		}
		if got.Size != 1000 {
			t.Errorf("Size = %d, want 1000", got.Size)
		}
		if len(store.gotRanges) != 1 || store.gotRanges[0] != "bytes=992-999" {
			t.Errorf("Range sent = %q, want [bytes=992-999]", store.gotRanges)
		}
	})

	t.Run("a range running past the end is cut short at the end", func(t *testing.T) {
		c := newRangeStoreClient(t, &rangeStore{objects: objects})

		got, err := c.GetObjectRange(context.Background(), "b", "data.bin", 998, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(got.Body, data[998:]) || got.Size != 1000 {
			t.Errorf("got body %v size %d, want %v size 1000", got.Body, got.Size, data[998:])
		}
	})

	t.Run("a store that ignores Range yields the bytes at offset, length of them", func(t *testing.T) {
		c := newRangeStoreClient(t, &rangeStore{objects: objects, ignoreRange: true})

		got, err := c.GetObjectRange(context.Background(), "b", "data.bin", 500, 8)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(got.Body, data[500:508]) {
			t.Errorf("body = %v, want %v", got.Body, data[500:508])
		}
		if got.Size != 1000 {
			t.Errorf("Size = %d, want 1000", got.Size)
		}
	})

	t.Run("a zero-byte object is an unsatisfiable range, not an empty read", func(t *testing.T) {
		for _, ignore := range []bool{false, true} {
			c := newRangeStoreClient(t, &rangeStore{objects: objects, ignoreRange: ignore})

			got, err := c.GetObjectRange(context.Background(), "b", "empty.bin", 0, 8)
			if !errors.Is(err, ErrRangeNotSatisfiable) {
				t.Errorf("ignoreRange=%v: expected ErrRangeNotSatisfiable, got result %+v, err %v", ignore, got, err)
			}
		}
	})

	t.Run("an offset past the end is an unsatisfiable range", func(t *testing.T) {
		c := newRangeStoreClient(t, &rangeStore{objects: objects})

		_, err := c.GetObjectRange(context.Background(), "b", "data.bin", 1000, 8)
		if !errors.Is(err, ErrRangeNotSatisfiable) {
			t.Errorf("expected ErrRangeNotSatisfiable, got %v", err)
		}
	})

	t.Run("a missing object is an error, not an unsatisfiable range", func(t *testing.T) {
		c := newRangeStoreClient(t, &rangeStore{objects: objects})

		_, err := c.GetObjectRange(context.Background(), "b", "nope.bin", 0, 8)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, ErrRangeNotSatisfiable) {
			t.Errorf("a 404 must not read as an unsatisfiable range: %v", err)
		}
	})
}
