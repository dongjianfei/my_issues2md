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
