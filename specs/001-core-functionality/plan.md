# Technical Implementation Plan - issue2md Core Functionality

## 文档信息
- **版本**: 1.0
- **创建日期**: 2026-05-20
- **状态**: Draft
- **对应Spec**: specs/001-core-functionality/spec.md

---

## 1. 技术上下文总结

### 1.1 技术选型

| 技术领域 | 选型 | 理由 |
|---------|------|------|
| **编程语言** | Go >= 1.21.0 | 项目要求 |
| **Web框架** | `net/http` (标准库) | 简单性原则，不引入Gin/Echo |
| **GitHub REST API** | `google/go-github/v60` | 官方推荐，社区成熟，用于Issue和PR |
| **GitHub GraphQL API** | `shurcooL/githubv4` | Discussion仅GraphQL API支持 |
| **Markdown生成** | 字符串拼接 + `fmt` | 输出格式简单，无需模板引擎 |
| **命令行解析** | `flag` (标准库) | 满足需求，无需cobra |
| **测试框架** | `testing` (标准库) | 表格驱动测试 |
| **数据存储** | 无 | 实时API获取，无需持久化 |

### 1.2 外部依赖清单

```go
// go.mod
module github.com/yourusername/issue2md

go 1.21

require (
    github.com/google/go-github/v60 v60.0.0   // GitHub REST API v3
    github.com/shurcooL/githubv4 v0.0.0        // GitHub GraphQL API v4 (Discussion)
    golang.org/x/oauth2 v0.18.0                // GitHub Token认证
)
```

### 1.3 API选型说明

| 内容类型 | API | 理由 |
|---------|-----|------|
| Issue | REST API v3 (`go-github`) | REST API功能完整，获取Issue+评论+Reactions |
| Pull Request | REST API v3 (`go-github`) | REST API支持PR评论和Review评论 |
| Discussion | GraphQL API v4 (`githubv4`) | Discussion **仅**GraphQL API支持，REST API无此端点 |

---

## 2. "合宪性"审查

逐条对照`constitution.md`，确认本方案合规。

### 第一条：简单性原则 (Simplicity First)

| 条款 | 审查结果 | 说明 |
|-----|---------|------|
| 1.1 YAGNI | ✅ 通过 | 仅实现spec.md明确要求的功能 |
| 1.2 标准库优先 | ✅ 通过 | Web用`net/http`，CLI用`flag`，Markdown用字符串拼接 |
| 1.3 反过度工程 | ✅ 通过 | 使用简单struct和函数，仅在测试需要时定义interface |

### 第二条：测试先行铁律 (Test-First Imperative)

| 条款 | 审查结果 | 说明 |
|-----|---------|------|
| 2.1 TDD循环 | ✅ 承诺 | 所有功能严格遵循Red-Green-Refactor |
| 2.2 表格驱动 | ✅ 承诺 | 所有单元测试使用Table-Driven Tests |
| 2.3 拒绝Mocks | ⚠️ 部分例外 | GitHub API使用`httptest.NewServer`模拟HTTP响应，而非Mock接口 |

**关于2.3的说明：** 宪法要求"拒绝Mocks，优先集成测试"。本方案采用折中策略：
- **单元测试**：使用`net/http/httptest`启动本地HTTP Server模拟GitHub API响应，这是真实的HTTP交互而非接口Mock
- **集成测试**：提供针对真实GitHub公开仓库的端到端测试（需网络）

### 第三条：明确性原则 (Clarity and Explicitness)

| 条款 | 审查结果 | 说明 |
|-----|---------|------|
| 3.1 错误处理 | ✅ 通过 | 所有错误显式处理，使用`fmt.Errorf("...: %w", err)`包装 |
| 3.2 无全局变量 | ✅ 通过 | 所有依赖通过函数参数或struct成员注入 |

---

## 3. 项目结构

### 3.1 目录结构

