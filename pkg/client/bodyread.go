package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// ErrReadIdleTimeout is returned by GetObject and GetObjectRange when an
// object body stops arriving for longer than Config.ReadIdleTimeout.
var ErrReadIdleTimeout = errors.New("object body read stalled")

// errRequestTimeout is the cause recorded when Config.Timeout expires before
// the response headers arrive. It wraps context.DeadlineExceeded so a caller
// testing for that keeps working: before the body read had its own bound,
// Timeout was a context deadline and the SDK error wrapped
// context.DeadlineExceeded directly.
var errRequestTimeout = fmt.Errorf("no response from S3 within the configured timeout: %w", context.DeadlineExceeded)

// readDeadlines bounds a call that reads an object body in two phases.
// Config.Timeout covers the request, up to the response headers. After that
// only Config.ReadIdleTimeout applies, restarted by every read that returns
// bytes, so a slow transfer that keeps moving is never cut off.
//
// Both bounds are timers that cancel one context rather than a deadline on
// it, because the body is read on the request's context: net/http aborts a
// response body read when the request context is canceled, so the context
// has to outlive the headers, and a context deadline cannot be lifted once
// set.
type readDeadlines struct {
	ctx       context.Context
	cancel    context.CancelCauseFunc
	request   *time.Timer
	idle      time.Duration
	idleTimer *time.Timer
}

// startRead begins the request phase. The caller must call stop when done.
func (c *Client) startRead(ctx context.Context) *readDeadlines {
	ctx, cancel := context.WithCancelCause(ctx)
	d := &readDeadlines{ctx: ctx, cancel: cancel, idle: c.config.ReadIdleTimeout}
	if c.config.Timeout > 0 {
		d.request = time.AfterFunc(c.config.Timeout, func() { cancel(errRequestTimeout) })
	}
	return d
}

// body ends the request phase and returns r bounded by the idle timeout.
func (d *readDeadlines) body(r io.Reader) io.Reader {
	if d.request != nil {
		d.request.Stop()
	}
	if d.idle <= 0 {
		return r
	}
	cause := fmt.Errorf("%w: no data for %s", ErrReadIdleTimeout, d.idle)
	d.idleTimer = time.AfterFunc(d.idle, func() { d.cancel(cause) })
	return &idleReader{r: r, timer: d.idleTimer, idle: d.idle}
}

// explain adds the reason to err when one of the timers ended the call and
// err does not already carry it. The SDK reports a canceled request as
// context.Canceled (smithy-go transport/http ClientHandler wraps ctx.Err(),
// not the cause), which says nothing about which bound was hit; net/http
// already returns the cause from a body read, so that error is left as is.
func (d *readDeadlines) explain(err error) error {
	cause := context.Cause(d.ctx)
	if !errors.Is(cause, errRequestTimeout) && !errors.Is(cause, ErrReadIdleTimeout) {
		return err
	}
	if errors.Is(err, cause) {
		return err
	}
	return fmt.Errorf("%w: %w", cause, err)
}

// stop releases the timers and the context.
func (d *readDeadlines) stop() {
	if d.request != nil {
		d.request.Stop()
	}
	if d.idleTimer != nil {
		d.idleTimer.Stop()
	}
	d.cancel(nil)
}

// idleReader restarts the idle timer on every read that returns bytes.
type idleReader struct {
	r     io.Reader
	timer *time.Timer
	idle  time.Duration
}

func (ir *idleReader) Read(p []byte) (int, error) {
	n, err := ir.r.Read(p)
	if n > 0 {
		ir.timer.Reset(ir.idle)
	}
	return n, err
}
