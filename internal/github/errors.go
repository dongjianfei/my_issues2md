package github

import (
	"fmt"
	"net/http"
	"strings"

	gogithub "github.com/google/go-github/v60/github"
)

// wrapAPIError 将GitHub API错误转换为用户友好的错误信息
func wrapAPIError(err error, resource string) error {
	if err == nil {
		return nil
	}

	// 检查是否是go-github的ErrorResponse
	if errResp, ok := err.(*gogithub.ErrorResponse); ok {
		switch errResp.Response.StatusCode {
		case http.StatusNotFound:
			return fmt.Errorf("%s not found (404). Please check the URL and ensure the resource exists", resource)
		case http.StatusForbidden:
			if strings.Contains(errResp.Message, "rate limit") {
				return fmt.Errorf("GitHub API rate limit exceeded. Please set GITHUB_TOKEN environment variable for higher limits")
			}
			return fmt.Errorf("access forbidden (403). Please check your GITHUB_TOKEN permissions or if the repository is private")
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401). Please check your GITHUB_TOKEN")
		default:
			return fmt.Errorf("%s: %s (HTTP %d)", resource, errResp.Message, errResp.Response.StatusCode)
		}
	}

	return fmt.Errorf("%s: %w", resource, err)
}
