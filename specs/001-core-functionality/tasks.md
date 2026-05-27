# Task List - issue2md Core Functionality

## 文档信息
- **创建日期**: 2026-05-20
- **对应Plan**: specs/001-core-functionality/plan.md
- **对应Spec**: specs/001-core-functionality/spec.md

## 约定
- `[P]` = 可与同阶段内其他`[P]`任务并行执行
- `[S]` = 必须串行执行，依赖前置任务
- `[TDD-Red]` = 编写失败测试
- `[TDD-Green]` = 编写实现使测试通过
- 每个任务只涉及一个主要文件的创建或修改

---

## Phase 0: Project Bootstrap

### T-000: 初始化Go Module [S]
- **文件**: `go.mod`, `go.sum`
- **动作**:
  - `go mod init github.com/yourusername/issue2md`
  - 添加依赖: `google/go-github/v60`, `shurcooL/githubv4`, `golang.org/x/oauth2`
  - `go mod tidy`
- **验收**: `go build ./...` 无报错

### T-001: 创建Makefile [S]
- **文件**: `Makefile`
- **动作**: 创建Makefile，包含以下target:
  - `build`: `go build -o bin/issue2md ./cmd/issue2md/`
  - `test`: `go test ./... -v -count=1`
  - `test-integration`: `go test ./... -v -count=1 -tags=integration`
  - `lint`: `go vet ./...`
  - `clean`: `rm -rf bin/`
  - `web`: `go build -o bin/issue2mdweb ./cmd/issue2mdweb/`
- **验收**: `make test` 可执行（即使暂无测试文件）

### T-002: 创建.gitignore [P]
- **文件**: `.gitignore`
- **动作**: 添加Go项目标准忽略规则: `bin/`, `*.exe`, `.env`, `.DS_Store`
- **验收**: `bin/`目录不被git追踪

---

## Phase 1: Foundation（数据结构定义）

> 本阶段定义所有核心数据类型。无外部API调用，无业务逻辑。

### T-100: 定义parser包类型与ParseURL函数签名 [P]
- **文件**: `internal/parser/parser.go`
- **动作**:
  - 定义`ContentType`类型及常量: `TypeIssue`, `TypePR`, `TypeDiscussion`
  - 定义`ParsedURL`结构体: `Owner`, `Repo`, `Number`, `ContentType`, `RawURL`
  - 写出`ParseURL(rawURL string) (*ParsedURL, error)`函数签名，函数体暂时`return nil, nil`
- **验收**: `go build ./internal/parser/` 通过

### T-101: 定义github包核心数据结构 [P]
- **文件**: `internal/github/types.go`
- **动作**: 定义以下结构体:
  - `User`: `Login string`, `HTMLURL string`
  - `Label`: `Name string`
  - `Reaction`: `PlusOne`, `MinusOne`, `Laugh`, `Confused`, `Heart`, `Hooray`, `Rocket`, `Eyes` (全部int)
  - `Comment`: `Author User`, `Body string`, `CreatedAt time.Time`, `IsReview bool`, `Reactions Reaction`
  - `DiscussionComment`: 嵌入`Comment`，添加`IsAnswer bool`
  - `Issue`: `Number`, `Title`, `Author`, `Body`, `State`, `Labels`, `CreatedAt`, `UpdatedAt`, `HTMLURL`, `Comments []Comment`, `Reactions`, `CommentCount`
  - `PullRequest`: 同Issue结构，State包含"merged"，Comments包含Review评论
  - `Discussion`: `Number`, `Title`, `Author`, `Body`, `Category`, `CreatedAt`, `UpdatedAt`, `HTMLURL`, `Comments []DiscussionComment`, `Reactions`, `CommentCount`
- **验收**: `go build ./internal/github/` 通过

