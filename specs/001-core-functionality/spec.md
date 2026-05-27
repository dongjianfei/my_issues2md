# Spec 001: Core Functionality - GitHub Content to Markdown Converter

## 版本信息
- **版本**: 1.0
- **创建日期**: 2026-05-20
- **状态**: Draft

---

## 1. 用户故事

### 1.1 MVP用户故事（本次实现）

**作为** 一名开发者  
**我想要** 通过命令行工具将GitHub Issue/PR/Discussion转换为Markdown文件  
**以便于** 我可以离线阅读、归档保存或进行二次加工

**验收条件：**
- 输入一个GitHub URL，自动识别类型（Issue/PR/Discussion）
- 输出标准的GitHub Flavored Markdown格式
- 包含完整的主楼内容和所有评论
- 支持输出到stdout或指定文件
- 可选地包含Reactions统计和用户链接

### 1.2 未来用户故事（暂不实现）

**作为** 一名团队管理者  
**我想要** 通过Web界面批量转换多个Issue/PR  
**以便于** 我可以快速归档整个项目的讨论历史

---

## 2. 功能性需求

### 2.1 支持的内容类型

工具必须支持以下三种GitHub内容类型：

1. **Issue**
   - 标题、作者、创建时间、状态（Open/Closed）
   - 主楼描述内容
   - 所有评论（按时间正序）
   - Labels（标签）
   - 可选：Reactions统计

2. **Pull Request (PR)**
   - 标题、作者、创建时间、状态（Open/Closed/Merged）
   - 主楼描述内容
   - 所有评论（包括普通评论和Review评论，按时间正序混合排列）
   - **不包含**：代码diff、commits历史
   - 可选：Reactions统计

3. **Discussion**
   - 标题、作者、创建时间、状态
   - 主楼描述内容
   - 所有评论（按时间正序）
   - 如果某个评论被标记为Answer，需要特殊标记（使用✅或引用块）
   - 可选：Reactions统计

### 2.2 URL识别与解析

**自动类型识别：** 工具必须能够自动解析URL并识别内容类型，用户无需手动指定。

**支���的URL格式：**
```
https://github.com/{owner}/{repo}/issues/{number}
https://github.com/{owner}/{repo}/pull/{number}
https://github.com/{owner}/{repo}/discussions/{number}
```

**URL验证规则：**
- 必须是有效的GitHub URL
- 必须包含owner、repo和number
- 如果URL格式无效，立即报错退出

### 2.3 命令行接口设计

**基本语法：**
```bash
issue2md [flags] <url> [output_file]
```

**位置参数：**
- `<url>` (必需): GitHub Issue/PR/Discussion的完整URL
- `[output_file]` (可选): 输出文件路径。如果不提供，输出到stdout

**可选Flags：**
- `-enable-reactions`: 在主楼和评论下方显示Reactions统计
- `-enable-user-links`: 将用户名渲染为指向其GitHub主页的链接

**环境变量：**
- `GITHUB_TOKEN`: GitHub Personal Access Token（可选）
  - 用于提高API rate limit
  - 用于访问私有仓库（未来支持）
  - **不提供**命令行参数传入token，避免在Shell历史中泄露

**使用示例：**
```bash
# 输出到stdout
issue2md https://github.com/owner/repo/issues/123

# 输出到文件
issue2md https://github.com/owner/repo/issues/123 output.md

# 启用Reactions和用户链接
issue2md -enable-reactions -enable-user-links https://github.com/owner/repo/pull/456 pr-456.md

# 使用环境变量提供token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
issue2md https://github.com/owner/repo/discussions/789
```

### 2.4 输出格式规范

**文件格式：** GitHub Flavored Markdown (GFM)

**文件结构：**
1. YAML Frontmatter（元数据）
2. 标题
3. 元信息表格
4. 主楼内容
5. 评论列表

**YAML Frontmatter字段：**
```yaml
---
title: "Issue/PR/Discussion标题"
url: "原始GitHub URL"
type: "issue" | "pull_request" | "discussion"
author: "作者用户名"
created_at: "2026-05-20T10:30:00Z"
updated_at: "2026-05-20T15:45:00Z"
state: "open" | "closed" | "merged"
labels: ["bug", "enhancement"]
comments_count: 15
---
```

### 2.5 评论处理规则

**评论排序：** 所有评论统一按创建时间正序排列

**PR评论合并：** 对于Pull Request：
- 普通评论（Issue Comments）
- Review评论（Review Comments）
- 以上两种评论按时间线混合排列，不做分组

**Discussion Answer标记：** 如果某个评论被标记为Answer：
```markdown
> ✅ **Accepted Answer**

[评论内容]
```

**Reactions显示：** 当启用`-enable-reactions`时：
```markdown
**Reactions:** 👍 5 | ❤️ 3 | 🎉 1
```

### 2.6 图片与附件处理

