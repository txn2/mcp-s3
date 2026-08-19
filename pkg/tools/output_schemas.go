package tools

// nullableArray returns an array schema that also admits JSON null.
//
// A nil Go slice marshals as null, and null is not an "array". That matters
// beyond empty results: the SDK's typed-handler wrapper validates output even
// for error results, substituting the zero value of the result struct whenever
// a handler returns an error result with no typed output. Rejecting null there
// turns every failed call into a JSON-RPC output-validation error and discards
// the tool's own error message (issue #141).
func nullableArray(items any) map[string]any {
	return map[string]any{
		"type":  []string{"array", "null"},
		"items": items,
	}
}

// nullableObject returns an object schema that also admits JSON null, for
// properties backed by a Go map. See nullableArray for why null is admitted.
func nullableObject(additionalProperties any) map[string]any {
	return map[string]any{
		"type":                 []string{"object", "null"},
		"additionalProperties": additionalProperties,
	}
}

// defaultOutputSchemas holds the default JSON Schema (2020-12) for each tool's
// structured output. Schemas use map[string]any so they can be remaarshaleld by
// the MCP SDK's schema resolution pipeline.
//
// All schemas declare only "type" and "properties" — no "required" constraints
// and no "additionalProperties": false — so that partial results and
// implementation-specific fields never fail schema validation at runtime.
//
// Properties backed by a Go slice or map are declared with nullableArray and
// nullableObject rather than a bare "array"/"object" type. See those functions
// for why null must be admitted.
var defaultOutputSchemas = map[ToolName]any{
	ToolListBuckets: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"buckets": nullableArray(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":          map[string]any{"type": "string"},
					"creation_date": map[string]any{"type": "string"},
				},
			}),
			"count": map[string]any{"type": "integer"},
		},
	},

	ToolListObjects: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket":    map[string]any{"type": "string"},
			"prefix":    map[string]any{"type": "string"},
			"delimiter": map[string]any{"type": "string"},
			"objects": nullableArray(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key":           map[string]any{"type": "string"},
					"size":          map[string]any{"type": "integer"},
					"last_modified": map[string]any{"type": "string"},
					"etag":          map[string]any{"type": "string"},
					"storage_class": map[string]any{"type": "string"},
				},
			}),
			"common_prefixes":         nullableArray(map[string]any{"type": "string"}),
			"count":                   map[string]any{"type": "integer"},
			"is_truncated":            map[string]any{"type": "boolean"},
			"next_continuation_token": map[string]any{"type": "string"},
		},
	},

	ToolGetObject: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket":        map[string]any{"type": "string"},
			"key":           map[string]any{"type": "string"},
			"size":          map[string]any{"type": "integer"},
			"content_type":  map[string]any{"type": "string"},
			"last_modified": map[string]any{"type": "string"},
			"etag":          map[string]any{"type": "string"},
			"metadata":      nullableObject(map[string]any{"type": "string"}),
			"content":       map[string]any{"type": "string"},
			"is_base64":     map[string]any{"type": "boolean"},
			"truncated":     map[string]any{"type": "boolean"},
		},
	},

	ToolGetObjectMetadata: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket":         map[string]any{"type": "string"},
			"key":            map[string]any{"type": "string"},
			"size":           map[string]any{"type": "integer"},
			"content_type":   map[string]any{"type": "string"},
			"content_length": map[string]any{"type": "integer"},
			"last_modified":  map[string]any{"type": "string"},
			"etag":           map[string]any{"type": "string"},
			"metadata":       nullableObject(map[string]any{"type": "string"}),
		},
	},

	ToolPutObject: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket":     map[string]any{"type": "string"},
			"key":        map[string]any{"type": "string"},
			"size":       map[string]any{"type": "integer"},
			"etag":       map[string]any{"type": "string"},
			"version_id": map[string]any{"type": "string"},
		},
	},

	ToolDeleteObject: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket":  map[string]any{"type": "string"},
			"key":     map[string]any{"type": "string"},
			"deleted": map[string]any{"type": "boolean"},
		},
	},

	ToolCopyObject: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source_bucket": map[string]any{"type": "string"},
			"source_key":    map[string]any{"type": "string"},
			"dest_bucket":   map[string]any{"type": "string"},
			"dest_key":      map[string]any{"type": "string"},
			"etag":          map[string]any{"type": "string"},
			"last_modified": map[string]any{"type": "string"},
			"version_id":    map[string]any{"type": "string"},
		},
	},

	ToolPresignURL: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bucket":             map[string]any{"type": "string"},
			"key":                map[string]any{"type": "string"},
			"url":                map[string]any{"type": "string"},
			"method":             map[string]any{"type": "string"},
			"expires_in_seconds": map[string]any{"type": "integer"},
			"expires_at":         map[string]any{"type": "string"},
		},
	},

	ToolListConnections: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connections": nullableArray(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":     map[string]any{"type": "string"},
					"region":   map[string]any{"type": "string"},
					"endpoint": map[string]any{"type": "string"},
				},
			}),
			"default_connection": map[string]any{"type": "string"},
			"count":              map[string]any{"type": "integer"},
		},
	},
}

// DefaultOutputSchema returns the default JSON Schema for a tool's structured output.
// Returns nil for unknown tool names.
func DefaultOutputSchema(name ToolName) any {
	return defaultOutputSchemas[name]
}

// getOutputSchema resolves the output schema for a tool using the priority chain:
// 1. Per-registration override (cfg.outputSchema) — highest priority
// 2. Toolkit-level override (t.outputSchemas) — medium priority
// 3. Default output schema — lowest priority.
func (t *Toolkit) getOutputSchema(name ToolName, cfg *toolConfig) any {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.outputSchema != nil {
		return cfg.outputSchema
	}

	// Toolkit-level override (medium priority)
	if schema, ok := t.outputSchemas[name]; ok {
		return schema
	}

	// Default output schema (lowest priority)
	return defaultOutputSchemas[name]
}