### T-102: 定义github包Client结构体与构造函数 [S, 依赖T-101]
- **文件**: `internal/github/client.go`
- **动作**:
  - 定义`Client`结构体，持有`rest *gogithub.Client`和`graphql *githubv4.Client`
  - 实现`NewClient(token string) *Client`
  - token为空时创建匿名客户端，非空时创建认证客户端
  - 定义Fetch方法的签名（函数体暂时`return nil, fmt.Errorf("not implemented")`）:
    - `FetchIssue(owner, repo string, number int) (*Issue, error)`
    - `FetchPullRequest(owner, repo string, number int) (*PullRequest, error)`
    - `FetchDiscussion(owner, repo string, number int) (*Discussion, error)`
- **验收**: `go build ./internal/github/` 通过

### T-103: 定义converter包Options和函数签名 [P]
- **文件**: `internal/converter/converter.go`
- **动作**:
  - 定义`Options`结构体: `EnableReactions bool`, `EnableUserLinks bool`
  - 写出三个函数签名（函数体暂时返回空字符串）:
    - `ConvertIssue(issue *github.Issue, opts Options) string`
    - `ConvertPullRequest(pr *github.PullRequest, opts Options) string`
    - `ConvertDiscussion(disc *github.Discussion, opts Options) string`
- **验收**: `go build ./internal/converter/` 通过

### T-104: 定义cli包RunOptions和函数签名 [P]
- **文件**: `internal/cli/cli.go`
- **动作**:
  - 定义`RunOptions`结构体: `URL`, `OutputFile`, `EnableReactions`, `EnableUserLinks`
  - 写出函数签名（函数体暂时返回错误）:
    - `ParseArgs(args []string) (*RunOptions, error)`
    - `Run(w io.Writer, opts *RunOptions) error`
- **验收**: `go build ./internal/cli/` 通过

### T-105: 创建cmd/issue2md入口占位 [S, 依赖T-104]
- **文件**: `cmd/issue2md/main.go`
- **动作**:
  - 创建`main()`函数
  - 调用`cli.ParseArgs(os.Args[1:])`和`cli.Run(os.Stdout, opts)`
  - 错误时`fmt.Fprintf(os.Stderr, ...)`并`os.Exit(1)`
- **验收**: `go build ./cmd/issue2md/` 通过；`make build` 通过

### Phase 1 验收门禁
- `make build` 通过
- `make lint` 通过
- 所有包可独立编译

---

## Phase 2: GitHub Fetcher（API交互逻辑，TDD）

> 本阶段实现GitHub API数据获取。严格遵循TDD: 先Red后Green。

### 2A: URL Parser（无外部依赖，最先实现）

#### T-200: [TDD-Red] 编写parser包测试 [S]
- **文件**: `internal/parser/parser_test.go`
- **动作**: 编写表格驱动测试，覆盖以下Case:
  - 有效Issue URL `https://github.com/owner/repo/issues/123` → `TypeIssue`, owner="owner", repo="repo", number=123
  - 有效PR URL `https://github.com/owner/repo/pull/456` → `TypePR`
  - 有效Discussion URL `https://github.com/owner/repo/discussions/789` → `TypeDiscussion`
  - 带尾部斜杠的URL → 正确解析
  - 带锚点的URL `...#issuecomment-123` → 正确解析（忽略锚点）
  - 带查询参数的URL → 正确解析（忽略参数）
  - 空字符串 → 错误
  - 非GitHub URL `https://gitlab.com/...` → 错误
  - 缺少number `https://github.com/owner/repo/issues/` → 错误
  - number非数字 `https://github.com/owner/repo/issues/abc` → 错误
  - 不支持的路径类型 `https://github.com/owner/repo/wiki/...` → 错误
- **验收**: `go test ./internal/parser/` 全部FAIL（Red状态）

#### T-201: [TDD-Green] 实现ParseURL函数 [S, 依赖T-200]
- **文件**: `internal/parser/parser.go`
- **动作**:
  - 使用`net/url`解析URL
  - 提取path segments，识别owner/repo/type/number
  - 根据路径段判断ContentType
  - 处理所有错误Case