**图片链接：** 保留原始的GitHub图片URL，不下载到本地

**附件链接：** 保留原始链接

**代码块：** 保持原样输出

### 2.7 认证与权限

**MVP阶段：** 仅支持公有仓库

**Token使用：**
- 通过环境变量`GITHUB_TOKEN`传入
- 如果未提供token，使用GitHub API的匿名访问（rate limit较低）
- 如果提供token，提高rate limit

**未来扩展：** 支持私有仓库访问

---

## 3. 非功能性需求

### 3.1 架构原则

**遵循项目宪法：**
- **简单性原则**：优先使用Go标准库，避免不必要的依赖
- **测试先行**：所有功能必须先编写测试
- **明确性原则**：��式错误处理，无全局变量

**模块解耦：**
- GitHub API客户端层
- URL解析层
- Markdown生成层
- CLI命令层

### 3.2 错误处理策略

**错误类型与处理：**

| 错误类型 | 处理方式 | 退出码 |
|---------|---------|--------|
| URL格式无效 | 输出清晰错误信息到stderr，立即退出 | 1 |
| 资源不存在（404） | 输出"Issue/PR/Discussion not found"，立即退出 | 1 |
| 网络超时 | 输出超时错误，立即退出 | 1 |
| API rate limit超限 | 透传GitHub API错误信息，立即退出 | 1 |
| 无权限访问（403） | 输出权限错误，提示检查token，立即退出 | 1 |

**错误信息格式：**
```
Error: [错误类型] - [详细描述]
```

**不实现的功能：**
- 自动重试机制
- 复杂的错误恢复
- 进度条（MVP阶段）

### 3.3 性能要求

**API调用优化：**
- 使用GitHub API的分页机制获取所有评论
- 无评论数量限制，获取所有内容

**响应时间：**
- 对于少量评论（<50条）：< 5秒
- 对于大量评论（100+条）：可接受较长时间，但不��进��显示（MVP阶段）

### 3.4 依赖管理

**必需依赖：**
- Go标准库（`net/http`, `encoding/json`, `flag`等）
- GitHub API客户端库（推荐使用官方或社区成熟库）

**禁止依赖：**
- 重量级框架
- 非必需的第三方库

---

## 4. 验收标准

### 4.1 功能验收测试用例

**TC-001: Issue转换**
- **前置条件：** 存在一个公开的GitHub Issue
- **输入：** `issue2md https://github.com/owner/repo/issues/123`
- **预期输出：**
  - 包含YAML Frontmatter
  - 包含Issue标题、作者、状态
  - 包含主楼内容
  - 包含所有评论（按时间正序）
  - 输出到stdout

**TC-002: PR转换**
- **前置条件：** 存在一个公开的GitHub PR
- **输入：** `issue2md https://github.com/owner/repo/pull/456 output.md`
- **预期输出：**
  - 生成`output.md`文件
  - 包含PR描述和所有评论
  - 不包含代码diff
  - Review评论和普通评论按时间混合排列

**TC-003: Discussion转换**
- **前置条件：** 存在一个公开的GitHub Discussion，其中一个评论被标记为Answer
- **输入：** `issue2md https://github.com/owner/repo/discussions/789`
- **预期输出：**
  - Answer评论有特殊标记（✅）
  - 其他评论按时间正序排列

**TC-004: Reactions功能**
- **输入：** `issue2md -enable-reactions https://github.com/owner/repo/issues/123`
- **预期输出：**
  - 主楼和评论下方显示Reactions统计
  - 格式：`**Reactions:** 👍 5 | ❤️ 3`

**TC-005: 用户链接功能**
- **输入：** `issue2md -enable-user-links https://github.com/owner/repo/issues/123`
- **预期输出：**
  - 用户名渲染为链接：`[@username](https://github.com/username)`

**TC-006: URL格式错误**
- **输入：** `issue2md https://invalid-url`
- **预期输出：**
  - 输出错误信息到stderr
  - 退出码为1

**TC-007: 资源不存在**
- **输入：** `issue2md https://github.com/owner/repo/issues/999999`
- **预期输出：**
  - 输出"Issue not found"错误
  - 退出码为1

**TC-008: 使用Token**
- **前置条件：** 设置环境变量`GITHUB_TOKEN`
- **输入：** `issue2md https://github.com/owner/repo/issues/123`
- **预期输出：**
  - 成功使用token访问API
  - 不在命令行或日志中泄露token

### 4.2 非功能验收标准

**代码质量：**
- 所有功能必须有对应的单元测试
- 测试覆盖率 > 80%
- 使用表格驱动测试（Table-Driven Tests）

**错误处理：**
- 所有错误必须显式处理
- 错误传递使用`fmt.Errorf("...: %w", err)`包装

**文档：**
- README包含安装和使用说明
- 代码注释清晰（仅在逻辑不自明时添加）

---

## 5. 输出格式示例

### 5.1 Issue输出示例

