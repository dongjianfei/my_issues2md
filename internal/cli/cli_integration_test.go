// +build integration

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunIntegration(t *testing.T) {
	t.Run("valid issue URL produces non-empty output", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &RunOptions{URL: "https://github.com/golang/go/issues/1"}
		err := Run(&buf, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("expected non-empty output, got empty")
		}
		// Verify basic structure
		output := buf.String()
		if !strings.Contains(output, "---") {
			t.Error("output missing frontmatter")
		}
		if !strings.Contains(output, "type: \"issue\"") {
			t.Error("output missing type field")
		}
	})

	t.Run("valid PR URL produces non-empty output", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &RunOptions{URL: "https://github.com/cli/cli/pull/1"}
		err := Run(&buf, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("expected non-empty output, got empty")
		}
		output := buf.String()
		if !strings.Contains(output, "type: \"pull_request\"") {
			t.Error("output missing pull_request type")
		}
	})

	t.Run("valid discussion URL produces non-empty output", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &RunOptions{URL: "https://github.com/vercel/next.js/discussions/48427"}
		err := Run(&buf, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("expected non-empty output, got empty")
		}
		output := buf.String()
		if !strings.Contains(output, "type: \"discussion\"") {
			t.Error("output missing discussion type")
		}
	})
}