- **验收**: `go test ./internal/parser/ -v` 全部PASS（Green状态）

### 2B: Issue Fetcher

#### T-210: [TDD-Red] 编写Issue获取测试 [S, 依赖T-201]
- **文件**: `internal/github/issue_test.go`
- **动作**:
  - 使用`httptest.NewServer`模拟GitHub REST API响应
  - 测试Case:
    - 正常Issue（含标题、描述、标签、状态）→ 返回完整Issue结构
    - Issue带评论（模拟分页，2页，每页1条） → 返回所有评论，按时间正序
    - Issue带Reactions → Reaction字段正确填充
    - 404响应 → 返回明确的"not found"错误
    - 403响应 → 返回权限错误
- **验收**: `go test ./internal/github/ -run TestFetchIssue` 全部FAIL

#### T-211: [TDD-Green] 实现FetchIssue方法 [S, 依赖T-210]
- **文件**: `internal/github/issue.go`
- **动作**:
  - 使用`go-github`客户端获取Issue基本信息
  - 获取Issue评论（处理分页，PerPage=100）
  - 获取Reactions（Issue主楼 + 各评论）
  - 组装为`*Issue`返回
  - 显式错误处理，使用`fmt.Errorf("fetch issue %s/%s#%d: %w", ...)`包装
- **验收**: `go test ./internal/github/ -run TestFetchIssue -v` 全部PASS

### 2C: Pull Request Fetcher

#### T-220: [TDD-Red] 编写PR获取测试 [S, 依赖T-211]
- **文件**: `internal/github/pullrequest_test.go`
- **动作**:
  - 使用`httptest.NewServer`模拟GitHub REST API响应
  - 测试Case:
    - 正常PR（含标题、描述、状态=merged） → 返回完整PullRequest结构
    - PR带普通评论和Review评论 → 两种评论合并，按时间正序排列
    - Review评论的`IsReview`字段 → 标记为true
    - 普通评论的`IsReview`字段 → 标记为false
    - 仅有Review评论，无普通评论 → 正确返回
    - 404响应 → 返回明确错误
- **验收**: `go test ./internal/github/ -run TestFetchPullRequest` 全部FAIL

#### T-221: [TDD-Green] 实现FetchPullRequest方法 [S, 依赖T-220]
- **文件**: `internal/github/pullrequest.go`
- **动作**:
  - 使用`go-github`获取PR基本信息
  - 分别获取Issue Comments和Review Comments（各自处理分页）
  - 合并两种评论，标记`IsReview`
  - `sort.Slice`按`CreatedAt`排序
  - 获取Reactions
  - 组装为`*PullRequest`返回
- **验收**: `go test ./internal/github/ -run TestFetchPullRequest -v` 全部PASS

### 2D: Discussion Fetcher（GraphQL）

#### T-230: [TDD-Red] 编写Discussion获取测试 [S, 依赖T-221]
- **文件**: `internal/github/discussion_test.go`
- **动作**:
  - 使用`httptest.NewServer`模拟GraphQL API响应
  - 测试Case:
    - 正常Discussion（含标题、描述、Category） → 返回完整Discussion结构
    - Discussion带评论（含一个标记为Answer的评论） → `IsAnswer=true`
    - Discussion评论分页（模拟`hasNextPage`和cursor） → 返回所有评论
    - Discussion带Reactions → 正确统计
    - 不存在的Discussion → 返回错误
- **验收**: `go test ./internal/github/ -run TestFetchDiscussion` 全部FAIL

#### T-231: [TDD-Green] 实现FetchDiscussion方法 [S, 依赖T-230]
- **文件**: `internal/github/discussion.go`
- **动作**:
  - 使用`githubv4`客户端执行GraphQL查询
  - 定义GraphQL查询结构体（嵌套struct映射查询字段）
  - 处理评论分页（cursor-based pagination）
  - 映射Answer标记到`DiscussionComment.IsAnswer`
  - 统计Reactions
  - 组装为`*Discussion`返回
