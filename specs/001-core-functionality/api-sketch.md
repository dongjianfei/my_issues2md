# API Sketch - Internal Package Interfaces

## 版本信息
- **版本**: 1.0
- **创建日期**: 2026-05-20
- **状态**: Draft

---

## 设计原则

遵循项目宪法的核心原则：
1. **简单性优先**：避免过度抽象，优先使用简单的函数和数据结构
2. **明确性**：显式错误处理，无全局变量
3. **可测试性**：所有包都应易于编写单元测试

---

## 1. `internal/parser` - URL解析

### 职责
解析GitHub URL，识别内容类型（Issue/PR/Discussion）并提取关键信息。

### 核心类型

```go
package parser

// ContentType 表示GitHub内容类型
type ContentType string

const (
    ContentTypeIssue      ContentType = "issue"
    ContentTypePR         ContentType = "pull_request"
    ContentTypeDiscussion ContentType = "discussion"
)

// ParsedURL 包含解析后的URL信息
type ParsedURL struct {
    Owner       string      // 仓库所有者
    Repo        string      // 仓库名称
    Number      int         // Issue/PR/Discussion编号
    Type        ContentType // 内容类型
    OriginalURL string      // 原始URL
}
```

### 核心函数

```go
// ParseURL 解析GitHub URL并返回结构化信息
// 支持的URL格式：
//   - https://github.com/{owner}/{repo}/issues/{number}
//   - https://github.com/{owner}/{repo}/pull/{number}
//   - https://github.com/{owner}/{repo}/discussions/{number}
//
// 返回错误情况：
//   - URL格式无效
//   - 不是GitHub URL
//   - 缺少必需字段
func ParseURL(rawURL string) (*ParsedURL, error)
```

### 测试要点
- 有效URL的各种格式
- 无效URL（格式错误、非GitHub URL）
- 边界情况（空字符串、特殊字符）

---

## 2. `internal/github` - GitHub API客户端

### 职责
与GitHub API交互，获取Issue/PR/Discussion的完整数据。

### 核心类型

```go
package github

import "time"

// Client GitHub API客户端
type Client struct {
    token      string // GitHub Personal Access Token（可选）
    httpClient *http.Client
}

// User 表示GitHub用户
type User struct {
    Login     string // 用户名
    AvatarURL string // 头像URL
    HTMLURL   string // 用户主页URL
}

// Reaction 表示反应
type Reaction struct {
    Content string // 反应类型：+1, -1, laugh, confused, heart, hooray, rocket, eyes
    Count   int    // 数量
}

// Comment 表示评论
type Comment struct {
    ID        int64
    Author    User
    Body      string    // Markdown格式的评论内容
    CreatedAt time.Time
    UpdatedAt time.Time
    Reactions []Reaction
}

// Issue 表示GitHub Issue
type Issue struct {
    Number    int
    Title     string
    Author    User
    Body      string // Markdown格式的描述
    State     string // open, closed
    Labels    []string
    CreatedAt time.Time
    UpdatedAt time.Time
    Comments  []Comment
    Reactions []Reaction
    HTMLURL   string // 原始URL
}

// PullRequest 表示GitHub Pull Request
type PullRequest struct {
    Number    int
    Title     string
    Author    User
    Body      string
    State     string // open, closed, merged
    Labels    []string
    CreatedAt time.Time
    UpdatedAt time.Time
    Comments  []Comment // 包含普通评论和Review评论，已按时间排序
    Reactions []Reaction
    HTMLURL   string
}

// Discussion 表示GitHub Discussion
type Discussion struct {
    Number    int
    Title     string
    Author    User
    Body      string
    Category  string // Q&A, General, etc.
    CreatedAt time.Time
    UpdatedAt time.Time
    Comments  []DiscussionComment // 使用专门的类型以支持Answer标记
    Reactions []Reaction
    HTMLURL   string
}

// DiscussionComment Discussion的评论（支持Answer标记）
type DiscussionComment struct {
    Comment
    IsAnswer bool // 是否被标记为Answer
}
```

### 核心函数

