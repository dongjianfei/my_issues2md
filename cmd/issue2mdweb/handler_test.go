package main

import (
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
