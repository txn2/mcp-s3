package tools

// defaultDescriptions holds the default description for each built-in tool.
// These are used when no override is provided via WithDescription or WithDescriptions.
var defaultDescriptions = map[ToolName]string{
	ToolListBuckets: "List all accessible S3 buckets. Returns bucket names and creation dates.",

	ToolListConnections: "List all configured S3 connections. Returns connection names, regions, " +
		"and endpoints (if custom endpoints are configured).",

	ToolListObjects: "List objects in an S3 bucket. Supports prefix filtering, delimiter for " +
		"folder simulation, and pagination.",

	ToolGetObject: "Retrieve the content of an S3 object. For text content, returns the content " +
		"directly. For binary content, returns base64-encoded data. Large objects may be " +
		"truncated based on size limits.",

	ToolGetObjectMetadata: "Get metadata for an S3 object without downloading its content. Returns " +
		"size, content type, last modified date, ETag, and custom metadata.",

	ToolPresignURL: "Generate a presigned URL for temporary access to an S3 object. The URL allows " +
		"temporary access without requiring AWS credentials. Supports both GET (download) " +
		"and PUT (upload) operations.",

	ToolPutObject: "Upload an object to S3. For text content, provide the content directly. For " +
		"binary content, provide base64-encoded content and set is_base64 to true. This " +
		"operation may be blocked in read-only mode.",

	ToolCopyObject: "Copy an object within S3, either within the same bucket or between different " +
		"buckets. Can optionally update metadata during the copy. This operation may be " +
		"blocked in read-only mode.",

	ToolDeleteObject: "Delete an object from S3. This operation is irreversible unless versioning " +
		"is enabled on the bucket. This operation may be blocked in read-only mode.",
}

// DefaultDescription returns the default description for a tool.
// Returns an empty string for unknown tool names.
func DefaultDescription(name ToolName) string {
	return defaultDescriptions[name]
}

// getDescription resolves the description for a tool using the priority chain:
// 1. Per-registration override (cfg.description) — highest priority
// 2. Toolkit-level override (t.descriptions) — medium priority
// 3. Default description — lowest priority.
func (t *Toolkit) getDescription(name ToolName, cfg *toolConfig) string {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.description != nil {
		return *cfg.description
	}

	// Toolkit-level override (medium priority)
	if desc, ok := t.descriptions[name]; ok {
		return desc
	}

	// Default description (lowest priority)
	return defaultDescriptions[name]
}