```
issue2md/
├── cmd/
│   ├── issue2md/
│   │   └── main.go            # CLI入口
│   └── issue2mdweb/
│       └── main.go            # Web入口（未来）
├── internal/
│   ├── parser/
│   │   ├── parser.go          # URL解析逻辑
│   │   └── parser_test.go
│   ├── github/
│   │   ├── types.go           # 核心数据结构
│   │   ├── client.go          # Client构造与通用逻辑
│   │   ├── issue.go           # Issue获取
│   │   ├── pullrequest.go     # PR获取
│   │   ├── discussion.go      # Discussion获取(GraphQL)
│   │   └── *_test.go
│   ├── converter/
│   │   ├── converter.go       # 转换入口与通用逻辑
│   │   ├── issue.go           # Issue → Markdown
│   │   ├── pullrequest.go     # PR → Markdown
│   │   ├── discussion.go      # Discussion → Markdown
│   │   └── *_test.go
│   └── cli/
│       ├── cli.go             # 参数解析与流程编排
│       └── cli_test.go
├── web/
│   ├── templates/             # Web模板（未来）
│   └── static/                # 静态资源（未来）
├── specs/
│   └── 001-core-functionality/
│       ├── spec.md
│       ├── api-sketch.md
│       └── plan.md            # 本文件
├── CLAUDE.md
├── constitution.md
├── Makefile
├── go.mod
└── go.sum
```

### 3.2 包职责与依赖关系

```
cmd/issue2md/main.go
    │
    ▼
internal/cli         # 参数解析 + 流程编排
    │
    ├──► internal/parser      # URL → ParsedURL (无外部依赖)
    ├──► internal/github      # ParsedURL → Issue/PR/Discussion (依赖go-github, githubv4)
    └──► internal/converter   # Issue/PR/Discussion → Markdown字符串 (依赖github/types.go)
```

**依赖规则：**
- `parser` 零外部依赖，仅依赖标准库
- `github` 依赖 `go-github`、`githubv4`、`oauth2`
- `converter` 依赖 `internal/github`（仅数据类型）
- `cli` 依赖 `parser`、`github`、`converter`
- 包之间不允许循环依赖

**注意：** 相比api-sketch.md，移除了`internal/config`包。Token读取仅一行代码`os.Getenv("GITHUB_TOKEN")`，为此创建一个包违反简单性原则，直接在`cli`包中处理即可。

---

## 4. 核心数据结构

### 4.1 `internal/parser` - URL解析结果

```go
package parser

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
```

### 4.2 `internal/github` - 核心数据模型

```go
package github

import "time"

// User GitHub用户
type User struct {
    Login   string
    HTMLURL string
}

// Label 标签
type Label struct {
    Name string
}

// Reaction 反应统计
type Reaction struct {
    PlusOne    int // 👍
    MinusOne   int // 👎
    Laugh      int // 😄
    Confused   int // 😕
    Heart      int // ❤️
    Hooray     int // 🎉
    Rocket     int // 🚀
    Eyes       int // 👀
}

// Comment 评论
type Comment struct {
    Author    User
    Body      string
    CreatedAt time.Time
    IsReview  bool      // 是否为PR Review评论
    Reactions Reaction
}

// DiscussionComment Discussion评论（扩展Comment，支持Answer标记）
type DiscussionComment struct {
    Comment
    IsAnswer bool
}

// Issue GitHub Issue完整数据
type Issue struct {
    Number       int
    Title        string
    Author       User
    Body         string
    State        string   // "open", "closed"
    Labels       []Label
    CreatedAt    time.Time
    UpdatedAt    time.Time
    HTMLURL      string
    Comments     []Comment
    Reactions    Reaction
    CommentCount int
}

// PullRequest GitHub PR完整数据
type PullRequest struct {
    Number       int
    Title        string
    Author       User
    Body         string
    State        string   // "open", "closed", "merged"
    Labels       []Label
    CreatedAt    time.Time
    UpdatedAt    time.Time
    HTMLURL      string
    Comments     []Comment  // 普通评论 + Review评论，已按时间排序
    Reactions    Reaction
    CommentCount int
}

// Discussion GitHub Discussion完整数据
type Discussion struct {
    Number       int
    Title        string
    Author       User
    Body         string
    Category     string
    CreatedAt    time.Time
    UpdatedAt    time.Time
    HTMLURL      string
    Comments     []DiscussionComment
    Reactions    Reaction
    CommentCount int
}
```

### 4.3 `internal/converter` - 转换选项

```go
package converter

// Options 控制Markdown输出的可选功能
type Options struct {
    EnableReactions bool
    EnableUserLinks bool
}
```

### 4.4 `internal/cli` - 命令行选项

```go
package cli

// RunOptions 从命令行参数解析出的运行选项
type RunOptions struct {
    URL             string
    OutputFile      string // 空字符串表示输出到stdout
    EnableReactions bool
    EnableUserLinks bool
}
```

---

## 5. 接口设计

### 5.1 `internal/parser`

```go
// ParseURL 解析GitHub URL，识别内容类型
// 错误场景：非GitHub URL、缺少owner/repo/number、不支持的路径类型
func ParseURL(rawURL string) (*ParsedURL, error)
```

