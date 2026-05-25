# Web UI 批量转换服务 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 issue2md 添加 Web UI，支持 SSE 流式批量转换 GitHub URL 为 Markdown，并提供单个下载和 ZIP 打包下载。

**Architecture:** 薄 HTTP 适配层，通过 `Server` struct 注入 `convertFunc`（生产环境为 `cli.Run`，测试时为 mock）。所有代码放在 `cmd/issue2mdweb/`，不新增 internal 包，不修改任何现有文件。前端使用 `fetch` + `ReadableStream` 消费 SSE 流。

**Tech Stack:** Go 标准库（`net/http`、`html/template`、`embed`、`archive/zip`、`encoding/json`）、vanilla JS（无框架）

**TDD 铁律:** 每个 Task 严格遵循 Red-Green-Refactor 循环。先写失败测试，再写最小实现使测试通过，最后重构。绝不跳过任何一步。

---

## File Structure

| 文件 | 操作 | 职责 |
|------|------|------|
| `cmd/issue2mdweb/server.go` | 新建 | `convertFunc` 类型定义、`Server` struct、`NewServer()` 构造函数、路由注册、`go:embed` 模板加载 |
| `cmd/issue2mdweb/handler.go` | 新建 | `ConvertResult` 类型、`handleIndex`、`handleConvert`（SSE）、`handleDownload`、`handleDownloadAll`（ZIP）、URL 解析/去重/校验辅助函数、`generateFilename` |
| `cmd/issue2mdweb/handler_test.go` | 新建 | 全部表格驱动 httptest 测试，mock convertFunc |
| `cmd/issue2mdweb/templates/index.html` | 新建 | 单页 HTML 模板：表单 + 内联 CSS + vanilla JS（fetch SSE、结果渲染、下载） |
| `cmd/issue2mdweb/main.go` | 新建 | flag 解析（`-port`）、构造 Server、启动 `http.ListenAndServe` |

不修改的文件：`internal/cli/cli.go`、`internal/parser/*`、`internal/github/*`、`internal/converter/*`、`Makefile`

---

### Task 1: Server 骨架与路由注册

**Files:**
- Create: `cmd/issue2mdweb/server.go`
- Create: `cmd/issue2mdweb/handler.go` (空 handler 桩)
- Create: `cmd/issue2mdweb/handler_test.go`
- Create: `cmd/issue2mdweb/templates/index.html` (最小模板)

#### Red: 写失败测试

- [ ] **Step 1: 创建最小模板文件**

```bash
mkdir -p cmd/issue2mdweb/templates
```

写入 `cmd/issue2mdweb/templates/index.html`：

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><title>issue2md</title></head>
<body><h1>issue2md</h1></body>
</html>
```

- [ ] **Step 2: 写 Server 构造与路由测试**

写入 `cmd/issue2mdweb/handler_test.go`：

```go
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
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
```

- [ ] **Step 3: 运行测试，确认失败**

```bash
go test ./cmd/issue2mdweb/ -v -run "TestNewServer|TestRoutes"
```

预期：编译失败 — `NewServer` 未定义。

#### Green: 写最小实现

- [ ] **Step 4: 实现 server.go**

写入 `cmd/issue2mdweb/server.go`：

```go
package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"

	"github.com/dongjianfei/issue2md/internal/cli"
)

//go:embed templates/*
var templateFS embed.FS

// convertFunc 是核心依赖 — 与 cli.Run 签名一致
type convertFunc func(w io.Writer, opts *cli.RunOptions) error

// Server 持有所有依赖
type Server struct {
	convert   convertFunc
	templates *template.Template
	mux       *http.ServeMux
}

// NewServer 创建 Server 实例，注册路由
func NewServer(fn convertFunc) *Server {
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))

	s := &Server{
		convert:   fn,
		templates: tmpl,
		mux:       http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("POST /convert", s.handleConvert)
	s.mux.HandleFunc("POST /download", s.handleDownload)
	s.mux.HandleFunc("POST /download-all", s.handleDownloadAll)

	return s
}
```

- [ ] **Step 5: 实现空 handler 桩**

写入 `cmd/issue2mdweb/handler.go`：

```go
package main

