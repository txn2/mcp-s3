// Package tools provides MCP tool implementations for S3 operations.
package tools

// Tool name constants for all S3 MCP tools.
const (
	// ToolListBuckets lists all accessible S3 buckets.
	ToolListBuckets = "s3_list_buckets"

	// ToolListObjects lists objects in a bucket with optional prefix/delimiter.
	ToolListObjects = "s3_list_objects"

	// ToolGetObject retrieves object content from S3.
	ToolGetObject = "s3_get_object"

	// ToolGetObjectMetadata retrieves object metadata without downloading content.
	ToolGetObjectMetadata = "s3_get_object_metadata"

	// ToolPutObject uploads an object to S3.
	ToolPutObject = "s3_put_object"

	// ToolDeleteObject deletes an object from S3.
	ToolDeleteObject = "s3_delete_object"

	// ToolCopyObject copies an object within or between buckets.
	ToolCopyObject = "s3_copy_object"

	// ToolPresignURL generates presigned URLs for GET or PUT operations.
	ToolPresignURL = "s3_presign_url"

	// ToolListConnections lists configured S3 connections.
	ToolListConnections = "s3_list_connections"
)

// AllTools returns a list of all tool names.
func AllTools() []string {
	return []string{
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
func WriteTools() []string {
	return []string{
		ToolPutObject,
		ToolDeleteObject,
		ToolCopyObject,
	}
}

// ReadTools returns a list of tool names that perform read operations.
func ReadTools() []string {
	return []string{
		ToolListBuckets,
		ToolListObjects,
		ToolGetObject,
		ToolGetObjectMetadata,
		ToolPresignURL,
		ToolListConnections,
	}
}

// IsWriteTool returns true if the tool name is a write operation.
func IsWriteTool(name string) bool {
	switch name {
	case ToolPutObject, ToolDeleteObject, ToolCopyObject:
		return true
	default:
		return false
	}
}
