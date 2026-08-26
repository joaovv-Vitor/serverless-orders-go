package observability

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewJSONLoggerUsesStandardFields(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := NewJSONLogger(&output)
	logger.Info("order accepted", "service", "create-order", "orderId", "order-123")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}
	if entry["level"] != "INFO" {
		t.Fatalf("level = %#v, want INFO", entry["level"])
	}
	if entry["message"] != "order accepted" {
		t.Fatalf("message = %#v", entry["message"])
	}
	if _, exists := entry["msg"]; exists {
		t.Fatalf("legacy msg field is present: %#v", entry)
	}
	if entry["service"] != "create-order" || entry["orderId"] != "order-123" {
		t.Fatalf("correlation fields = %#v", entry)
	}
}