import (
	"net/http"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "index.html", nil)
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) handleDownloadAll(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
```

- [ ] **Step 6: 运行测试，确认通过**

```bash
go test ./cmd/issue2mdweb/ -v -run "TestNewServer|TestRoutes"
```

预期：PASS

#### Refactor: 检查代码质量

- [ ] **Step 7: 审查 — 确认无全局变量（除 embed.FS）、无多余代码**

- [ ] **Step 8: 提交**

```bash
git add cmd/issue2mdweb/
git commit -m "feat(web): add Server skeleton with route registration and stub handlers"
```

---

### Task 2: handleIndex 表单渲染

**Files:**
- Modify: `cmd/issue2mdweb/templates/index.html`
- Modify: `cmd/issue2mdweb/handler_test.go`

#### Red: 写失败测试

- [ ] **Step 1: 添加 handleIndex 内容测试**

追加到 `cmd/issue2mdweb/handler_test.go`：

```go
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
		if !contains(body, want) {
			t.Errorf("response body missing %q", want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleIndex
```

预期：FAIL — body 中缺少 `<form`、`<textarea` 等元素。

#### Green: 写最小实现

- [ ] **Step 3: 更新 index.html 模板**

替换 `cmd/issue2mdweb/templates/index.html` 为完整表单模板（含内联 CSS，不含 JS — JS 在后续 Task 中添加）：

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>issue2md</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; color: #1a1a1a; background: #fff; }
  .header { background: #24292e; color: #fff; padding: 12px 24px; display: flex; align-items: center; gap: 12px; }
  .header h1 { font-size: 20px; }
  .header span { font-size: 13px; opacity: 0.7; }
  .form-section { padding: 24px; border-bottom: 1px solid #e1e4e8; }
  .form-section label.field-label { font-weight: 600; font-size: 14px; display: block; margin-bottom: 6px; }
  textarea { width: 100%; min-height: 120px; padding: 12px; font-family: monospace; font-size: 13px; border: 1px solid #d0d7de; border-radius: 6px; background: #f6f8fa; resize: vertical; }
  .options { display: flex; gap: 24px; align-items: center; margin: 16px 0; }
  .options label { font-size: 14px; display: flex; align-items: center; gap: 6px; cursor: pointer; }
  .btn-convert { background: #2da44e; color: #fff; border: none; padding: 10px 24px; border-radius: 6px; font-size: 14px; font-weight: 600; cursor: pointer; }
  .btn-convert:hover { background: #2c974b; }
  .btn-convert:disabled { background: #94d3a2; cursor: not-allowed; }
  .progress-section { padding: 16px 24px; background: #f6f8fa; border-bottom: 1px solid #e1e4e8; display: none; }
  .progress-bar { background: #d0d7de; border-radius: 4px; height: 8px; overflow: hidden; }
  .progress-fill { background: #2da44e; height: 100%; width: 0%; border-radius: 4px; transition: width 0.3s; }
  .progress-text { display: flex; justify-content: space-between; margin-bottom: 6px; font-size: 13px; }
  .results-section { padding: 16px 24px; }
  .result-card { border: 1px solid #d0d7de; border-radius: 6px; margin-bottom: 12px; overflow: hidden; }
  .result-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; background: #f6f8fa; border-bottom: 1px solid #e1e4e8; }
  .result-header.error { background: #fff5f5; }
  .result-info { display: flex; align-items: center; gap: 8px; }
  .result-icon-ok { color: #2da44e; font-size: 16px; }
  .result-icon-err { color: #cf222e; font-size: 16px; }
  .result-ref { font-size: 13px; font-family: monospace; }
  .result-type { font-size: 12px; color: #57606a; background: #ddf4ff; padding: 2px 8px; border-radius: 12px; }
  .result-preview { padding: 12px 16px; font-family: monospace; font-size: 12px; color: #57606a; max-height: 60px; overflow: hidden; line-height: 1.5; white-space: pre-wrap; }
  .result-full { display: none; padding: 12px 16px; font-family: monospace; font-size: 12px; max-height: 400px; overflow-y: auto; white-space: pre-wrap; background: #f6f8fa; border-top: 1px solid #e1e4e8; }
  .result-error { padding: 12px 16px; font-size: 13px; color: #cf222e; }
  .btn-sm { background: #fff; border: 1px solid #d0d7de; padding: 4px 12px; border-radius: 4px; font-size: 12px; cursor: pointer; }
  .btn-sm:hover { background: #f3f4f6; }
  .btn-download-all { background: #0969da; color: #fff; border: none; padding: 10px 20px; border-radius: 6px; font-size: 14px; font-weight: 600; cursor: pointer; display: none; }
  .btn-download-all:hover { background: #0860ca; }
  .footer-actions { display: flex; justify-content: flex-end; padding: 8px 0; }
</style>
</head>
<body>
  <div class="header">
    <h1>issue2md</h1>
    <span>GitHub Issue/PR/Discussion → Markdown</span>
  </div>

  <form id="convertForm" class="form-section">
    <label class="field-label" for="urls">GitHub URLs（每行一个，最多 20 条）</label>
    <textarea id="urls" name="urls" placeholder="https://github.com/golang/go/issues/1&#10;https://github.com/cli/cli/pull/1234&#10;https://github.com/vercel/next.js/discussions/48427"></textarea>

    <div class="options">
      <label><input type="checkbox" name="enable_reactions" value="on"> 显示 Reactions</label>
      <label><input type="checkbox" name="enable_user_links" value="on"> 启用用户链接</label>
    </div>

    <button type="submit" class="btn-convert" id="btnConvert">开始转换</button>
  </form>

  <div class="progress-section" id="progressSection">
    <div class="progress-text">
      <span>转换进度</span>
      <span id="progressCount">0 / 0 完成</span>
    </div>
    <div class="progress-bar">
      <div class="progress-fill" id="progressFill"></div>
    </div>
  </div>

  <div class="results-section" id="resultsSection"></div>

  <div class="results-section">
    <div class="footer-actions">
      <button class="btn-download-all" id="btnDownloadAll">全部下载 (.zip)</button>
    </div>
  </div>

<script>
// JS will be added in Task 5
</script>
</body>
</html>
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleIndex
```

预期：PASS

#### Refactor

- [ ] **Step 5: 审查 HTML — 确认语义化标签、无冗余 CSS**

- [ ] **Step 6: 提交**

```bash
git add cmd/issue2mdweb/
git commit -m "feat(web): implement index page with form, progress bar, and result layout"
```

---

### Task 3: URL 解析、去重、校验辅助函数

**Files:**
- Modify: `cmd/issue2mdweb/handler.go`
- Modify: `cmd/issue2mdweb/handler_test.go`

#### Red: 写失败测试

- [ ] **Step 1: 添加 parseURLList 和 generateFilename 测试**

追加到 `cmd/issue2mdweb/handler_test.go`：

```go
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

// generateNURLs 生成 n 个不重复的 URL 字符串（换行分隔）
func generateNURLs(n int) string {
	var s string
	for i := 1; i <= n; i++ {
		if i > 1 {
			s += "\n"
		}
		s += "https://github.com/owner/repo/issues/" + itoa(i)
	}
	return s
}

// generateNURLSlice 生成 n 个不重复的 URL 切片
func generateNURLSlice(n int) []string {
	urls := make([]string, n)
	for i := 0; i < n; i++ {
		urls[i] = "https://github.com/owner/repo/issues/" + itoa(i+1)
	}
	return urls
}

// itoa 简单整数转字符串
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
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
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./cmd/issue2mdweb/ -v -run "TestParseURLList|TestGenerateFilename"
```

预期：编译失败 — `parseURLList`、`generateFilename` 未定义。

#### Green: 写最小实现

- [ ] **Step 3: 在 handler.go 中添加辅助函数**

在 `cmd/issue2mdweb/handler.go` 顶部添加 import 并实现：

```go
package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dongjianfei/issue2md/internal/parser"
)

const maxURLs = 20

// parseURLList 解析多行 URL 文本，去空行、去空白、去重，校验数量限制
func parseURLList(input string) ([]string, error) {
	lines := strings.Split(input, "\n")
	seen := make(map[string]bool)
	var urls []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if seen[line] {
			continue
		}
		seen[line] = true
		urls = append(urls, line)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("URL 列表不能为空")
	}
	if len(urls) > maxURLs {
		return nil, fmt.Errorf("最多支持 20 个 URL")
	}

	return urls, nil
}

// generateFilename 从 GitHub URL 生成下载文件名
func generateFilename(rawURL string) string {
	parsed, err := parser.ParseURL(rawURL)
	if err != nil {
		return "output.md"
	}

	var typeStr string
	switch parsed.ContentType {
	case parser.TypeIssue:
		typeStr = "issue"
	case parser.TypePR:
		typeStr = "pr"
	case parser.TypeDiscussion:
		typeStr = "discussion"
	default:
		typeStr = "unknown"
	}

	return fmt.Sprintf("%s_%s_%s_%d.md", parsed.Owner, parsed.Repo, typeStr, parsed.Number)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "index.html", nil)
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) handleDownloadAll(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
```

同时在 `handler_test.go` 顶部 import 中添加 `"fmt"`。

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./cmd/issue2mdweb/ -v -run "TestParseURLList|TestGenerateFilename"
```

预期：PASS

#### Refactor

- [ ] **Step 5: 审查 — 确认错误信息清晰、边界条件完备**

- [ ] **Step 6: 提交**

```bash
git add cmd/issue2mdweb/
git commit -m "feat(web): add URL list parsing, deduplication, validation, and filename generation"
```

---

### Task 4: handleConvert — SSE 流式转换

**Files:**
- Modify: `cmd/issue2mdweb/handler.go`
- Modify: `cmd/issue2mdweb/handler_test.go`

#### Red: 写失败测试

- [ ] **Step 1: 添加 mock helpers 和 SSE 解析工具**

追加到 `cmd/issue2mdweb/handler_test.go`：

```go
import (
	"bufio"
	"encoding/json"
	"strings"
)

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
	// 处理末尾没有空行的情况
	if currentEvent.Event != "" {
		events = append(events, currentEvent)
	}

	return events
}
```

- [ ] **Step 2: 添加 SSE 行为测试**

追加到 `cmd/issue2mdweb/handler_test.go`：

```go
func TestHandleConvert(t *testing.T) {
	tests := []struct {
		name            string
		convertFn       convertFunc
		formURLs        string
		enableReactions string
		enableUserLinks string
		wantEvents      []string // 期望的 event 类型序列
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
```

- [ ] **Step 3: 运行测试，确认失败**

```bash
go test ./cmd/issue2mdweb/ -v -run "TestHandleConvert"
```

预期：FAIL — `handleConvert` 返回 501 Not Implemented，且 `ConvertResult` 类型未定义。

#### Green: 写最小实现

- [ ] **Step 4: 在 handler.go 中实现 ConvertResult 和 handleConvert**

更新 `cmd/issue2mdweb/handler.go`，替换 `handleConvert` stub：

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dongjianfei/issue2md/internal/cli"
	"github.com/dongjianfei/issue2md/internal/parser"
)

const maxURLs = 20

// ConvertResult 表示单个 URL 的转换结果
type ConvertResult struct {
	URL      string `json:"url"`
	Success  bool   `json:"success"`
	Markdown string `json:"markdown,omitempty"`
	Error    string `json:"error,omitempty"`
	Filename string `json:"filename,omitempty"`
	Index    int    `json:"index"`
	Total    int    `json:"total"`
}

// parseURLList 解析多行 URL 文本，去空行、去空白、去重，校验数量限制
func parseURLList(input string) ([]string, error) {
	lines := strings.Split(input, "\n")
	seen := make(map[string]bool)
	var urls []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if seen[line] {
			continue
		}
		seen[line] = true
		urls = append(urls, line)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("URL 列表不能为空")
	}
	if len(urls) > maxURLs {
		return nil, fmt.Errorf("最多支持 20 个 URL")
	}

	return urls, nil
}

// generateFilename 从 GitHub URL 生成下载文件名
func generateFilename(rawURL string) string {
	parsed, err := parser.ParseURL(rawURL)
	if err != nil {
		return "output.md"
	}

	var typeStr string
	switch parsed.ContentType {
	case parser.TypeIssue:
		typeStr = "issue"
	case parser.TypePR:
		typeStr = "pr"
	case parser.TypeDiscussion:
		typeStr = "discussion"
	default:
		typeStr = "unknown"
	}

	return fmt.Sprintf("%s_%s_%s_%d.md", parsed.Owner, parsed.Repo, typeStr, parsed.Number)
}

// writeSSE 写入一个 SSE 事件
func writeSSE(w http.ResponseWriter, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "index.html", nil)
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		writeSSE(w, "error", map[string]string{"message": "表单解析失败"})
		return
	}

	urlsRaw := r.FormValue("urls")
	enableReactions := r.FormValue("enable_reactions") == "on"
	enableUserLinks := r.FormValue("enable_user_links") == "on"

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	urls, err := parseURLList(urlsRaw)
	if err != nil {
		writeSSE(w, "error", map[string]string{"message": err.Error()})
		return
	}

	total := len(urls)
	writeSSE(w, "start", map[string]int{"total": total})

	completed := 0
	failed := 0

	for i, u := range urls {
		// 检查客户端是否断开
		if r.Context().Err() != nil {
			return
		}

		opts := &cli.RunOptions{
			URL:             u,
			EnableReactions: enableReactions,
			EnableUserLinks: enableUserLinks,
		}

		var buf bytes.Buffer
		result := ConvertResult{
			URL:   u,
			Index: i + 1,
			Total: total,
		}

		if err := s.convert(&buf, opts); err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Filename = generateFilename(u)
			failed++
		} else {
			result.Success = true
			result.Markdown = buf.String()
			result.Filename = generateFilename(u)
			completed++
		}

		writeSSE(w, "result", result)
	}

	writeSSE(w, "done", map[string]int{"completed": completed, "failed": failed})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) handleDownloadAll(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
```

- [ ] **Step 5: 运行测试，确认通过**

```bash
go test ./cmd/issue2mdweb/ -v -run "TestHandleConvert"
```

预期：PASS

#### Refactor

- [ ] **Step 6: 审查 — SSE 格式规范、Flush 调用、Context 检查**

- [ ] **Step 7: 提交**

```bash
git add cmd/issue2mdweb/
git commit -m "feat(web): implement SSE streaming conversion with per-URL error isolation"
```

---

### Task 5: handleDownload — 单个文件下载

**Files:**
- Modify: `cmd/issue2mdweb/handler.go`
- Modify: `cmd/issue2mdweb/handler_test.go`

#### Red: 写失败测试

- [ ] **Step 1: 添加 handleDownload 测试**

追加到 `cmd/issue2mdweb/handler_test.go`：

```go
func TestHandleDownload(t *testing.T) {
	tests := []struct {
		name           string
		formFilename   string
		formContent    string
		wantStatus     int
		wantFilename   string
		wantBody       string
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
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleDownload
```

预期：FAIL — 返回 501 Not Implemented。

#### Green: 写最小实现

- [ ] **Step 3: 实现 handleDownload**

替换 `handler.go` 中的 `handleDownload` stub：

```go
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}

	filename := r.FormValue("filename")
	content := r.FormValue("content")

	if filename == "" || content == "" {
		http.Error(w, "filename 和 content 不能为空", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write([]byte(content))
}
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleDownload
```

预期：PASS

#### Refactor

- [ ] **Step 5: 审查 — Content-Disposition 安全性（filename 不含路径遍历字符）**

- [ ] **Step 6: 提交**

```bash
git add cmd/issue2mdweb/
git commit -m "feat(web): implement single file download endpoint"
```

---

### Task 6: handleDownloadAll — ZIP 打包下载

**Files:**
- Modify: `cmd/issue2mdweb/handler.go`
- Modify: `cmd/issue2mdweb/handler_test.go`

#### Red: 写失败测试

- [ ] **Step 1: 添加 handleDownloadAll 测试**

追加到 `cmd/issue2mdweb/handler_test.go`：

```go
import (
	"archive/zip"
)

func TestHandleDownloadAll(t *testing.T) {
	tests := []struct {
		name       string
		filesJSON  string
		wantStatus int
		wantFiles  map[string]string // filename → content
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

				// 验证 ZIP 内容
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
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleDownloadAll
```

预期：FAIL — 返回 501 Not Implemented。

#### Green: 写最小实现

- [ ] **Step 3: 实现 handleDownloadAll**

在 `handler.go` 顶部添加 `"archive/zip"` 到 import，并定义文件结构体。替换 `handleDownloadAll` stub：

```go
// downloadFile 表示 ZIP 中的单个文件
type downloadFile struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func (s *Server) handleDownloadAll(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}

	filesJSON := r.FormValue("files")
	if filesJSON == "" {
		http.Error(w, "files 不能为空", http.StatusBadRequest)
		return
	}

	var files []downloadFile
	if err := json.Unmarshal([]byte(filesJSON), &files); err != nil {
		http.Error(w, "files JSON 格式错误", http.StatusBadRequest)
		return
	}

	if len(files) == 0 {
		http.Error(w, "文件列表不能为空", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="issue2md-export.zip"`)

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, f := range files {
		entry, err := zipWriter.Create(f.Filename)
		if err != nil {
			return
		}
		entry.Write([]byte(f.Content))
	}
}
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleDownloadAll
```

预期：PASS

#### Refactor

- [ ] **Step 5: 审查 — ZIP 写入错误处理、filename 安全性**

- [ ] **Step 6: 提交**

```bash
git add cmd/issue2mdweb/
git commit -m "feat(web): implement ZIP batch download endpoint"
```

---

### Task 7: 前端 JavaScript — SSE 消费与交互

**Files:**
- Modify: `cmd/issue2mdweb/templates/index.html`
- Modify: `cmd/issue2mdweb/handler_test.go` (验证完整流程)

#### Red: 写失败测试

- [ ] **Step 1: 添加端到端页面行为测试**

追加到 `cmd/issue2mdweb/handler_test.go`：

```go
func TestHandleIndexContainsJS(t *testing.T) {
	s := NewServer(mockConvertSuccess)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	wants := []string{
		"EventSource",    // 或 fetch + getReader
		"toggleExpand",
		"btnConvert",
		"resultsSection",
		"progressFill",
	}

	// 至少需要包含 fetch/getReader 和 toggleExpand
	jsWants := []string{"toggleExpand", "progressFill", "resultsSection"}
	for _, want := range jsWants {
		if !contains(body, want) {
			t.Errorf("index.html missing JS reference %q", want)
		}
	}
	_ = wants
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleIndexContainsJS
```

预期：FAIL — 当前 `index.html` 的 `<script>` 块只有注释。

#### Green: 写最小实现

- [ ] **Step 3: 在 index.html 中实现完整 JavaScript**

替换 `cmd/issue2mdweb/templates/index.html` 中 `<script>` 块的内容：

```html
<script>
(function() {
  const form = document.getElementById('convertForm');
  const btnConvert = document.getElementById('btnConvert');
  const progressSection = document.getElementById('progressSection');
  const progressCount = document.getElementById('progressCount');
  const progressFill = document.getElementById('progressFill');
  const resultsSection = document.getElementById('resultsSection');
  const btnDownloadAll = document.getElementById('btnDownloadAll');

  let results = [];

  form.addEventListener('submit', async function(e) {
    e.preventDefault();
    results = [];
    resultsSection.innerHTML = '';
    btnDownloadAll.style.display = 'none';
    progressSection.style.display = 'block';
    progressCount.textContent = '0 / 0 完成';
    progressFill.style.width = '0%';
    btnConvert.disabled = true;

    const formData = new FormData(form);
    try {
      const response = await fetch('/convert', { method: 'POST', body: new URLSearchParams(formData) });
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      let total = 0;
      let processed = 0;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        let currentEvent = '';
        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEvent = line.substring(7);
          } else if (line.startsWith('data: ')) {
            const data = JSON.parse(line.substring(6));
            handleEvent(currentEvent, data);
          }
        }
      }
    } catch (err) {
      resultsSection.innerHTML += '<div class="result-card"><div class="result-header error"><div class="result-info"><span class="result-icon-err">✗</span><span>网络错误: ' + escapeHtml(err.message) + '</span></div></div></div>';
    } finally {
      btnConvert.disabled = false;
    }

    function handleEvent(event, data) {
      if (event === 'start') {
        total = data.total;
        progressCount.textContent = '0 / ' + total + ' 完成';
      } else if (event === 'result') {
        processed++;
        progressCount.textContent = processed + ' / ' + total + ' 完成';
        progressFill.style.width = (processed / total * 100) + '%';
        appendResult(data);
        if (data.success) {
          results.push({ filename: data.filename, content: data.markdown });
        }
      } else if (event === 'done') {
        if (results.length > 0) {
          btnDownloadAll.style.display = 'inline-block';
        }
      } else if (event === 'error') {
        resultsSection.innerHTML += '<div class="result-card"><div class="result-header error"><div class="result-info"><span class="result-icon-err">✗</span><span>' + escapeHtml(data.message) + '</span></div></div></div>';
      }
    }
  });

  function appendResult(result) {
    const card = document.createElement('div');
    card.className = 'result-card';

    if (result.success) {
      const preview = escapeHtml(result.markdown.substring(0, 200));
      card.innerHTML =
        '<div class="result-header">' +
          '<div class="result-info">' +
            '<span class="result-icon-ok">✓</span>' +
            '<span class="result-ref">' + escapeHtml(result.filename.replace('.md','')) + '</span>' +
          '</div>' +
          '<div>' +
            '<button class="btn-sm" onclick="toggleExpand(this)">展开</button> ' +
            '<button class="btn-sm" onclick="downloadOne(\'' + escapeAttr(result.filename) + '\',' + result.index + ')">下载 .md</button>' +
          '</div>' +
        '</div>' +
        '<div class="result-preview">' + preview + '</div>' +
        '<div class="result-full">' + escapeHtml(result.markdown) + '</div>';
    } else {
      card.innerHTML =
        '<div class="result-header error">' +
          '<div class="result-info">' +
            '<span class="result-icon-err">✗</span>' +
            '<span class="result-ref">' + escapeHtml(result.url) + '</span>' +
          '</div>' +
        '</div>' +
        '<div class="result-error">' + escapeHtml(result.error) + '</div>';
    }
    resultsSection.appendChild(card);
  }

  window.toggleExpand = function(btn) {
    const card = btn.closest('.result-card');
    const full = card.querySelector('.result-full');
    const preview = card.querySelector('.result-preview');
    if (full.style.display === 'block') {
      full.style.display = 'none';
      preview.style.display = 'block';
      btn.textContent = '展开';
    } else {
      full.style.display = 'block';
      preview.style.display = 'none';
      btn.textContent = '收起';
    }
  };

  window.downloadOne = function(filename, index) {
    const item = results.find(r => r.filename === filename);
    if (!item) return;
    const f = document.createElement('form');
    f.method = 'POST';
    f.action = '/download';
    f.style.display = 'none';
    f.innerHTML = '<input name="filename" value="' + escapeAttr(item.filename) + '">' +
                  '<input name="content" value="' + escapeAttr(item.content) + '">';
    document.body.appendChild(f);
    f.submit();
    document.body.removeChild(f);
  };

  btnDownloadAll.addEventListener('click', function() {
    const f = document.createElement('form');
    f.method = 'POST';
    f.action = '/download-all';
    f.style.display = 'none';
    f.innerHTML = '<input name="files" value="' + escapeAttr(JSON.stringify(results)) + '">';
    document.body.appendChild(f);
    f.submit();
    document.body.removeChild(f);
  });

  function escapeHtml(str) {
    const d = document.createElement('div');
    d.textContent = str;
    return d.innerHTML;
  }

  function escapeAttr(str) {
    return str.replace(/&/g,'&amp;').replace(/"/g,'&quot;').replace(/'/g,'&#39;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
  }
})();
</script>
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./cmd/issue2mdweb/ -v -run TestHandleIndexContainsJS
```

预期：PASS

#### Refactor

- [ ] **Step 5: 审查 JS — XSS 防护（escapeHtml/escapeAttr）、无全局污染（IIFE 包裹）**

- [ ] **Step 6: 提交**

```bash
git add cmd/issue2mdweb/
git commit -m "feat(web): implement SSE client, progress bar, expand/collapse, and download logic"
```

---

### Task 8: main.go 入口与构建验证

**Files:**
- Create: `cmd/issue2mdweb/main.go`

#### Red: 写失败测试

- [ ] **Step 1: 验证编译**

```bash
go build ./cmd/issue2mdweb/
```

预期：编译失败 — 缺少 `main` 函数。

#### Green: 写最小实现

- [ ] **Step 2: 实现 main.go**

写入 `cmd/issue2mdweb/main.go`：

```go
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/dongjianfei/issue2md/internal/cli"
)

