package github

import (
	"context"
	"net/http"
	"time"

	gogithub "github.com/google/go-github/v60/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Client GitHub API客户端
type Client struct {
	rest    *gogithub.Client
	graphql *githubv4.Client
}

// NewClient 创建GitHub API客户端。
// token为空字符串时使用匿名访问（rate limit较低）。
func NewClient(token string) *Client {
	var httpClient *http.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(context.Background(), ts)
	} else {
		httpClient = &http.Client{}
	}

	// 设置30秒超时，避免网络异常时无限挂起
	httpClient.Timeout = 30 * time.Second

	return &Client{
		rest:    gogithub.NewClient(httpClient),
		graphql: githubv4.NewClient(httpClient),
	}
}

// newGraphQLClientWithURL creates a githubv4 client pointing at a custom URL (for testing).
func newGraphQLClientWithURL(httpClient *http.Client, url string) *githubv4.Client {
	return githubv4.NewEnterpriseClient(url, httpClient)
}
