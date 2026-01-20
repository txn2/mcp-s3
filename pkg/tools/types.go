package tools

// ListBucketsInput defines the input parameters for the list_buckets tool.
type ListBucketsInput struct {
	Connection string `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// ListObjectsInput defines the input parameters for the list_objects tool.
type ListObjectsInput struct {
	Bucket            string `json:"bucket" jsonschema_description:"Name of the S3 bucket to list objects from."`
	Prefix            string `json:"prefix,omitempty" jsonschema_description:"Filter objects by key prefix. Only objects with keys starting with this prefix are returned."`
	Delimiter         string `json:"delimiter,omitempty" jsonschema_description:"Character used to group keys. Commonly '/' to simulate folders. Common prefixes are returned separately."`
	MaxKeys           int32  `json:"max_keys,omitempty" jsonschema_description:"Maximum number of objects to return (1-1000). Default: 1000."`
	ContinuationToken string `json:"continuation_token,omitempty" jsonschema_description:"Token from a previous response to continue listing from where you left off."`
	Connection        string `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// GetObjectInput defines the input parameters for the get_object tool.
type GetObjectInput struct {
	Bucket     string `json:"bucket" jsonschema_description:"Name of the S3 bucket containing the object."`
	Key        string `json:"key" jsonschema_description:"Key (path) of the object to retrieve."`
	Connection string `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// GetObjectMetadataInput defines the input parameters for the get_object_metadata tool.
type GetObjectMetadataInput struct {
	Bucket     string `json:"bucket" jsonschema_description:"Name of the S3 bucket containing the object."`
	Key        string `json:"key" jsonschema_description:"Key (path) of the object to get metadata for."`
	Connection string `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// PutObjectInput defines the input parameters for the put_object tool.
type PutObjectInput struct {
	Bucket      string            `json:"bucket" jsonschema_description:"Name of the S3 bucket to upload to."`
	Key         string            `json:"key" jsonschema_description:"Key (path) for the object in the bucket."`
	Content     string            `json:"content" jsonschema_description:"Content to upload. For text, provide directly. For binary, provide base64-encoded content."`
	ContentType string            `json:"content_type,omitempty" jsonschema_description:"MIME type of the content (e.g., 'text/plain', 'application/json'). Defaults to 'application/octet-stream'."`
	IsBase64    bool              `json:"is_base64,omitempty" jsonschema_description:"Set to true if the content is base64-encoded binary data."`
	Metadata    map[string]string `json:"metadata,omitempty" jsonschema_description:"Custom metadata key-value pairs to attach to the object."`
	Connection  string            `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// DeleteObjectInput defines the input parameters for the delete_object tool.
type DeleteObjectInput struct {
	Bucket     string `json:"bucket" jsonschema_description:"Name of the S3 bucket containing the object to delete."`
	Key        string `json:"key" jsonschema_description:"Key (path) of the object to delete."`
	Connection string `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// CopyObjectInput defines the input parameters for the copy_object tool.
type CopyObjectInput struct {
	SourceBucket string            `json:"source_bucket" jsonschema_description:"Name of the source S3 bucket."`
	SourceKey    string            `json:"source_key" jsonschema_description:"Key (path) of the source object."`
	DestBucket   string            `json:"dest_bucket" jsonschema_description:"Name of the destination S3 bucket."`
	DestKey      string            `json:"dest_key" jsonschema_description:"Key (path) for the destination object."`
	Metadata     map[string]string `json:"metadata,omitempty" jsonschema_description:"New metadata to assign to the copied object. If provided, replaces source metadata."`
	Connection   string            `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// PresignURLInput defines the input parameters for the presign_url tool.
type PresignURLInput struct {
	Bucket     string `json:"bucket" jsonschema_description:"Name of the S3 bucket containing the object."`
	Key        string `json:"key" jsonschema_description:"Key (path) of the object to generate a URL for."`
	Method     string `json:"method,omitempty" jsonschema_description:"HTTP method for the presigned URL. 'GET' for downloads, 'PUT' for uploads. Default: 'GET'."`
	ExpiresIn  int    `json:"expires_in,omitempty" jsonschema_description:"URL expiration time in seconds. Default: 3600 (1 hour). Maximum: 604800 (7 days)."`
	Connection string `json:"connection,omitempty" jsonschema_description:"Name of the S3 connection to use. If not specified, uses the default connection."`
}

// ListConnectionsInput defines the input parameters for the list_connections tool.
type ListConnectionsInput struct {
	// No parameters required
}
