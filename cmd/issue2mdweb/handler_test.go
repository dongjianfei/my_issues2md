package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dongjianfei/issue2md/internal/cli"
)

// mockConvertSuccess 模拟成功的转换
func mockConvertSuccess(w io.Writer, opts *cli.RunOptions) error {
	_, _ = io.WriteString(w, "---\ntitle: \"Test Issue\"\ntype: issue\n---\n# Test\n")
	return nil
}

func TestNewServer(t *testing.T) {
	s := NewServer(mockConvertSuccess)
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.mux == nil {
		t.Fatal("Server.mux is nil")
	}
	if s.templates == nil {
		t.Fatal("Server.templates is nil")
	}
}

func TestRoutes(t *testing.T) {
	s := NewServer(mockConvertSuccess)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "GET / returns 200",
			method:     http.MethodGet,
			path:       "/",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET /convert returns 405",
			method:     http.MethodGet,
			path:       "/convert",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandleIndex(t *testing.T) {
	s := NewServer(mockConvertSuccess)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	wants := []string{
		"<form",
		"<textarea",
		`name="urls"`,
		`name="enable_reactions"`,
		`name="enable_user_links"`,
		"开始转换",
	}
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("response body missing %q", want)
		}
	}
}

func TestParseURLList(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantURLs  []string
		wantError string
	}{
		{
			name:      "single valid URL",
			input:     "https://github.com/golang/go/issues/1",
			wantURLs:  []string{"https://github.com/golang/go/issues/1"},
			wantError: "",
		},
		{
			name:      "multiple URLs with blank lines",
			input:     "https://github.com/golang/go/issues/1\n\nhttps://github.com/cli/cli/pull/2\n  \n",
			wantURLs:  []string{"https://github.com/golang/go/issues/1", "https://github.com/cli/cli/pull/2"},
			wantError: "",
		},
		{
			name:      "duplicate URLs deduplicated",
			input:     "https://github.com/golang/go/issues/1\nhttps://github.com/golang/go/issues/1",
			wantURLs:  []string{"https://github.com/golang/go/issues/1"},
			wantError: "",
		},
		{
			name:      "empty input",
			input:     "",
			wantURLs:  nil,
			wantError: "URL 列表不能为空",
		},
		{
			name:      "whitespace only",
			input:     "  \n  \n  ",
			wantURLs:  nil,
			wantError: "URL 列表不能为空",
		},
		{
			name:      "exceeds 20 URL limit",
			input:     generateNURLs(21),
			wantURLs:  nil,
			wantError: "最多支持 20 个 URL",
		},
		{
			name:      "exactly 20 URLs",
			input:     generateNURLs(20),
			wantURLs:  generateNURLSlice(20),
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := parseURLList(tt.input)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantError)
				}
				if err.Error() != tt.wantError {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(urls) != len(tt.wantURLs) {
				t.Fatalf("got %d URLs, want %d", len(urls), len(tt.wantURLs))
			}
			for i, u := range urls {
				if u != tt.wantURLs[i] {
					t.Errorf("url[%d] = %q, want %q", i, u, tt.wantURLs[i])
				}
			}
		})
	}
}

func generateNURLs(n int) string {
	var s string
	for i := 1; i <= n; i++ {
		if i > 1 {
			s += "\n"
		}
		s += fmt.Sprintf("https://github.com/owner/repo/issues/%d", i)
	}
	return s
}

func generateNURLSlice(n int) []string {
	urls := make([]string, n)
	for i := 0; i < n; i++ {
		urls[i] = fmt.Sprintf("https://github.com/owner/repo/issues/%d", i+1)
	}
	return urls
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "issue URL",
			url:  "https://github.com/golang/go/issues/1",
			want: "golang_go_issue_1.md",
		},
		{
			name: "PR URL",
			url:  "https://github.com/cli/cli/pull/1234",
			want: "cli_cli_pr_1234.md",
		},
		{
			name: "discussion URL",
			url:  "https://github.com/vercel/next.js/discussions/48427",
			want: "vercel_next.js_discussion_48427.md",
		},
		{
			name: "invalid URL fallback",
			url:  "not-a-url",
			want: "output.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFilename(tt.url)
			if got != tt.want {
				t.Errorf("generateFilename(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