- **验收**: `go test ./internal/github/ -run TestFetchDiscussion -v` 全部PASS

### Phase 2 验收门禁
- `make test` 全部通过
- `make lint` 通过
- 所有GitHub Fetcher测试绿色

---

## Phase 3: Markdown Converter（转换逻辑，TDD）

> 本阶段实现数据到Markdown的转换。三种类型可并行开发。

### 3A: 通用辅助函数

#### T-300: [TDD-Red] 编写通用格式化函数测试 [P]
- **文件**: `internal/converter/converter_test.go`
- **动作**: 编写表格驱动测试，覆盖以下辅助函数:
  - `formatReactions(r github.Reaction) string`:
    - 全部为0 → 返回空字符串
    - 部分有值 → 仅显示>0的，格式`**Reactions:** 👍 5 | ❤️ 3`
    - 全部有值 → 按固定顺序显示
  - `formatUser(u github.User, enableLinks bool) string`:
    - enableLinks=false → `@username`
    - enableLinks=true → `[@username](https://github.com/username)`
  - `formatTime(t time.Time) string`:
    - 返回格式 `2026-05-20 10:30:00 UTC`
  - `formatLabels(labels []github.Label) string`:
    - 空列表 → 空字符串
    - 多个标签 → `` `bug`, `enhancement` ``
- **验收**: `go test ./internal/converter/ -run TestFormat` 全部FAIL

#### T-301: [TDD-Green] 实现通用格式化函数 [S, 依赖T-300]
- **文件**: `internal/converter/converter.go`
- **动作**:
  - 在已有的`converter.go`中实现上述辅助函数（unexported）
  - 使用`fmt.Sprintf`和`strings.Builder`拼接
- **验收**: `go test ./internal/converter/ -run TestFormat -v` 全部PASS

### 3B: Issue Converter

#### T-310: [TDD-Red] 编写Issue转换测试 [S, 依赖T-301]
- **文件**: `internal/converter/issue_test.go`
- **动作**: 构造`github.Issue`结构体作为输入，编写表格驱动测试:
  - 基本Issue（无评论，无Reactions） → 包含Frontmatter、标题、元信息、描述，无Comments部分
  - Issue带2条评论 → Comments部分按时间排列
  - 启用Reactions → 主楼和评论下方有Reactions行
  - 禁用Reactions → 无Reactions行
  - 启用UserLinks → 用户名为链接格式
  - 禁用UserLinks → 用户名为纯文本格式
  - Issue带Labels → 元信息中显示Labels
  - Frontmatter验证 → 包含title, url, type, author, created_at, state, labels, comments_count
- **验收**: `go test ./internal/converter/ -run TestConvertIssue` 全部FAIL

#### T-311: [TDD-Green] 实现ConvertIssue函数 [S, 依赖T-310]
- **文件**: `internal/converter/issue.go`
- **动作**:
  - 使用`strings.Builder`拼接Markdown
  - 输出YAML Frontmatter
  - 输出标题、元信息（Author, Created, Status, Labels）
  - 输出描述（Description部分）
  - 可选输出Reactions
  - 输出评论列表（Comments部分）
- **验收**: `go test ./internal/converter/ -run TestConvertIssue -v` 全部PASS

### 3C: PR Converter

#### T-320: [TDD-Red] 编写PR转换测试 [P, 可与T-310并行]
- **文件**: `internal/converter/pullrequest_test.go`
- **动作**: 构造`github.PullRequest`结构体作为输入，编写表格驱动测试:
  - 基本PR（status=merged） → Frontmatter中type="pull_request"，status="merged"
  - PR带混合评论（普通+Review） → Review评论标题为"Review Comment by ..."
  - 启用/禁用Reactions → 同Issue
  - 启用/禁用UserLinks → 同Issue
- **验收**: `go test ./internal/converter/ -run TestConvertPullRequest` 全部FAIL