```go
// NewClient 创建GitHub API客户端
// token为空时使用匿名访问（rate limit较低）
func NewClient(token string) *Client

// GetIssue 获取Issue的完整信息
// 包括主楼内容和所有评论
func (c *Client) GetIssue(owner, repo string, number int) (*Issue, error)

// GetPullRequest 获取PR的完整信息
// 包括主楼内容、普通评论和Review评论（已按时间排序）
func (c *Client) GetPullRequest(owner, repo string, number int) (*PullRequest, error)

// GetDiscussion 获取Discussion的完整信息
// 包括主楼内容和所有评论（标记Answer）
func (c *Client) GetDiscussion(owner, repo string, number int) (*Discussion, error)
```

### 实现细节
- 使用GitHub REST API v3或GraphQL API v4
- 处理分页（获取所有评论）
- 错误处理：404、403、rate limit等
- 对于PR，需要合并普通评论和Review评论，并按时间排序

### 测试要点
- Mock HTTP响应进行单元测试
- 测试分页逻辑
- 测试错误处理（404、403、超时）
- 集成测试：使用真实的公开Issue/PR/Discussion

---

## 3. `internal/converter` - Markdown转换器

### 职责
将GitHub数据结构转换为符合规范的Markdown格式。

### 核心类型

```go
package converter

// Options 转换选项
type Options struct {
    EnableReactions bool // 是否包含Reactions统计
    EnableUserLinks bool // 是否将用户名转换为链接
}
```

### 核心函数

```go
// ConvertIssue 将Issue转换为Markdown
func ConvertIssue(issue *github.Issue, opts Options) (string, error)

// ConvertPullRequest 将PR转换为Markdown
func ConvertPullRequest(pr *github.PullRequest, opts Options) (string, error)

// ConvertDiscussion 将Discussion转换为Markdown
func ConvertDiscussion(discussion *github.Discussion, opts Options) (string, error)
```

### 输出格式
- YAML Frontmatter（元数据）
- 标题和元信息表格
- 主楼内容
- 评论列表（按时间排序）
- 可选：Reactions统计
- 可选：用户链接

### 实现细节
- 使用Go标准库的`text/template`或直接字符串拼接
- 处理Markdown转义（如果需要）
- 格式化时间戳（ISO 8601格式）
- Reactions格式化：`**Reactions:** 👍 5 | ❤️ 3`
- Discussion Answer标记：`> ✅ **Accepted Answer**`

### 测试要点
- 测试各种内容类型的转换
- 测试Options的不同组合
- 测试特殊字符和Markdown语法
- 验证输出格式符合spec.md

---

## 4. `internal/config` - 配置管理

### 职责
管理应用配置，包括环境变量读取。

### 核心类型

```go
package config

// Config 应用配置
type Config struct {
    GitHubToken string // GitHub Personal Access Token
}
```

### 核心函数

```go
// Load 从环境变量加载配置
// 读取 GITHUB_TOKEN 环境变量
func Load() *Config

// GetToken 获取GitHub Token（如果存在）
func (c *Config) GetToken() string
```

### 实现细节
- 使用`os.Getenv("GITHUB_TOKEN")`
- Token为空时返回空字符串（使用匿名访问）
- 不在日志中输出token

### 测试要点
- 测试环境变量存在和不存在的情况
- 测试token不会泄露到日志

---

## 5. `internal/cli` - 命令行接口

### 职责
处理命令行参数解析和主要业务流程编排。

### 核心类型

```go
package cli

// App CLI应用
type App struct {
    parser    *parser.Parser
    github    *github.Client
    converter *converter.Converter
    config    *config.Config
}

// Options CLI选项
type Options struct {
    URL             string // GitHub URL（必需）
    OutputFile      string // 输出文件路径（可选，为空则输出到stdout）
    EnableReactions bool   // 是否包含Reactions
    EnableUserLinks bool   // 是否包含用户链接
}
```

### 核心函数

```go
// NewApp 创建CLI应用实例
func NewApp(cfg *config.Config) *App

// Run 执行主要业务逻辑
// 1. 解析URL
// 2. 调用GitHub API获取数据
// 3. 转换为Markdown
// 4. 输出到stdout或文件
func (a *App) Run(opts Options) error

// ParseFlags 解析命令行参数
// 返回Options或错误
func ParseFlags(args []string) (*Options, error)
```

### 实现细节
- 使用Go标准库`flag`包解析参数
- 错误输出到stderr
- 成功时退出码为0，失败时为1
- 输出到stdout时使用`fmt.Print`，输出到文件时使用`os.WriteFile`

### 测试要点
- 测试参数解析（有效和无效参数）
- 测试业务流程编排
- 测试错误处理和退出码

---

