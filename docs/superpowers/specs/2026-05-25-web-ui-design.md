# issue2md Web UI 批量转换服务 — 设计规格书

> **版本**: 1.0  
> **日期**: 2026-05-25  
> **状态**: Draft  
> **作者**: dongjianfei + Claude  

---

## 1. 背景与目标

### 1.1 背景

issue2md CLI 工具已完成 MVP（Phase 1 & 2），支持将单个 GitHub Issue/PR/Discussion URL 转换为结构化 Markdown 文件。当前用户必须通过命令行逐条执行，无法高效进行批量归档。

### 1.2 目标

提供一个 Web 界面，使内部团队成员无需使用 CLI 即可批量转换 GitHub URL 为 Markdown 文件，并支持实时查看转换进度和批量下载结果。

### 1.3 非目标

- 不做用户认证/权限系统（内部工具，信任内网环境）
- 不做限流/多租户（单实例内部部署）
- 不做私有仓库支持（留给后续独立迭代）
- 不做自定义模板（Phase 3 其他功能）
- 不做持久化存储（无数据库，纯无状态服务）

---

## 2. 用户场景

### 2.1 目标用户

内部团队成员（开发、产品、项目管理），需要批量归档 GitHub Issue/PR/Discussion 为 Markdown 文档。

### 2.2 核心用户流程

1. 用户打开 Web 页面
2. 在文本框中粘贴 1-20 个 GitHub URL（每行一个）
3. 勾选可选项（显示 Reactions、启用用户链接）
4. 点击"开始转换"
5. 页面实时展示每个 URL 的转换进度和结果（SSE 流式推送）
6. 转换完成后，可逐个下载 .md 文件或一键打包下载 .zip

---

## 3. 架构设计

### 3.1 设计原则

- **薄适配层**：Web 服务是 `cli.Run()` 的 HTTP 包装，不重复业务逻辑
- **标准库优先**：仅使用 `net/http`、`html/template`、`embed`、`archive/zip`（遵守宪法 1.2）
- **无状态**：不引入 session/数据库，转换结果由前端 JS 在内存中收集
- **依赖注入**：通过 `convertFunc` 类型注入转换函数，生产环境为 `cli.Run`，测试时为 mock

### 3.2 文件结构

```
cmd/issue2mdweb/
├── main.go              # 入口: flag解析(-port), 构造Server, 启动http.ListenAndServe
├── server.go            # Server struct, NewServer(), 路由注册, go:embed 模板加载
├── handler.go           # HTTP handlers + 数据类型 + URL校验辅助函数
├── handler_test.go      # 表格驱动 httptest 单元测试
└── templates/
    └── index.html       # 单页模板: 表单 + 内联CSS + vanilla JS(EventSource)
```

### 3.3 依赖关系

```
cmd/issue2mdweb/main.go
  └── cmd/issue2mdweb/server.go
        ├── cmd/issue2mdweb/handler.go
        │     └── internal/cli.Run()  (通过 convertFunc 注入)
        └── cmd/issue2mdweb/templates/index.html (go:embed)
```

不新增 `internal/` 包。不修改任何现有文件（`internal/cli/cli.go`、`internal/parser/`、`internal/github/`、`internal/converter/`、`Makefile` 均不变）。

### 3.4 新增标准库依赖

| 包 | 用途 |
|---|---|
| `archive/zip` | ZIP 打包下载 |
| `embed` | 模板文件嵌入 |
| `html/template` | 服务端 HTML 模板渲染 |
| `encoding/json` | SSE data 字段序列化 |
| `bufio` | 测试中解析 SSE 流 |

无第三方依赖。

---

## 4. 数据模型

### 4.1 核心类型

```go
// convertFunc 是核心依赖 — 与 cli.Run 签名一致
type convertFunc func(w io.Writer, opts *cli.RunOptions) error

// Server 持有所有依赖，通过构造函数注入
type Server struct {
    convert   convertFunc
    templates *template.Template
    mux       *http.ServeMux
}

// ConvertResult 表示单个 URL 的转换结果（SSE 逐条推送）
type ConvertResult struct {
    URL      string `json:"url"`
    Success  bool   `json:"success"`
    Markdown string `json:"markdown,omitempty"`
    Error    string `json:"error,omitempty"`
    Filename string `json:"filename,omitempty"` // e.g. "golang_go_issue_1.md"
    Index    int    `json:"index"`              // 在批次中的序号（从1开始）
    Total    int    `json:"total"`              // 批次总数
}
```

