package tools

import "testing"

func TestWriteTools(t *testing.T) {
	writeTools := WriteTools()

	// Should include known write tools
	expectedTools := []ToolName{ToolPutObject, ToolDeleteObject}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range writeTools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("WriteTools() missing %s", expected)
		}
	}

	// Should not include read-only tools
	readOnlyTools := []ToolName{ToolListBuckets, ToolListObjects, ToolGetObject}
	for _, readOnly := range readOnlyTools {
		for _, tool := range writeTools {
			if tool == readOnly {
				t.Errorf("WriteTools() should not include %s", readOnly)
			}
		}
	}
}

func TestReadTools(t *testing.T) {
	readTools := ReadTools()

	// Should include known read tools
	expectedTools := []ToolName{ToolListBuckets, ToolListObjects, ToolGetObject, ToolGetObjectMetadata}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range readTools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ReadTools() missing %s", expected)
		}
	}

	// Should not include write tools
	writeOnlyTools := []ToolName{ToolPutObject, ToolDeleteObject}
	for _, writeOnly := range writeOnlyTools {
		for _, tool := range readTools {
			if tool == writeOnly {
				t.Errorf("ReadTools() should not include %s", writeOnly)
			}
		}
	}
}

func TestIsWriteTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName ToolName
		want     bool
	}{
		{"put object is write", ToolPutObject, true},
		{"delete object is write", ToolDeleteObject, true},
		{"list buckets is not write", ToolListBuckets, false},
		{"get object is not write", ToolGetObject, false},
		{"list objects is not write", ToolListObjects, false},
		{"copy object is write", ToolCopyObject, true},
		{"unknown tool is not write", ToolName("unknown_tool"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWriteTool(tt.toolName); got != tt.want {
				t.Errorf("IsWriteTool(%s) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}