```markdown
---
title: "Bug: Application crashes on startup"
url: "https://github.com/owner/repo/issues/123"
type: "issue"
author: "johndoe"
created_at: "2026-05-15T10:30:00Z"
updated_at: "2026-05-20T15:45:00Z"
state: "open"
labels: ["bug", "priority-high"]
comments_count: 5
---

# Bug: Application crashes on startup

**Author:** [@johndoe](https://github.com/johndoe)  
**Created:** 2026-05-15 10:30:00 UTC  
**Status:** Open  
**Labels:** `bug`, `priority-high`

---

## Description

The application crashes immediately after startup with the following error:

\`\`\`
Error: Cannot read property 'foo' of undefined
\`\`\`

**Reactions:** 👍 3 | 😕 1

---

## Comments

### Comment by [@janedoe](https://github.com/janedoe) on 2026-05-15 11:00:00 UTC

I can reproduce this issue on macOS 13.2.

**Reactions:** 👍 2

---

### Comment by [@johndoe](https://github.com/johndoe) on 2026-05-15 14:30:00 UTC

Thanks for confirming! I'll investigate the root cause.

---
```

### 5.2 PR输出示例

```markdown
---
title: "feat: Add user authentication"
url: "https://github.com/owner/repo/pull/456"
type: "pull_request"
author: "developer"
created_at: "2026-05-18T09:00:00Z"
updated_at: "2026-05-20T16:00:00Z"
state: "merged"
labels: ["enhancement"]
comments_count: 8
---

# feat: Add user authentication

**Author:** [@developer](https://github.com/developer)  
**Created:** 2026-05-18 09:00:00 UTC  
**Status:** Merged  
**Labels:** `enhancement`

---

## Description

This PR implements user authentication using JWT tokens.

Changes:
- Add login endpoint
- Add token validation middleware
- Update user model

---

## Comments

### Comment by [@reviewer1](https://github.com/reviewer1) on 2026-05-18 10:30:00 UTC

Looks good overall! Just a few minor suggestions.

---

### Review Comment by [@reviewer1](https://github.com/reviewer1) on 2026-05-18 10:35:00 UTC

Consider adding rate limiting to the login endpoint.

---

### Comment by [@developer](https://github.com/developer) on 2026-05-18 14:00:00 UTC

Good point! I'll add rate limiting in the next commit.

---
```

### 5.3 Discussion输出示例

```markdown
---
title: "How to configure database connection?"
url: "https://github.com/owner/repo/discussions/789"
type: "discussion"
author: "newuser"
created_at: "2026-05-19T08:00:00Z"
updated_at: "2026-05-20T12:00:00Z"
state: "open"
labels: ["question"]
comments_count: 3
---

# How to configure database connection?

**Author:** [@newuser](https://github.com/newuser)  
**Created:** 2026-05-19 08:00:00 UTC  
**Status:** Open  
**Category:** Q&A

---

## Question

I'm trying to configure the database connection but getting errors. What's the correct way to do this?

---

## Comments

### Comment by [@expert](https://github.com/expert) on 2026-05-19 09:30:00 UTC

> ✅ **Accepted Answer**

You need to set the following environment variables:

\`\`\`bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=myapp
\`\`\`

Then the application will automatically connect.

---

### Comment by [@newuser](https://github.com/newuser) on 2026-05-19 10:00:00 UTC

Perfect! That worked. Thank you!

**Reactions:** 🎉 1 | ❤️ 1

---
```

---

## 6. 实现优先级

### Phase 1: MVP核心功能（本次实现）
1. URL解析与类型识别
2. GitHub API集成（Issue/PR/Discussion）
3. Markdown生成（包含Frontmatter）
4. CLI命令行接口
5. 基本错误处理
6. 单元测试与集成测试

### Phase 2: 可选功能（本次实现）
1. Reactions支持（`-enable-reactions`）
2. 用户链接支持（`-enable-user-links`）
3. Discussion Answer标记

### Phase 3: 未来扩展（暂不实现）
1. Web界面
2. 批量转换
3. 私有仓库支持
4. 自定义模板
5. 进度显示
6. 自动重试机制

---

## 7. 技术约束

**Go版本：** >= 1.24

**构建工具：** Makefile

**测试框架：** Go标准库 `testing`

**GitHub API：** REST API v3 或 GraphQL API v4（根据实现需要选择）

**依赖管理：** Go Modules

---

## 8. 参考资料

- [GitHub REST API Documentation](https://docs.github.com/en/rest)
- [GitHub GraphQL API Documentation](https://docs.github.com/en/graphql)
- [GitHub Flavored Markdown Spec](https://github.github.com/gfm/)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

## 9. 变更历史

| 版本 | 日期 | 作者 | 变更说明 |
|-----|------|------|---------|
| 1.0 | 2026-05-20 | CTO & Claude | 初始版本，定义MVP核心功能 |

