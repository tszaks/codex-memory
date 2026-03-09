package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, map[string]string{"hello": "world"}, true, func() string { return "" }); err != nil {
		t.Fatalf("Write JSON failed: %v", err)
	}
	if !strings.Contains(buf.String(), `"hello": "world"`) {
		t.Fatalf("expected JSON output, got %q", buf.String())
	}
}

func TestWriteText(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, nil, false, func() string { return "hello" }); err != nil {
		t.Fatalf("Write text failed: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "hello" {
		t.Fatalf("expected text output, got %q", buf.String())
	}
}