### 5.2 `internal/github`

```go
// Client GitHub API客户端
type Client struct {
    rest    *gogithub.Client    // REST API (Issue, PR)
    graphql *githubv4.Client    // GraphQL API (Discussion)
}

// NewClient 创建GitHub API客户端
// token为空字符串时使用匿名访问
func NewClient(token string) *Client

// FetchIssue 获取Issue完整数据（含所有评论，已分页）
func (c *Client) FetchIssue(owner, repo string, number int) (*Issue, error)

// FetchPullRequest 获取PR完整数据（含普通评论+Review评论，已按时间排序）
func (c *Client) FetchPullRequest(owner, repo string, number int) (*PullRequest, error)

// FetchDiscussion 获取Discussion完整数据（含Answer标记）
func (c *Client) FetchDiscussion(owner, repo string, number int) (*Discussion, error)
```

### 5.3 `internal/converter`

```go
// ConvertIssue 将Issue转换为Markdown字符串
func ConvertIssue(issue *github.Issue, opts Options) string

// ConvertPullRequest 将PR转换为Markdown字符串
func ConvertPullRequest(pr *github.PullRequest, opts Options) string

// ConvertDiscussion 将Discussion转换为Markdown字符串
func ConvertDiscussion(disc *github.Discussion, opts Options) string
```

**设计决策：** Convert函数返回`string`而非`(string, error)`。原因：转换逻辑是纯数据格式化，输入数据结构已经过验证，不会产生运行时错误。如果后续引入模板引擎需要错误处理，再修改签名。

### 5.4 `internal/cli`

```go
// ParseArgs 解析命令行参数
// args 通常传入 os.Args[1:]
func ParseArgs(args []string) (*RunOptions, error)

// Run 执行主流程：解析URL → 获取数据 → 转换Markdown → 输出
// w 是输出目标（stdout或文件），便于测试
func Run(w io.Writer, opts *RunOptions) error
```

**设计决策：** `Run`接受`io.Writer`参数而非直接写文件/stdout。这样：
- 测试时可以传入`bytes.Buffer`
- 实际运行时传入`os.Stdout`或`os.File`
- 遵循Go惯用的io接口模式

---

## 6. 关键实现细节

### 6.1 PR评论合并排序

PR有两种评论来源，需要合并后按时间排序：

```go
// pullrequest.go 中的实现思路
func (c *Client) FetchPullRequest(owner, repo string, number int) (*PullRequest, error) {
    // 1. 获取PR基本信息
    // 2. 获取Issue Comments (普通评论)
    // 3. 获取Review Comments (代码审查评论)
    // 4. 合并两种评论，标记IsReview
    // 5. 按CreatedAt排序
    sort.Slice(allComments, func(i, j int) bool {
        return allComments[i].CreatedAt.Before(allComments[j].CreatedAt)
    })
}
```

### 6.2 Discussion GraphQL查询

Discussion仅通过GraphQL API可获取：

```graphql
query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    discussion(number: $number) {
      title
      body
      author { login url }
      category { name }
      createdAt
      updatedAt
      url
      reactions(first: 100) {
        nodes { content }
      }
      comments(first: 100) {
        nodes {
          author { login url }
          body
          createdAt
          isAnswer
          reactions(first: 100) {
            nodes { content }
          }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}
```

### 6.3 分页策略

- **Issue/PR评论**：使用`go-github`内置的`ListOptions{PerPage: 100}`分页，循环直到所有页面获取完毕
- **Discussion评论**：GraphQL使用cursor分页，每次获取100条，循环直到`hasNextPage`为false

### 6.4 Markdown输出结构

所有三种类型共享统一的输出骨架：

```
[YAML Frontmatter]
---
[标题]
[元信息行: Author / Created / Status / Labels]
---
## Description
[主楼内容]
[可选: Reactions]
---
## Comments
### Comment by [@user] on [time]
[评论内容]
[可选: Reactions]
---
...
```

### 6.5 Reactions格式化

```go
// 将Reaction结构体转换为显示字符串
// 输出示例: **Reactions:** 👍 5 | ❤️ 3 | 🚀 1
// 仅显示数量>0的反应类型
func formatReactions(r github.Reaction) string
```

### 6.6 错误处理链