### 4.2 Filename 生成规则

从 URL 推导：`{owner}_{repo}_{type}_{number}.md`

示例：
- `https://github.com/golang/go/issues/1` → `golang_go_issue_1.md`
- `https://github.com/cli/cli/pull/1234` → `cli_cli_pr_1234.md`
- `https://github.com/vercel/next.js/discussions/48427` → `vercel_next.js_discussion_48427.md`

---

## 5. API 设计

### 5.1 端点总览

| 方法 | 路径 | 功能 | Content-Type |
|------|------|------|-------------|
| GET | `/` | 渲染表单页面 | `text/html` |
| POST | `/convert` | SSE 流式推送转换结果 | `text/event-stream` |
| POST | `/download` | 下载单个 .md 文件 | `application/octet-stream` |
| POST | `/download-all` | 打包下载所有结果为 .zip | `application/zip` |

### 5.2 GET /

渲染 `index.html` 模板，展示空表单。无参数。

### 5.3 POST /convert

**请求**：`application/x-www-form-urlencoded`

| 字段 | 类型 | 说明 |
|------|------|------|
| `urls` | string | 多行文本，每行一个 GitHub URL |
| `enable_reactions` | string | "on" 或缺失 |
| `enable_user_links` | string | "on" 或缺失 |

**处理流程**：

1. 解析 `urls` 字段，按换行分割
2. 去除空行、首尾空白
3. 去重（相同 URL 只保留第一次出现）
4. 校验：空列表返回 `event: error`；超过 20 条返回 `event: error`
5. 设置响应头：`Content-Type: text/event-stream`、`Cache-Control: no-cache`、`Connection: keep-alive`
6. 发送 `event: start`，data 包含 `{"total": N}`
7. 顺序遍历每个 URL：
   - 调用 `s.convert(&buf, &cli.RunOptions{URL: url, ...})`
   - 构造 `ConvertResult`
   - 发送 `event: result`，data 为 JSON 序列化的 `ConvertResult`
   - 调用 `flusher.Flush()`
8. 所有 URL 处理完毕后，发送 `event: done`，data 包含 `{"completed": M, "failed": F}`
9. 如果客户端断开（`r.Context().Done()`），提前终止

**SSE 事件格式**：

```
event: start
data: {"total": 3}

event: result
data: {"url":"https://...","success":true,"markdown":"---\ntitle:...","filename":"golang_go_issue_1.md","index":1,"total":3}

event: result
data: {"url":"https://...","success":false,"error":"fetch issue: API request failed with status 404","index":2,"total":3}

event: result
data: {"url":"https://...","success":true,"markdown":"---\ntitle:...","filename":"cli_cli_pr_1234.md","index":3,"total":3}

event: done
data: {"completed": 2, "failed": 1}
```

### 5.4 POST /download

**请求**：`application/x-www-form-urlencoded`

| 字段 | 类型 | 说明 |
|------|------|------|
| `filename` | string | 文件名（如 `golang_go_issue_1.md`） |
| `content` | string | Markdown 内容 |

**响应**：
- `Content-Type: application/octet-stream`
- `Content-Disposition: attachment; filename="golang_go_issue_1.md"`
- Body: Markdown 原文

校验：`filename` 或 `content` 为空时返回 400。

### 5.5 POST /download-all

**请求**：`application/x-www-form-urlencoded`

| 字段 | 类型 | 说明 |
|------|------|------|
| `files` | string | JSON 数组，每项为 `{"filename": "xxx.md", "content": "..."}` |

**响应**：
- `Content-Type: application/zip`
- `Content-Disposition: attachment; filename="issue2md-export.zip"`
- Body: ZIP 文件，包含所有 .md 文件

校验：`files` 为空或解析失败时返回 400。

---

## 6. 前端设计

### 6.1 页面结构

单页设计，所有状态通过 JS 在客户端管理：

1. **顶栏**：深色背景（#24292e），显示 "issue2md" 标题 + 说明文字
2. **表单区**：
   - URL textarea（每行一个，placeholder 示例 3 条 URL）
   - 两个复选框：显示 Reactions、启用用户链接
   - 绿色"开始转换"按钮（#2da44e）
3. **进度区**（转换开始后显示）：
   - 文字 "N / M 完成"
   - 绿色进度条，宽度按百分比动态更新
