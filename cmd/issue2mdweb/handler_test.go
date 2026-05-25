package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
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

// mockConvertError 模拟失败的转换
func mockConvertError(w io.Writer, opts *cli.RunOptions) error {
	return fmt.Errorf("fetch issue: API request failed with status 404")
}

// mockConvertByURL 根据 URL 决定成功或失败
func mockConvertByURL(w io.Writer, opts *cli.RunOptions) error {
	if strings.Contains(opts.URL, "fail") {
		return fmt.Errorf("fetch issue: API request failed with status 404")
	}
	_, _ = io.WriteString(w, "---\ntitle: \"Test\"\ntype: issue\n---\n# Content for "+opts.URL+"\n")
	return nil
}

// sseEvent 表示一个解析后的 SSE 事件
type sseEvent struct {
	Event string
	Data  string
}

// parseSSEEvents 从响应 body 中解析 SSE 事件
func parseSSEEvents(body string) []sseEvent {
	var events []sseEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	var currentEvent sseEvent

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentEvent.Data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && currentEvent.Event != "" {
			events = append(events, currentEvent)
			currentEvent = sseEvent{}
		}
	}
	if currentEvent.Event != "" {
		events = append(events, currentEvent)
	}

	return events
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

func TestHandleConvert(t *testing.T) {
	tests := []struct {
		name            string
		convertFn       convertFunc
		formURLs        string
		enableReactions string
		enableUserLinks string
		wantEvents      []string
		wantStatus      int
	}{
		{
			name:       "empty URL list",
			convertFn:  mockConvertSuccess,
			formURLs:   "",
			wantEvents: []string{"error"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "single success",
			convertFn:  mockConvertSuccess,
			formURLs:   "https://github.com/golang/go/issues/1",
			wantEvents: []string{"start", "result", "done"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "single failure",
			convertFn:  mockConvertError,
			formURLs:   "https://github.com/golang/go/issues/1",
			wantEvents: []string{"start", "result", "done"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "mixed success and failure",
			convertFn:  mockConvertByURL,
			formURLs:   "https://github.com/golang/go/issues/1\nhttps://github.com/fail/repo/issues/2\nhttps://github.com/cli/cli/pull/3",
			wantEvents: []string{"start", "result", "result", "result", "done"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "duplicate URLs deduplicated",
			convertFn:  mockConvertSuccess,
			formURLs:   "https://github.com/golang/go/issues/1\nhttps://github.com/golang/go/issues/1",
			wantEvents: []string{"start", "result", "done"},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(tt.convertFn)

			form := strings.NewReader("urls=" + strings.ReplaceAll(tt.formURLs, "\n", "%0A") +
				"&enable_reactions=" + tt.enableReactions +
				"&enable_user_links=" + tt.enableUserLinks)
			req := httptest.NewRequest(http.MethodPost, "/convert", form)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			s.mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tt.wantStatus)
			}

			events := parseSSEEvents(rec.Body.String())
			if len(events) != len(tt.wantEvents) {
				t.Fatalf("got %d events, want %d.\nBody:\n%s", len(events), len(tt.wantEvents), rec.Body.String())
			}
			for i, e := range events {
				if e.Event != tt.wantEvents[i] {
					t.Errorf("event[%d] = %q, want %q", i, e.Event, tt.wantEvents[i])
				}
			}
		})
	}
}

func TestHandleConvertResultContent(t *testing.T) {
	s := NewServer(mockConvertByURL)

	form := strings.NewReader("urls=https%3A%2F%2Fgithub.com%2Fgolang%2Fgo%2Fissues%2F1%0Ahttps%3A%2F%2Fgithub.com%2Ffail%2Frepo%2Fissues%2F2")
	req := httptest.NewRequest(http.MethodPost, "/convert", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	s.mux.ServeHTTP(rec, req)

	events := parseSSEEvents(rec.Body.String())

	// start event
	var startData map[string]int
	if err := json.Unmarshal([]byte(events[0].Data), &startData); err != nil {
		t.Fatalf("failed to parse start event data: %v", err)
	}
	if startData["total"] != 2 {
		t.Errorf("start total = %d, want 2", startData["total"])
	}

	// first result: success
	var result1 ConvertResult
	if err := json.Unmarshal([]byte(events[1].Data), &result1); err != nil {
		t.Fatalf("failed to parse result 1: %v", err)
	}
	if !result1.Success {
		t.Error("result 1 should be success")
	}
	if result1.Filename != "golang_go_issue_1.md" {
		t.Errorf("result 1 filename = %q, want %q", result1.Filename, "golang_go_issue_1.md")
	}
	if result1.Index != 1 {
		t.Errorf("result 1 index = %d, want 1", result1.Index)
	}

	// second result: failure
	var result2 ConvertResult
	if err := json.Unmarshal([]byte(events[2].Data), &result2); err != nil {
		t.Fatalf("failed to parse result 2: %v", err)
	}
	if result2.Success {
		t.Error("result 2 should be failure")
	}
	if result2.Error == "" {
		t.Error("result 2 should have error message")
	}

	// done event
	var doneData map[string]int
	if err := json.Unmarshal([]byte(events[3].Data), &doneData); err != nil {
		t.Fatalf("failed to parse done event data: %v", err)
	}
	if doneData["completed"] != 1 {
		t.Errorf("done completed = %d, want 1", doneData["completed"])
	}
	if doneData["failed"] != 1 {
		t.Errorf("done failed = %d, want 1", doneData["failed"])
	}
}

func TestHandleConvertOptionsPassthrough(t *testing.T) {
	var capturedOpts *cli.RunOptions
	captureFn := func(w io.Writer, opts *cli.RunOptions) error {
		capturedOpts = opts
		_, _ = io.WriteString(w, "---\ntitle: Test\n---\n")
		return nil
	}

	s := NewServer(captureFn)
	form := strings.NewReader("urls=https%3A%2F%2Fgithub.com%2Fgolang%2Fgo%2Fissues%2F1&enable_reactions=on&enable_user_links=on")
	req := httptest.NewRequest(http.MethodPost, "/convert", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	s.mux.ServeHTTP(rec, req)

	if capturedOpts == nil {
		t.Fatal("convertFunc was not called")
	}
	if !capturedOpts.EnableReactions {
		t.Error("EnableReactions should be true")
	}
	if !capturedOpts.EnableUserLinks {
		t.Error("EnableUserLinks should be true")
	}
}

func TestHandleConvertContentType(t *testing.T) {
	s := NewServer(mockConvertSuccess)
	form := strings.NewReader("urls=https%3A%2F%2Fgithub.com%2Fgolang%2Fgo%2Fissues%2F1")
	req := httptest.NewRequest(http.MethodPost, "/convert", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	s.mux.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
	}
}

func TestHandleDownload(t *testing.T) {
	tests := []struct {
		name         string
		formFilename string
		formContent  string
		wantStatus   int
		wantFilename string
		wantBody     string
	}{
		{
			name:         "successful download",
			formFilename: "golang_go_issue_1.md",
			formContent:  "---\ntitle: Test\n---\n# Content",
			wantStatus:   http.StatusOK,
			wantFilename: `attachment; filename="golang_go_issue_1.md"`,
			wantBody:     "---\ntitle: Test\n---\n# Content",
		},
		{
			name:         "empty filename",
			formFilename: "",
			formContent:  "some content",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "empty content",
			formFilename: "test.md",
			formContent:  "",
			wantStatus:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(mockConvertSuccess)
			form := fmt.Sprintf("filename=%s&content=%s",
				strings.ReplaceAll(tt.formFilename, " ", "+"),
				strings.ReplaceAll(strings.ReplaceAll(tt.formContent, "\n", "%0A"), "#", "%23"))
			req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			s.mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				cd := rec.Header().Get("Content-Disposition")
				if cd != tt.wantFilename {
					t.Errorf("Content-Disposition = %q, want %q", cd, tt.wantFilename)
				}
				if rec.Body.String() != tt.wantBody {
					t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
				}
			}
		})
	}
}

func TestHandleDownloadAll(t *testing.T) {
	tests := []struct {
		name       string
		filesJSON  string
		wantStatus int
		wantFiles  map[string]string
	}{
		{
			name:       "two files",
			filesJSON:  `[{"filename":"a.md","content":"# A"},{"filename":"b.md","content":"# B"}]`,
			wantStatus: http.StatusOK,
			wantFiles:  map[string]string{"a.md": "# A", "b.md": "# B"},
		},
		{
			name:       "empty list",
			filesJSON:  `[]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			filesJSON:  `not-json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing files field",
			filesJSON:  "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(mockConvertSuccess)
			form := "files=" + strings.ReplaceAll(
				strings.ReplaceAll(tt.filesJSON, `"`, "%22"),
				" ", "+")
			req := httptest.NewRequest(http.MethodPost, "/download-all", strings.NewReader(form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			s.mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("got status %d, want %d.\nBody: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				ct := rec.Header().Get("Content-Type")
				if ct != "application/zip" {
					t.Errorf("Content-Type = %q, want application/zip", ct)
				}

				cd := rec.Header().Get("Content-Disposition")
				if cd != `attachment; filename="issue2md-export.zip"` {
					t.Errorf("Content-Disposition = %q", cd)
				}

				zipReader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
				if err != nil {
					t.Fatalf("failed to read zip: %v", err)
				}

				gotFiles := make(map[string]string)
				for _, f := range zipReader.File {
					rc, err := f.Open()
					if err != nil {
						t.Fatalf("failed to open zip entry %q: %v", f.Name, err)
					}
					content, _ := io.ReadAll(rc)
					rc.Close()
					gotFiles[f.Name] = string(content)
				}

				for wantName, wantContent := range tt.wantFiles {
					gotContent, ok := gotFiles[wantName]
					if !ok {
						t.Errorf("zip missing file %q", wantName)
						continue
					}
					if gotContent != wantContent {
						t.Errorf("file %q content = %q, want %q", wantName, gotContent, wantContent)
					}
				}
			}
		})
	}
}
