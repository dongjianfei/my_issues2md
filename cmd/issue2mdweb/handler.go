package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dongjianfei/issue2md/internal/cli"
	"github.com/dongjianfei/issue2md/internal/parser"
)

const maxRequestBody = 10 << 20 // 10MB

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

// sanitizeFilename 过滤文件名中的路径遍历和危险字符
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, `"`, "")
	name = strings.ReplaceAll(name, `\`, "")
	if name == "" || name == "." {
		return "output.md"
	}
	return name
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		log.Printf("template render error: %v", err)
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

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

// writeSSE 写入一个 SSE 事件
func writeSSE(w http.ResponseWriter, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("SSE marshal error: %v", err)
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
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
			log.Printf("convert error for %s: %v", u, err)
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
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}

	filename := sanitizeFilename(r.FormValue("filename"))
	content := r.FormValue("content")

	if content == "" {
		http.Error(w, "filename 和 content 不能为空", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if _, err := w.Write([]byte(content)); err != nil {
		log.Printf("write download error: %v", err)
	}
}

// downloadFile 表示 ZIP 中的单个文件
type downloadFile struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func (s *Server) handleDownloadAll(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
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
		safeName := sanitizeFilename(f.Filename)
		entry, err := zipWriter.Create(safeName)
		if err != nil {
			log.Printf("zip create error: %v", err)
			return
		}
		if _, err := entry.Write([]byte(f.Content)); err != nil {
			log.Printf("zip write error: %v", err)
			return
		}
	}
}