4. **结果区**（逐条追加）：
   - 每个 URL 一张卡片
   - 成功：绿色勾 ✓ + `owner/repo#number` + 类型标签（issue/pr/discussion）+ markdown 摘要预览（YAML frontmatter 前 3-4 行，max-height: 60px）+ "展开/收起"按钮 + "下载 .md"按钮
   - 失败：红色叉 ✗ + 错误信息，红色背景
   - 展开后：完整 markdown 内容，`<pre>` 标签，max-height: 400px + overflow-y: auto 滚动
5. **底部操作栏**（有成功结果时显示）：蓝色"全部下载 (.zip)"按钮（#0969da）

### 6.2 CSS 方案

内联在 `index.html` 的 `<style>` 标签中，GitHub 风格配色：
- 背景：#fff / #f6f8fa
- 边框：#d0d7de / #e1e4e8
- 文字：#1a1a1a / #57606a
- 成功绿：#2da44e
- 错误红：#cf222e
- 链接蓝：#0969da

无外部 CSS 文件、无 CSS 框架。

### 6.3 JavaScript 方案

Vanilla JS，无框架，内联在 `index.html` 的 `<script>` 标签中。功能：

1. **SSE 监听**：`EventSource` 或 `fetch` + `ReadableStream` 监听 POST /convert 响应
   - 注意：标准 `EventSource` 只支持 GET。POST 场景使用 `fetch` + 流式读取 + 手动解析 SSE 格式
2. **动态 DOM**：收到 `event: result` 时，创建结果卡片 DOM 插入页面
3. **进度更新**：收到 `event: result` 时更新进度条宽度和文字
4. **结果收集**：JS 数组 `results[]` 在内存中收集所有成功的 `{filename, content}`
5. **展开/收起**：`toggleExpand(el)` 切换预览区 display
6. **单个下载**：构造隐藏 form，POST 到 `/download`
7. **全部下载**：构造隐藏 form，将 `results[]` JSON 序列化后 POST 到 `/download-all`
8. **表单状态**：转换进行中禁用提交按钮，完成后恢复

### 6.4 SSE 客户端实现说明

由于 `EventSource` API 不支持 POST 请求，前端使用 `fetch` API 发送 POST 请求，然后通过 `response.body.getReader()` 流式读取响应，手动解析 SSE 格式（`event:` 和 `data:` 行）。

```javascript
// 伪代码
const response = await fetch('/convert', { method: 'POST', body: formData });
const reader = response.body.getReader();
const decoder = new TextDecoder();
// 逐块读取，按换行分割，解析 event/data 对
```

---

## 7. Token 处理

服务端读取 `GITHUB_TOKEN` 环境变量，与 CLI 行为完全一致。用户无感知。

`cli.Run()` 内部调用 `os.Getenv("GITHUB_TOKEN")` 创建 GitHub client，Web 服务不做额外处理。

---

## 8. 输入校验与错误处理

### 8.1 输入校验

| 场景 | 行为 |
|------|------|
| URL 列表为空 | SSE 发送 `event: error`，data 包含错误信息 |
| 超过 20 条 URL | SSE 发送 `event: error`，data 包含错误信息 |
| 重复 URL | 自动去重，只保留第一次出现 |
| 空行/纯空白行 | 自动过滤 |
| 无效 URL 格式 | 该 URL 的 `ConvertResult.Success = false`，继续处理下一个 |
| GitHub API 404 | 该 URL 的 `ConvertResult.Success = false`，Error 字段包含错误信息 |
| GitHub API 超时 | 同上 |

### 8.2 错误处理原则

- 批量处理中，单个 URL 失败不影响其他 URL（隔离性）
- 所有错误通过 `ConvertResult.Error` 字段返回给前端
- 错误信息使用 `fmt.Errorf("...: %w", err)` 链式包装（遵守宪法 3.1）
- 服务端日志记录所有错误（`log.Printf`）

---

## 9. 测试策略

### 9.1 测试原则

- 严格遵循 TDD：Red → Green → Refactor
- 所有测试为表格驱动（遵守宪法 2.2）
- 通过 `convertFunc` 依赖注入实现 mock，不调用真实 GitHub API
- 使用 `httptest` 做真实 HTTP 调用

### 9.2 测试文件

| 文件 | 测试类型 | 构建标签 |
|------|----------|----------|
| `handler_test.go` | 单元测试 + SSE 集成测试 | 无（默认运行） |
| `handler_integration_test.go` | 端到端集成测试 | `//go:build integration` |