#### T-321: [TDD-Green] 实现ConvertPullRequest函数 [S, 依赖T-320, T-301]
- **文件**: `internal/converter/pullrequest.go`
- **动作**:
  - 结构与Issue类似
  - 评论标题区分普通评论和Review评论:
    - 普通: `### Comment by @user on 2026-05-20 10:30:00 UTC`
    - Review: `### Review Comment by @user on 2026-05-20 10:30:00 UTC`
- **验收**: `go test ./internal/converter/ -run TestConvertPullRequest -v` 全部PASS

### 3D: Discussion Converter

#### T-330: [TDD-Red] 编写Discussion转换测试 [P, 可与T-310并行]
- **文件**: `internal/converter/discussion_test.go`
- **动作**: 构造`github.Discussion`结构体作为输入，编写表格驱动测试:
  - 基本Discussion → Frontmatter中type="discussion"，含Category行
  - Discussion主楼标题为"Question"而非"Description"
  - Discussion带Answer评论 → 评论体前有`> ✅ **Accepted Answer**`标记
  - Discussion无Answer → 所有评论正常显示
  - 启用/禁用Reactions → 同Issue
  - 启用/禁用UserLinks → 同Issue
- **验收**: `go test ./internal/converter/ -run TestConvertDiscussion` 全部FAIL

#### T-331: [TDD-Green] 实现ConvertDiscussion函数 [S, 依赖T-330, T-301]
- **文件**: `internal/converter/discussion.go`
- **动作**:
  - 结构与Issue类似，但:
    - 主楼部分标题为"## Question"
    - 元信息增加Category行
    - Answer评论添加`> ✅ **Accepted Answer**`引用块
- **验收**: `go test ./internal/converter/ -run TestConvertDiscussion -v` 全部PASS

### Phase 3 验收门禁
- `make test` 全部通过
- 所有Converter测试绿色
- 输出格式符合spec.md第5节示例

---

## Phase 4: CLI Assembly（命令行入口集成）

> 本阶段组装所有模块，实现端到端功能。

### 4A: 参数解析

#### T-400: [TDD-Red] 编写CLI参数解析测试 [S]
- **文件**: `internal/cli/cli_test.go`
- **动作**: 编写表格驱动测试:
  - `["https://github.com/o/r/issues/1"]` → URL正确，OutputFile为空，Flags默认false
  - `["https://github.com/o/r/issues/1", "out.md"]` → OutputFile="out.md"
  - `["-enable-reactions", "https://github.com/o/r/issues/1"]` → EnableReactions=true
  - `["-enable-user-links", "https://github.com/o/r/issues/1"]` → EnableUserLinks=true
  - `["-enable-reactions", "-enable-user-links", "URL", "out.md"]` → 所有字段正确
  - `[]`（空参数） → 错误
  - 缺少URL → 错误
  - 多余位置参数（3个以上） → 错误
- **验收**: `go test ./internal/cli/ -run TestParseArgs` 全部FAIL

#### T-401: [TDD-Green] 实现ParseArgs函数 [S, 依赖T-400]
- **文件**: `internal/cli/cli.go`
- **动作**:
  - 使用`flag.NewFlagSet`创建FlagSet
  - 注册`-enable-reactions`和`-enable-user-links`两个bool flag
  - 解析后取剩余位置参数: `flagSet.Args()`
  - 第一个位置参数为URL（必需），第二个为OutputFile（可选）
  - 返回`*RunOptions`
- **验收**: `go test ./internal/cli/ -run TestParseArgs -v` 全部PASS

### 4B: 主流程编排

#### T-410: [TDD-Red] 编写Run函数测试 [S, 依赖T-401]
- **文件**: `internal/cli/cli_test.go`（追加）
- **动作**: 编写Run函数的测试:
  - 无效URL → 返回错误（测试parser集成）
  - 输出写入`bytes.Buffer` → 验证Markdown内容非空（集成测试桩）
