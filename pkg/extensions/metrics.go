package extensions

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-s3/pkg/tools"
)

// Metrics tracks tool usage statistics.
type Metrics struct {
	// Tool-level metrics
	toolCalls   map[string]*atomic.Int64
	toolErrors  map[string]*atomic.Int64
	toolLatency map[string]*latencyTracker

	mu sync.RWMutex
}

// latencyTracker tracks latency statistics.
type latencyTracker struct {
	count   atomic.Int64
	totalNs atomic.Int64
	minNs   atomic.Int64
	maxNs   atomic.Int64
}

// NewMetrics creates a new metrics tracker.
func NewMetrics() *Metrics {
	return &Metrics{
		toolCalls:   make(map[string]*atomic.Int64),
		toolErrors:  make(map[string]*atomic.Int64),
		toolLatency: make(map[string]*latencyTracker),
	}
}

// getOrCreateCounter returns or creates a counter for the given tool.
func (m *Metrics) getOrCreateCounter(counters map[string]*atomic.Int64, tool string) *atomic.Int64 {
	m.mu.RLock()
	counter, ok := counters[tool]
	m.mu.RUnlock()

	if ok {
		return counter
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if counter, ok := counters[tool]; ok {
		return counter
	}

	counter = &atomic.Int64{}
	counters[tool] = counter
	return counter
}

// getOrCreateLatencyTracker returns or creates a latency tracker for the given tool.
func (m *Metrics) getOrCreateLatencyTracker(tool string) *latencyTracker {
	m.mu.RLock()
	tracker, ok := m.toolLatency[tool]
	m.mu.RUnlock()

	if ok {
		return tracker
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if tracker, ok := m.toolLatency[tool]; ok {
		return tracker
	}

	tracker = &latencyTracker{}
	m.toolLatency[tool] = tracker
	return tracker
}

// RecordCall records a tool call.
func (m *Metrics) RecordCall(tool string, duration time.Duration, isError bool) {
	// Increment call counter
	m.getOrCreateCounter(m.toolCalls, tool).Add(1)

	// Increment error counter if error
	if isError {
		m.getOrCreateCounter(m.toolErrors, tool).Add(1)
	}

	// Record latency
	tracker := m.getOrCreateLatencyTracker(tool)
	ns := duration.Nanoseconds()
	tracker.count.Add(1)
	tracker.totalNs.Add(ns)

	// Update min (using CAS loop)
	for {
		min := tracker.minNs.Load()
		if min != 0 && min <= ns {
			break
		}
		if tracker.minNs.CompareAndSwap(min, ns) {
			break
		}
	}

	// Update max (using CAS loop)
	for {
		max := tracker.maxNs.Load()
		if max >= ns {
			break
		}
		if tracker.maxNs.CompareAndSwap(max, ns) {
			break
		}
	}
}

// ToolStats returns statistics for a specific tool.
type ToolStats struct {
	Calls      int64   `json:"calls"`
	Errors     int64   `json:"errors"`
	ErrorRate  float64 `json:"error_rate"`
	AvgLatency float64 `json:"avg_latency_ms"`
	MinLatency float64 `json:"min_latency_ms"`
	MaxLatency float64 `json:"max_latency_ms"`
}

// GetToolStats returns statistics for a specific tool.
func (m *Metrics) GetToolStats(tool string) ToolStats {
	calls := m.getOrCreateCounter(m.toolCalls, tool).Load()
	errors := m.getOrCreateCounter(m.toolErrors, tool).Load()

	tracker := m.getOrCreateLatencyTracker(tool)
	count := tracker.count.Load()
	totalNs := tracker.totalNs.Load()

	stats := ToolStats{
		Calls:  calls,
		Errors: errors,
	}

	if calls > 0 {
		stats.ErrorRate = float64(errors) / float64(calls)
	}

	if count > 0 {
		stats.AvgLatency = float64(totalNs) / float64(count) / 1e6 // Convert to ms
		stats.MinLatency = float64(tracker.minNs.Load()) / 1e6
		stats.MaxLatency = float64(tracker.maxNs.Load()) / 1e6
	}

	return stats
}

// GetAllStats returns statistics for all tools.
func (m *Metrics) GetAllStats() map[string]ToolStats {
	m.mu.RLock()
	toolNames := make([]string, 0, len(m.toolCalls))
	for name := range m.toolCalls {
		toolNames = append(toolNames, name)
	}
	m.mu.RUnlock()

	result := make(map[string]ToolStats, len(toolNames))
	for _, name := range toolNames {
		result[name] = m.GetToolStats(name)
	}

	return result
}

// MetricsMiddleware tracks metrics for tool operations.
type MetricsMiddleware struct {
	metrics *Metrics
}

// NewMetricsMiddleware creates a new metrics middleware.
func NewMetricsMiddleware(metrics *Metrics) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: metrics,
	}
}

// Name returns the middleware name.
func (m *MetricsMiddleware) Name() string {
	return "metrics"
}

// Before is a no-op for metrics; start time is tracked in ToolContext.
func (m *MetricsMiddleware) Before(ctx context.Context, tc *tools.ToolContext) (context.Context, error) {
	return ctx, nil
}

// After records metrics for the tool call.
func (m *MetricsMiddleware) After(ctx context.Context, tc *tools.ToolContext, result *mcp.CallToolResult, handlerErr error) (*mcp.CallToolResult, error) {
	duration := time.Since(tc.StartTime)
	isError := handlerErr != nil || (result != nil && result.IsError)

	m.metrics.RecordCall(string(tc.ToolName), duration, isError)

	return result, handlerErr
}

// Ensure MetricsMiddleware implements ToolMiddleware.
var _ tools.ToolMiddleware = (*MetricsMiddleware)(nil)