### 9.3 Mock 策略

```go
// 测试用 mock convertFunc
func mockConvertSuccess(w io.Writer, opts *cli.RunOptions) error {
    fmt.Fprintf(w, "---\ntitle: \"Test Issue\"\n---\n# Test\n")
    return nil
}

func mockConvertError(w io.Writer, opts *cli.RunOptions) error {
    return fmt.Errorf("fetch issue: API request failed with status 404")
}
```

### 9.4 单元测试用例清单

**handleIndex 测试：**

| 用例 | 输入 | 期望 |
|------|------|------|
| 正常渲染 | GET / | 200, body 包含 `<form>`, `<textarea>`, 提交按钮 |

**handleConvert 测试（SSE）：**

| 用例 | 输入 | 期望 |
|------|------|------|
| 空 URL 列表 | POST urls="" | event:error, 错误信息 |
| 超过 20 条 URL | POST 21 个 URL | event:error, 超限错误信息 |
| 1 个有效 URL | POST 1 个 URL, mock 成功 | event:start → event:result(success) → event:done |
| 多个 URL 部分失败 | POST 3 个 URL, mock 2 成功 1 失败 | event:start(total:3) → 3x event:result → event:done(completed:2, failed:1) |
| 选项传递 | POST enable_reactions=on | mock convertFunc 收到 EnableReactions=true |
| 响应头 | POST 有效请求 | Content-Type: text/event-stream |
| 去重 | POST 相同 URL 两次 | 只处理一次 |
| 纯空白行 | POST urls="  \n\n  " | event:error, 空列表 |

**handleDownload 测试：**

| 用例 | 输入 | 期望 |
|------|------|------|
| 正常下载 | filename + content | 200, Content-Disposition: attachment, body = content |
| 空 filename | content only | 400 |
| 空 content | filename only | 400 |

**handleDownloadAll 测试：**

| 用例 | 输入 | 期望 |
|------|------|------|
| 正常打包 | 2 个文件的 JSON | 200, Content-Type: application/zip, zip 内含 2 个 .md 文件 |
| 空列表 | files=[] | 400 |
| JSON 格式错误 | files=invalid | 400 |

### 9.5 SSE 流测试方法

```go
// 使用 httptest.NewRecorder 捕获响应
// 逐行读取 body，解析 "event: xxx" 和 "data: {...}" 行
// 断言事件顺序：start → result × N → done
// 断言 data 字段 JSON 反序列化后的值
```

### 9.6 覆盖率目标

- `handler.go` 覆盖率 ≥ 90%
- `server.go` 覆盖率 ≥ 80%
- 整体项目覆盖率保持 ≥ 80%

---

## 10. 构建与部署

### 10.1 构建

Makefile `build-web` target 已就绪，编译产出 `bin/issue2md-web`：

```bash
make build-web    # 编译 Web 服务
make build        # 编译 CLI + Web
make test         # 运行所有测试（含 handler 测试）
```

### 10.2 运行

```bash
# 默认端口 8080
GITHUB_TOKEN=ghp_xxx ./bin/issue2md-web

# 自定义端口
GITHUB_TOKEN=ghp_xxx ./bin/issue2md-web -port 3000
```

### 10.3 Docker

现有 Dockerfile 已支持多阶段构建。Web 服务可作为独立 binary 打入镜像，或与 CLI 共用同一镜像（后续决定）。

---

## 11. 验证方案

### 11.1 自动化验证

```bash
make test              # 所有单元测试通过
make build-web         # 编译成功，产出 bin/issue2md-web
```

### 11.2 手动验证

1. 启动 `GITHUB_TOKEN=ghp_xxx ./bin/issue2md-web -port 8080`
2. 浏览器访问 `http://localhost:8080`
3. 输入 3 个不同类型的 GitHub URL（issue、PR、discussion）
4. 勾选"显示 Reactions"
5. 点击"开始转换"
6. 验证：进度条实时更新、结果卡片逐条出现
7. 点击"展开"查看完整 markdown 内容
8. 下载单个 .md 文件，确认内容正确
9. 点击"全部下载 (.zip)"，确认 zip 包含所有成功的 .md 文件
10. 测试错误场景：输入无效 URL，确认错误卡片正确显示

### 11.3 覆盖率验证

```bash
go test ./cmd/issue2mdweb/ -coverprofile=coverage.out
go tool cover -func=coverage.out
# handler.go 覆盖率 ≥ 90%
```