- **注意**: Run的完整集成测试需要真实API或大量Mock，此处仅测试错误路径和基本流程

#### T-411: [TDD-Green] 实现Run函数 [S, 依赖T-410]
- **文件**: `internal/cli/cli.go`（追加）
- **动作**:
  - 读取`os.Getenv("GITHUB_TOKEN")`
  - 调用`parser.ParseURL(opts.URL)`
  - 调用`github.NewClient(token)`
  - 根据`ContentType`分发调用:
    - `TypeIssue` → `client.FetchIssue()` → `converter.ConvertIssue()`
    - `TypePR` → `client.FetchPullRequest()` → `converter.ConvertPullRequest()`
    - `TypeDiscussion` → `client.FetchDiscussion()` → `converter.ConvertDiscussion()`
  - 将Markdown字符串写入`io.Writer`
  - 所有错误用`fmt.Errorf`包装后返回
- **验收**: `go test ./internal/cli/ -v` 全部PASS

### 4C: main入口与文件输出

#### T-420: 完善main.go入口 [S, 依赖T-411]
- **文件**: `cmd/issue2md/main.go`
- **动作**:
  - 调用`cli.ParseArgs(os.Args[1:])`
  - 判断`opts.OutputFile`:
    - 为空 → 传入`os.Stdout`
    - 非空 → `os.Create(opts.OutputFile)`，defer Close
  - 调用`cli.Run(writer, opts)`
  - 错误输出到stderr，`os.Exit(1)`
- **验收**: `make build` 通过

#### T-421: 编写端到端集成测试 [S, 依赖T-420]
- **文件**: `cmd/issue2md/main_test.go`
- **动作**:
  - 使用`//go:build integration` build tag
  - 测试Case（需网络，使用真实的知名公开Issue/PR/Discussion）:
    - 转换一个真实Issue → 输出包含Frontmatter和评论
    - 转换一个真实PR → 输出包含Review评论
    - 转换一个真实Discussion → 输出包含Answer标记（如有）
  - 运行方式: `make test-integration`
- **验收**: `make test-integration` 全部PASS（需网络环境）

### Phase 4 验收门禁
- `make build` 通过
- `make test` 全部通过
- `make lint` 通过
- 手动执行`./bin/issue2md https://github.com/golang/go/issues/1` 输出正确的Markdown

---

## 任务依赖关系总览

```
Phase 0: T-000 → T-001
         T-002 [P]

Phase 1: T-100 [P] ──────────────────────────────────► T-200
         T-101 [P] → T-102 ─────────────────────────► T-210
         T-103 [P] ──────────────────────────────────► T-300
         T-104 [P] → T-105 ─────────────────────────► T-400

Phase 2: T-200 → T-201 (parser)
         T-210 → T-211 (issue fetcher)
         T-220 → T-221 (PR fetcher)
         T-230 → T-231 (discussion fetcher)
         [串行: T-201 → T-210 → T-220 → T-230]

Phase 3: T-300 → T-301 (通用格式化)
         T-310 → T-311 (issue converter)     [依赖T-301]
         T-320 → T-321 (PR converter)        [依赖T-301, 可与T-310并行]
         T-330 → T-331 (discussion converter) [依赖T-301, 可与T-310并行]

Phase 4: T-400 → T-401 (参数解析)
         T-410 → T-411 (Run函数)
         T-420 (main入口)
         T-421 (集成测试)
         [串行: T-400 → T-410 → T-420 → T-421]
```

---

## 任务统计

| Phase | 任务数 | TDD对数 | 并行机会 |
|-------|--------|---------|---------|
| Phase 0 | 3 | 0 | T-001/T-002 |
| Phase 1 | 6 | 0 | T-100/T-101/T-103/T-104 |
| Phase 2 | 8 | 4 | 无（串行依赖链） |
| Phase 3 | 8 | 4 | T-310/T-320/T-330 |
| Phase 4 | 4 | 1 | 无（串行依赖链） |
| **合计** | **29** | **9** | - |