```
cli.Run()
  → parser.ParseURL() 失败 → fmt.Errorf("invalid URL: %w", err)
  → github.FetchXxx() 失败 → fmt.Errorf("failed to fetch issue: %w", err)
  → 写入文件失败         → fmt.Errorf("failed to write output: %w", err)
  → 所有错误最终在 main.go 中输出到 stderr 并 os.Exit(1)
```

---

## 7. 测试策略

### 7.1 测试分层

| 层级 | 包 | 策略 | 工具 |
|-----|-----|------|------|
| 单元测试 | `parser` | 纯函数测试，表格驱动 | `testing` |
| 单元测试 | `github` | `httptest.NewServer`模拟API响应 | `testing`, `net/http/httptest` |
| 单元测试 | `converter` | 构造struct输入，验证输出字符串 | `testing` |
| 单元测试 | `cli` | 测试参数解析，Mock io.Writer | `testing` |
| 集成测试 | `cmd/issue2md` | 真实GitHub API端到端测试 | `testing`, build tag |

### 7.2 测试用例覆盖

**parser包：**
- 有效Issue URL → 返回正确的ParsedURL
- 有效PR URL → 返回正确的ParsedURL
- 有效Discussion URL → 返回正确的ParsedURL
- 空字符串 → 错误
- 非GitHub URL → 错误
- 缺少number → 错误
- 含锚点/查询参数的URL → 正确解析

**github包：**
- 正常响应 → 返回完整数据
- 404 → 返回明确的"not found"错误
- 403 → 返回权限错误
- 429 (rate limit) → 返回rate limit错误
- 分页场景 → 返回所有评论
- PR评论合并 → 按时间正确排序

**converter包：**
- Issue转换 → 输出包含Frontmatter、标题、描述、评论
- PR转换 → Review评论标记正确
- Discussion转换 → Answer标记正确
- Reactions开启/关闭 → 输出正确
- UserLinks开启/关闭 → 输出正确
- 空评论列表 → 不输出Comments部分
- 特殊字符 → 正确保留

**cli包：**
- 正常参数解析
- 缺少URL → 错误
- 无效flag → 错误
- 输出到stdout vs 文件

---

## 8. Makefile设计

```makefile
.PHONY: build test lint clean web

# 构建CLI
build:
	go build -o bin/issue2md ./cmd/issue2md/

# 构建Web服务（未来）
web:
	go build -o bin/issue2mdweb ./cmd/issue2mdweb/

# 运行所有测试
test:
	go test ./... -v -count=1

# 运行集成测试（需要网络）
test-integration:
	go test ./... -v -count=1 -tags=integration

# 检查代码
lint:
	go vet ./...

# 清理构建产物
clean:
	rm -rf bin/
```

---

## 9. 开发顺序（TDD驱动）

严格按依赖顺序，每个包先写测试再实现。

### Phase 1: 基础层（无外部依赖）

| 步骤 | 包 | 任务 | 预计产出 |
|-----|-----|------|---------|
| 1.1 | 项目初始化 | `go mod init`, Makefile, .gitignore | 项目骨架 |
| 1.2 | `internal/parser` | TDD: URL解析 | parser.go, parser_test.go |

### Phase 2: 数据获取层

| 步骤 | 包 | 任务 | 预计产出 |
|-----|-----|------|---------|
| 2.1 | `internal/github` | 定义types.go | 核心数据结构 |
| 2.2 | `internal/github` | TDD: Issue获取 | issue.go, issue_test.go |
| 2.3 | `internal/github` | TDD: PR获取（含评论合并） | pullrequest.go, pullrequest_test.go |
| 2.4 | `internal/github` | TDD: Discussion获取(GraphQL) | discussion.go, discussion_test.go |

### Phase 3: 转换层

| 步骤 | 包 | 任务 | 预计产出 |
|-----|-----|------|---------|
| 3.1 | `internal/converter` | TDD: Issue → Markdown | issue.go, issue_test.go |
| 3.2 | `internal/converter` | TDD: PR → Markdown | pullrequest.go, pullrequest_test.go |
| 3.3 | `internal/converter` | TDD: Discussion → Markdown | discussion.go, discussion_test.go |

### Phase 4: 集成层

| 步骤 | 包 | 任务 | 预计产出 |
|-----|-----|------|---------|
| 4.1 | `internal/cli` | TDD: 参数解析 + 流程编排 | cli.go, cli_test.go |
| 4.2 | `cmd/issue2md` | main.go入口 + 端到端测试 | main.go |

---

## 10. 变更历史

| 版本 | 日期 | 变更说明 |
|-----|------|---------|
| 1.0 | 2026-05-20 | 初始版本 |
