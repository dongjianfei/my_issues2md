# issue2md

将 GitHub Issue、Pull Request、Discussion 转换为结构化 Markdown 文件。提供命令行工具和 Web UI 两种使用方式。

适用于离线阅读、归档保存、知识库构建或二次加工。

## 快速开始

### 环境要求

- Go 1.24+
- GitHub Personal Access Token（用于访问 API，尤其是 Discussion 需要 `read:discussion` 权限）

### 安装

```bash
# 从源码编译
git clone https://github.com/dongjianfei/my_issues2md.git
cd my_issues2md
make build

# 二进制文件产出在 bin/issue2md 和 bin/issue2md-web
```

### 设置 Token

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

### CLI 使用

```bash
# 输出到终端
bin/issue2md https://github.com/golang/go/issues/1

# 输出到文件
bin/issue2md https://github.com/golang/go/issues/1 output.md

# 启用 Reactions 统计
bin/issue2md --enable-reactions https://github.com/owner/repo/pull/123

# 启用用户链接
bin/issue2md --enable-user-links https://github.com/owner/repo/discussions/456
```

### Web UI 使用

```bash
# 启动 Web 服务（默认端口 8080）
GITHUB_TOKEN=$GITHUB_TOKEN bin/issue2md-web

# 自定义端口
GITHUB_TOKEN=$GITHUB_TOKEN bin/issue2md-web -port 3000
```

浏览器访问 `http://localhost:8080`，粘贴 GitHub URL（每行一个，最多 20 条），点击"开始转换"即可。

**Web UI 功能：**
- SSE 流式推送，实时显示每个 URL 的转换进度
- 批量处理，自动去重，单个失败不影响其他
- 支持单个 .md 下载和 ZIP 批量打包下载
- 展开/收起预览转换结果

### Docker 方式运行

```bash
make docker-build
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN issue2md https://github.com/golang/go/issues/1
```

## 支持的 URL 类型

| 类型 | URL 格式 | 说明 |
|------|----------|------|
| Issue | `https://github.com/{owner}/{repo}/issues/{number}` | 含所有评论和标签 |
| Pull Request | `https://github.com/{owner}/{repo}/pull/{number}` | 含普通评论和 Review 评论（不含 diff） |
| Discussion | `https://github.com/{owner}/{repo}/discussions/{number}` | 含楼中楼回复和 Accepted Answer 标记 |

## 项目结构

```
.
├── cmd/
│   ├── issue2md/          # CLI 入口
│   └── issue2mdweb/       # Web UI 批量转换服务
│       ├── main.go        # flag 解析(-port)、启动 HTTP 服务
│       ├── server.go      # Server struct、convertFunc DI、路由注册
│       ├── handler.go     # HTTP handlers + URL 辅助函数 + SSE helpers
│       ├── handler_test.go# 21 个表格驱动 httptest 测试
│       └── templates/
│           └── index.html # 单页模板（表单 + CSS + JS）
├── internal/
│   ├── parser/            # URL 解析，识别内容类型
│   ├── github/            # GitHub API 交互层（REST + GraphQL）
│   ├── converter/         # 数据 → Markdown 转换
│   └── cli/               # 参数解析 + 主流程编排
├── docs/superpowers/
│   ├── specs/             # 设计规格书
│   └── plans/             # 实现计划
├── Dockerfile             # 多阶段生产级容器构建
├── Makefile               # 构建、测试、lint、Docker 一站式命令
└── constitution.md        # 项目开发宪法（核心原则）
```

## 架构概览

```
┌─────────────┐     ┌──────────┐     ┌──────────────┐     ┌───────────┐
│  CLI/main   │────▶│  parser  │────▶│    github    │────▶│ converter │──▶ Markdown
│  (入口)     │     │(URL解析) │     │(API获取数据) │     │(格式转换) │
└─────────────┘     └──────────┘     └──────────────┘     └───────────┘
       ▲
       │ convertFunc DI
┌──────┴──────┐
│ Web Server  │──▶ SSE 流式推送 ──▶ 浏览器
│ (薄适配层)  │
└─────────────┘
```

- **parser**: 解析 GitHub URL，提取 owner/repo/number/type
- **github**: Issue/PR 走 REST API（go-github），Discussion 走 GraphQL（githubv4）
- **converter**: 将结构化数据渲染为带 YAML frontmatter 的 GFM Markdown
- **Web Server**: `cli.Run` 的 HTTP 包装，通过 `convertFunc` 依赖注入，不重复业务逻辑

## 开发指南

### 常用命令

```bash
make build            # 编译 CLI + Web 二进制
make build-cli        # 仅编译 CLI
make build-web        # 仅编译 Web 服务
make test             # 运行单元测试
make test-integration # 运行集成测试（需要 GITHUB_TOKEN）
make lint             # 静态分析（golangci-lint 或 go vet）
make docker-build     # 构建 Docker 镜像
make clean            # 清理构建产物
make help             # 查看所有可用命令
```

### 测试策略

- **单元测试**: 使用 `httptest` mock GitHub API，不依赖网络，`make test` 即可运行
- **集成测试**: 访问真实 GitHub API，需设置 `GITHUB_TOKEN`，通过 `make test-integration` 运行
- **测试风格**: 表格驱动测试（Table-Driven Tests）

### 核心开发原则

项目遵循 `constitution.md` 中定义的开发宪法：

1. **简单性原则** — 标准库优先，不过度抽象，只实现 spec 要求的功能
2. **测试先行铁律** — 严格 Red-Green-Refactor，优先集成测试
3. **明确性原则** — 所有错误必须用 `fmt.Errorf("...: %w", err)` 显式包装，禁止全局变量

### 添加新功能的流程

1. 阅读 `specs/` 下的相关规格文档
2. 在 `internal/` 对应包中编写失败的测试
3. 实现代码使测试通过
4. 运行 `make test && make lint` 确认无回归

## CLI 参数说明

```
issue2md [flags] <url> [output_file]

Flags:
  --enable-reactions    在评论下方显示 Reactions 统计（👍 ❤️ 🎉 等）
  --enable-user-links   将用户名渲染为 GitHub 个人主页链接

Arguments:
  url          GitHub Issue/PR/Discussion 的完整 URL（必填）
  output_file  输出文件路径（可选，不填则输出到 stdout）
```

## 已知限制

- Discussion 每条评论的回复最多获取 100 条（超出时 stderr 输出警告）
- 不支持 GitHub Enterprise（仅支持 github.com）
- PR 输出不含代码 diff 和 commit 历史
- Web UI 不支持私有仓库的 per-request Token 注入（使用服务端环境变量中的 Token）

## 依赖

| 依赖 | 用途 |
|------|------|
| `github.com/google/go-github/v60` | Issue/PR REST API 交互 |
| `github.com/shurcooL/githubv4` | Discussion GraphQL API 交互 |
| `golang.org/x/oauth2` | GitHub Token 认证 |

## License

MIT
