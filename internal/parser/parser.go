package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ContentType 表示GitHub内容类型
type ContentType string

const (
	TypeIssue      ContentType = "issue"
	TypePR         ContentType = "pull_request"
	TypeDiscussion ContentType = "discussion"
)

// ParsedURL 解析后的GitHub URL
type ParsedURL struct {
	Owner       string
	Repo        string
	Number      int
	ContentType ContentType
	RawURL      string
}

// ParseURL 解析GitHub URL，识别内容类型并提取关键信息。
// 支持的URL格式：
//   - https://github.com/{owner}/{repo}/issues/{number}
//   - https://github.com/{owner}/{repo}/pull/{number}
//   - https://github.com/{owner}/{repo}/discussions/{number}
func ParseURL(rawURL string) (*ParsedURL, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("empty URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Host != "github.com" {
		return nil, fmt.Errorf("not a GitHub URL: host is %q", u.Host)
	}

	// Split path into segments, filtering out empty strings from leading/trailing slashes
	var segments []string
	for _, s := range strings.Split(u.Path, "/") {
		if s != "" {
			segments = append(segments, s)
		}
	}

	// Expect at least: owner/repo/type/number
	if len(segments) < 4 {
		return nil, fmt.Errorf("invalid GitHub URL: expected /owner/repo/type/number, got %q", u.Path)
	}

	owner := segments[0]
	repo := segments[1]
	typeSeg := segments[2]
	numberStr := segments[3]

	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return nil, fmt.Errorf("invalid number in URL: %q: %w", numberStr, err)
	}

	var contentType ContentType
	switch typeSeg {
	case "issues":
		contentType = TypeIssue
	case "pull":
		contentType = TypePR
	case "discussions":
		contentType = TypeDiscussion
	default:
		return nil, fmt.Errorf("unsupported URL type: %q (expected issues, pull, or discussions)", typeSeg)
	}

	return &ParsedURL{
		Owner:       owner,
		Repo:        repo,
		Number:      number,
		ContentType: contentType,
		RawURL:      rawURL,
	}, nil
}