func main() {
	port := flag.String("port", "8080", "监听端口")
	flag.Parse()

	s := NewServer(cli.Run)

	log.Printf("issue2md web server listening on :%s", *port)
	if err := http.ListenAndServe(":"+*port, s.mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 3: 编译验证**

```bash
go build -o bin/issue2md-web ./cmd/issue2mdweb/
```

预期：成功，产出 `bin/issue2md-web`

- [ ] **Step 4: Makefile 构建验证**

```bash
make build-web
```

预期：成功

- [ ] **Step 5: 运行全量测试**

```bash
make test
```

预期：所有测试 PASS，覆盖率 ≥ 80%

#### Refactor

- [ ] **Step 6: 审查 — main.go 极简，无业务逻辑泄漏**

- [ ] **Step 7: 提交**

```bash
git add cmd/issue2mdweb/main.go
git commit -m "feat(web): add main entry point with flag parsing and server startup"
```

---

### Task 9: 全量验证与最终提交

**Files:**
- 无新增文件

- [ ] **Step 1: 运行全量单元测试**

```bash
make test
```

预期：全部 PASS

- [ ] **Step 2: 检查覆盖率**

```bash
go test ./cmd/issue2mdweb/ -coverprofile=coverage.out -v
go tool cover -func=coverage.out
```

预期：`handler.go` 覆盖率 ≥ 90%，`server.go` 覆盖率 ≥ 80%

- [ ] **Step 3: 构建 CLI 和 Web**

```bash
make build
```

预期：`bin/issue2md` 和 `bin/issue2md-web` 均成功产出

- [ ] **Step 4: 手动冒烟测试**

```bash
GITHUB_TOKEN=${GITHUB_TOKEN} ./bin/issue2md-web -port 8080 &
# 浏览器访问 http://localhost:8080
# 输入: https://github.com/golang/go/issues/1
# 点击"开始转换"
# 验证: 进度条更新、结果卡片出现、下载功能正常
kill %1
```

- [ ] **Step 5: go vet 检查**

```bash
make lint
```

预期：无警告

- [ ] **Step 6: 最终提交**

```bash
git add -A
git commit -m "feat(web): complete Web UI batch conversion service with SSE streaming

Implements Phase 3 Web UI for issue2md:
- SSE streaming for real-time conversion progress
- Batch URL processing (up to 20 URLs)
- Single .md download and ZIP batch download
- GitHub-style responsive UI with expand/collapse previews
- Full test coverage with mock convertFunc injection"
```

---

## Spec Coverage Verification

| Spec Section | Implementing Task |
|---|---|
| §3 架构设计 | Task 1 (Server 骨架) |
| §4 数据模型 | Task 3 (辅助函数) + Task 4 (ConvertResult) |
| §5.2 GET / | Task 2 (表单渲染) |
| §5.3 POST /convert (SSE) | Task 4 (SSE 流式转换) |
| §5.4 POST /download | Task 5 (单文件下载) |
| §5.5 POST /download-all | Task 6 (ZIP 打包) |
| §6 前端设计 | Task 2 (HTML/CSS) + Task 7 (JavaScript) |
| §7 Token 处理 | Task 8 (main.go 注入 cli.Run) |
| §8 输入校验与错误处理 | Task 3 (URL 校验) + Task 4 (per-URL 隔离) |
| §9 测试策略 | Task 1-8 (每个 Task 都是 Red-Green-Refactor) |
| §10 构建与部署 | Task 8 (main.go) + Task 9 (全量验证) |
| §11 验证方案 | Task 9 (冒烟测试 + 覆盖率) |