## 6. `cmd/issue2md` - CLI入口

### 职责
应用程序入口点，初始化并启动CLI应用。

### 核心代码结构

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/owner/issue2md/internal/cli"
    "github.com/owner/issue2md/internal/config"
)

func main() {
    // 1. 加载配置
    cfg := config.Load()
    
    // 2. 创建CLI应用
    app := cli.NewApp(cfg)
    
    // 3. 解析命令行参数
    opts, err := cli.ParseFlags(os.Args[1:])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    // 4. 执行
    if err := app.Run(opts); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### 测试要点
- 集成测试：端到端测试完整流程
- 使用真实的GitHub Issue/PR/Discussion

---

## 7. 数据流图

```
用户输入 (URL + Flags)
    ↓
cmd/issue2md/main.go
    ↓
internal/cli.App.Run()
    ↓
internal/parser.ParseURL() → ParsedURL
    ↓
internal/github.Client.Get*() → Issue/PR/Discussion
    ↓
internal/converter.Convert*() → Markdown string
    ↓
输出 (stdout 或 文件)
```

---

## 8. 错误处理策略

所有包的错误处理遵循以下原则：

1. **显式错误处理**：所有错误必须被处理，不允许忽略
2. **错误包装**：使用`fmt.Errorf("context: %w", err)`包装错误，保留错误链
3. **错误类型**：
   - `parser`包：返回格式错误
   - `github`包：返回API错误（404、403、超时等）
   - `converter`包：返回转换错误（理论上很少）
   - `cli`包：汇总所有错误并输出到stderr

### 错误示例

```go
// parser包
if !strings.HasPrefix(rawURL, "https://github.com/") {
    return nil, fmt.Errorf("invalid GitHub URL: must start with https://github.com/")
}

// github包
if resp.StatusCode == 404 {
    return nil, fmt.Errorf("issue not found: %s/%s#%d", owner, repo, number)
}

// cli包
if err := app.Run(opts); err != nil {
    return fmt.Errorf("failed to convert: %w", err)
}
```

---

## 9. 依赖关系

```
cmd/issue2md
    ↓
internal/cli
    ↓
├── internal/config
├── internal/parser
├── internal/github
└── internal/converter
        ↓
    internal/github (数据类型依赖)
```

**依赖规则：**
- `cmd`层依赖`internal/cli`
- `internal/cli`依赖其他所有`internal`包
- `internal/converter`依赖`internal/github`（数据类型）
- 其他`internal`包之间无依赖
- 所有包都可以依赖Go标准库

---

## 10. 测试策略

### 单元测试
- 每个包都有对应的`*_test.go`文件
- 使用表格驱动测试（Table-Driven Tests）
- Mock外部依赖（如HTTP请求）

### 集成测试
- 在`cmd/issue2md`层编写端到端测试
- 使用真实的公开GitHub Issue/PR/Discussion
- 验证完整的转换流程

### 测试覆盖率目标
- 总体覆盖率 > 80%
- 核心包（parser, github, converter）覆盖率 > 90%

---

## 11. 未来扩展点

### Web版本 (`cmd/issue2mdweb`)
- 复用所有`internal`包
- 添加HTTP handler层
- 使用`web/templates`和`web/static`

### 批量转换
- 在`internal/cli`或新包中添加批量处理逻辑
- 复用现有的转换流程

### 自定义模板
- 在`internal/converter`中添加模板支持
- 允许用户提供自定义Markdown模板

---

## 12. 开发顺序建议

基于TDD原则，建议按以下顺序开发：

1. **`internal/parser`** - 最简单，无外部依赖
   - 先写测试
   - 实现URL解析逻辑

2. **`internal/github`** - 核心数据获取
   - 先写测试（Mock HTTP）
   - 实现GitHub API客户端
   - 集成测试（真实API）

3. **`internal/converter`** - 数据转换
   - 先写测试
   - 实现Markdown生成逻辑

4. **`internal/config`** - 配置管理
   - 先写测试
   - 实现环境变量读取

5. **`internal/cli`** - 业务编排
   - 先写测试
   - 实现命令行接口和流程编排

6. **`cmd/issue2md`** - 入口点
   - 集成测试
   - 端到端验证

---

## 13. 变更历史

| 版本 | 日期 | 作者 | 变更说明 |
|-----|------|------|---------|
| 1.0 | 2026-05-20 | CTO & Claude | 初始版本，定义核心包接口 |
