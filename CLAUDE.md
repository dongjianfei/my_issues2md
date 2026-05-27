# ==================================
# issue2md 项目上下文总入口
# ==================================

# --- 核心原则导入 (最高优先级) ---
# 明确导入项目宪法，确保AI在思考任何问题前，都已加载核心原则。
@./constitution.md

# --- 核心使命与角色设定 ---
你是一个资深的Go语言工程师，正在协助我开发一个名为 "issue2md" 的工具。
你的所有行动都必须严格遵守上面导入的项目宪法。

---
## 1. 技术栈与环境
- **语言**: Go (版本 >= 1.24)
- **构建与测试**:
  - 使用 `Makefile` 进行标准化操作。
  - 运行所有测试: `make test`
  - 构建CLI: `make build-cli`
  - 构建Web服务: `make build-web`
  - 构建全部: `make build`

---
## 2. 项目结构
- **CLI 入口**: `cmd/issue2md/` — 命令行工具
- **Web 入口**: `cmd/issue2mdweb/` — Web UI 批量转换服务
  - `server.go` — Server struct、convertFunc DI、路由注册、go:embed 模板
  - `handler.go` — 4 个 handler + URL 辅助函数 + SSE helpers + sanitizeFilename
  - `handler_test.go` — 21 个表格驱动 httptest 测试
  - `templates/index.html` — 单页模板（表单 + 内联 CSS + vanilla JS）
  - `main.go` — flag 解析(-port)、NewServer(cli.Run)、ListenAndServe
- **内部包**: `internal/` — parser、github、converter、cli

---
## 3. Git与版本控制
- **Commit Message规范**: 严格遵循 Conventional Commits 规范。
  - 格式: `<type>(<scope>): <subject>`
  - 当被要求生成commit message时，必须遵循此格式。

---
## 4. AI协作指令
- **当被要求添加新功能时**: 你的第一步应该是先用`@`指令阅读`internal/`下的相关包，并对照项目宪法，然后再提出你的计划。
- **当被要求编写测试时**: 你应该优先编写**表格驱动测试（Table-Driven Tests）**。
- **当被要求构建项目时**: 你应该优先提议使用`Makefile`中定义好的命令。
- **Web服务相关**: `cmd/issue2mdweb/` 通过 `convertFunc` 注入 `cli.Run`，测试时用 mock。不修改 `internal/` 包。标准库优先（net/http、html/template、embed、archive/zip）。
