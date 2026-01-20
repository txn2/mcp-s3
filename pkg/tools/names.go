// Package tools provides MCP tool implementations for S3 operations.
package tools

// ToolName is a type-safe representation of S3 MCP tool names.
type ToolName string

// Tool name constants for all S3 MCP tools.
const (
	// ToolListBuckets lists all accessible S3 buckets.
	ToolListBuckets ToolName = "s3_list_buckets"

	// ToolListObjects lists objects in a bucket with optional prefix/delimiter.
	ToolListObjects ToolName = "s3_list_objects"

	// ToolGetObject retrieves object content from S3.
	ToolGetObject ToolName = "s3_get_object"

	// ToolGetObjectMetadata retrieves object metadata without downloading content.
	ToolGetObjectMetadata ToolName = "s3_get_object_metadata"

	// ToolPutObject uploads an object to S3.
	ToolPutObject ToolName = "s3_put_object"

	// ToolDeleteObject deletes an object from S3.
	ToolDeleteObject ToolName = "s3_delete_object"

	// ToolCopyObject copies an object within or between buckets.
	ToolCopyObject ToolName = "s3_copy_object"

	// ToolPresignURL generates presigned URLs for GET or PUT operations.
	ToolPresignURL ToolName = "s3_presign_url"

	// ToolListConnections lists configured S3 connections.
	ToolListConnections ToolName = "s3_list_connections"
)

// String returns the string representation of the tool name.
func (t ToolName) String() string {
	return string(t)
}

// AllTools returns a list of all tool names.
func AllTools() []ToolName {
	return []ToolName{
		ToolListBuckets,
		ToolListObjects,
		ToolGetObject,
		ToolGetObjectMetadata,
		ToolPutObject,
		ToolDeleteObject,
		ToolCopyObject,
		ToolPresignURL,
		ToolListConnections,
	}
}

// WriteTools returns a list of tool names that perform write operations.
func WriteTools() []ToolName {
	return []ToolName{
		ToolPutObject,
		ToolDeleteObject,
		ToolCopyObject,
	}
}

// ReadTools returns a list of tool names that perform read operations.
func ReadTools() []ToolName {
	return []ToolName{
		ToolListBuckets,
		ToolListObjects,
		ToolGetObject,
		ToolGetObjectMetadata,
		ToolPresignURL,
		ToolListConnections,
	}
}

// IsWriteTool returns true if the tool name is a write operation.
func IsWriteTool(name ToolName) bool {
	switch name {
	case ToolPutObject, ToolDeleteObject, ToolCopyObject:
		return true
	default:
		return false
	}
}
