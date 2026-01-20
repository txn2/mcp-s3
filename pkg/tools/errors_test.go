package tools

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestErrorResultf(t *testing.T) {
	result := ErrorResultf("error %d: %s", 42, "something went wrong")

	if !result.IsError {
		t.Error("ErrorResultf should set IsError to true")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	expected := "Error: error 42: something went wrong"
	if textContent.Text != expected {
		t.Errorf("text = %q, want %q", textContent.Text, expected)
	}
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult("something went wrong")

	if !result.IsError {
		t.Error("ErrorResult should set IsError to true")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	expected := "Error: something went wrong"
	if textContent.Text != expected {
		t.Errorf("text = %q, want %q", textContent.Text, expected)
	}
}

func TestTextResult(t *testing.T) {
	result := TextResult("hello world")

	if result.IsError {
		t.Error("TextResult should not set IsError")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	if textContent.Text != "hello world" {
		t.Errorf("text = %q, want %q", textContent.Text, "hello world")
	}
}

func TestJSONResult(t *testing.T) {
	t.Run("marshals valid data", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		result, err := JSONResult(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Error("JSONResult should not set IsError for valid data")
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content item, got %d", len(result.Content))
		}

		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("expected TextContent")
		}

		expected := "{\n  \"key\": \"value\"\n}"
		if textContent.Text != expected {
			t.Errorf("text = %q, want %q", textContent.Text, expected)
		}
	})

	t.Run("returns error for unmarshalable value", func(t *testing.T) {
		ch := make(chan int)
		_, err := JSONResult(ch)
		if err == nil {
			t.Error("JSONResult with unmarshalable value should return error")
		}
	})
}
