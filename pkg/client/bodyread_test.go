package client

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These tests run the real SDK against an HTTP server that controls when the
// response headers and each part of the body are sent, so the two phases of
// a read (request, then body) meet real timing rather than a mock.

const (
	testTimeout     = 200 * time.Millisecond
	testIdleTimeout = 200 * time.Millisecond
	// A read that should end on a timer must end well inside this; a hang is
	// a failure, not a slow pass.
	testHangLimit = 3 * time.Second
)

// timedHandler serves one object. Its hold channel is closed when the test
// ends, so a handler that stalls can block on it without outliving the test.
type timedHandler func(w http.ResponseWriter, r *http.Request, hold <-chan struct{})

// newTimedClient builds a Client with New, pointed at a server running h.
func newTimedClient(t *testing.T, h timedHandler, timeout, idle time.Duration) *Client {
	t.Helper()
	hold := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r, hold)
	}))
	// Cleanups run last-in first-out: release stalled handlers, then Close,
	// which waits for them.
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(hold) })

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
		Timeout:         timeout,
		ReadIdleTimeout: idle,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// trickle sends body in chunks with gap between them, flushing each, so the
// transfer takes len(chunks)*gap in total while never pausing longer than gap.
func trickle(w http.ResponseWriter, chunks [][]byte, gap time.Duration) {
	flusher, _ := w.(http.Flusher)
	for _, chunk := range chunks {
		_, _ = w.Write(chunk)
		if flusher != nil {
			flusher.Flush()
		}
		time.Sleep(gap)
	}
}

// stallAfter sends first, then sends nothing until the test ends.
func stallAfter(w http.ResponseWriter, r *http.Request, hold <-chan struct{}, first []byte) {
	_, _ = w.Write(first)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	select {
	case <-hold:
	case <-r.Context().Done():
	}
}

func TestGetObject_SlowSteadyBodyOutlastsTimeout(t *testing.T) {
	chunks := make([][]byte, 10)
	var want []byte
	for i := range chunks {
		chunks[i] = bytes.Repeat([]byte{byte('a' + i)}, 100)
		want = append(want, chunks[i]...)
	}
	// 10 chunks 50ms apart: about 500ms in all, well past Timeout, but never
	// idle for longer than 50ms.
	c := newTimedClient(t, func(w http.ResponseWriter, r *http.Request, _ <-chan struct{}) {
		w.Header().Set("Content-Length", strconv.Itoa(len(want)))
		trickle(w, chunks, 50*time.Millisecond)
	}, testTimeout, testIdleTimeout)

	start := time.Now()
	got, err := c.GetObject(context.Background(), "b", "k")
	if err != nil {
		t.Fatalf("a transfer that keeps moving must not be cut off: %v", err)
	}
	if elapsed := time.Since(start); elapsed <= testTimeout {
		t.Fatalf("transfer took %s, not longer than Timeout %s; the test proves nothing", elapsed, testTimeout)
	}
	if !bytes.Equal(got.Body, want) {
		t.Errorf("body mismatch: got %d bytes, want %d", len(got.Body), len(want))
	}
}

func TestGetObject_StalledBodyFailsWithIdleTimeout(t *testing.T) {
	c := newTimedClient(t, func(w http.ResponseWriter, r *http.Request, hold <-chan struct{}) {
		w.Header().Set("Content-Length", "1000")
		stallAfter(w, r, hold, bytes.Repeat([]byte{'x'}, 100))
	}, 10*time.Second, testIdleTimeout)

	start := time.Now()
	_, err := c.GetObject(context.Background(), "b", "k")
	elapsed := time.Since(start)
	if !errors.Is(err, ErrReadIdleTimeout) {
		t.Fatalf("expected ErrReadIdleTimeout, got %v", err)
	}
	if n := strings.Count(err.Error(), "no data for"); n != 1 {
		t.Errorf("the stall should be stated once, found %d times: %v", n, err)
	}
	if elapsed > testHangLimit {
		t.Errorf("stall detected after %s; expected about %s", elapsed, testIdleTimeout)
	}
}

func TestGetObject_NoResponseWithinTimeoutIsDeadlineExceeded(t *testing.T) {
	c := newTimedClient(t, func(w http.ResponseWriter, r *http.Request, hold <-chan struct{}) {
		select {
		case <-hold:
		case <-r.Context().Done():
		}
	}, testTimeout, 10*time.Second)

	start := time.Now()
	_, err := c.GetObject(context.Background(), "b", "k")
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected an error matching context.DeadlineExceeded, got %v", err)
	}
	if errors.Is(err, ErrReadIdleTimeout) {
		t.Errorf("a request timeout must not read as a stalled body: %v", err)
	}
	if n := strings.Count(err.Error(), "no response from S3"); n != 1 {
		t.Errorf("the request timeout should be stated once, found %d times: %v", n, err)
	}
	if elapsed > testHangLimit {
		t.Errorf("timed out after %s; expected about %s", elapsed, testTimeout)
	}
}

func TestGetObjectRange_StalledBodyFailsWithIdleTimeout(t *testing.T) {
	c := newTimedClient(t, func(w http.ResponseWriter, r *http.Request, hold <-chan struct{}) {
		w.Header().Set("Content-Range", "bytes 0-999/5000")
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusPartialContent)
		stallAfter(w, r, hold, bytes.Repeat([]byte{'x'}, 100))
	}, 10*time.Second, testIdleTimeout)

	_, err := c.GetObjectRange(context.Background(), "b", "k", 0, 1000)
	if !errors.Is(err, ErrReadIdleTimeout) {
		t.Fatalf("expected ErrReadIdleTimeout, got %v", err)
	}
}

func TestGetObjectRange_StallWhileSkippingToOffsetFailsWithIdleTimeout(t *testing.T) {
	// A store that ignores Range: the client discards bytes up to the offset,
	// and a stall there is bounded the same way as one in the range itself.
	c := newTimedClient(t, func(w http.ResponseWriter, r *http.Request, hold <-chan struct{}) {
		w.Header().Set("Content-Length", "5000")
		stallAfter(w, r, hold, bytes.Repeat([]byte{'x'}, 100))
	}, 10*time.Second, testIdleTimeout)

	_, err := c.GetObjectRange(context.Background(), "b", "k", 4000, 8)
	if !errors.Is(err, ErrReadIdleTimeout) {
		t.Fatalf("expected ErrReadIdleTimeout, got %v", err)
	}
}

func TestGetObject_CallerContextStillEndsTheRead(t *testing.T) {
	c := newTimedClient(t, func(w http.ResponseWriter, r *http.Request, hold <-chan struct{}) {
		w.Header().Set("Content-Length", "1000")
		stallAfter(w, r, hold, []byte("x"))
	}, 10*time.Second, 10*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	start := time.Now()
	_, err := c.GetObject(ctx, "b", "k")
	if err == nil {
		t.Fatal("expected the caller's deadline to end the read")
	}
	if errors.Is(err, ErrReadIdleTimeout) {
		t.Errorf("the caller's deadline must not read as a stalled body: %v", err)
	}
	if elapsed := time.Since(start); elapsed > testHangLimit {
		t.Errorf("read ended after %s; expected about %s", elapsed, testTimeout)
	}
}
