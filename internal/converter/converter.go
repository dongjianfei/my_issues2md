package converter

import (
	"fmt"
	"strings"
	"time"

	"github.com/dongjianfei/issue2md/internal/github"
)

// ConvertOptions 控制Markdown输出的可选功能
type ConvertOptions struct {
	EnableReactions bool
	EnableUserLinks bool
}

// formatReactions 格式化反应统计为Markdown字符串
// 全部为0时返回空字符串，否则仅显示>0的反应
// 格式: **Reactions:** 👍 5 | ❤️ 3
func formatReactions(r github.Reaction) string {
	type reactionItem struct {
		emoji string
		count int
	}

	// 按固定顺序定义反应: 👍 👎 😄 😕 ❤️ 🎉 🚀 👀
	reactions := []reactionItem{
		{"👍", r.PlusOne},
		{"👎", r.MinusOne},
		{"😄", r.Laugh},
		{"😕", r.Confused},
		{"❤️", r.Heart},
		{"🎉", r.Hooray},
		{"🚀", r.Rocket},
		{"👀", r.Eyes},
	}

	var parts []string
	for _, item := range reactions {
		if item.count > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", item.emoji, item.count))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "**Reactions:** " + strings.Join(parts, " | ")
}

// formatUser 格式化用户为Markdown字符串
// enableLinks=false: @username
// enableLinks=true: [@username](https://github.com/username)
func formatUser(u github.User, enableLinks bool) string {
	if enableLinks {
		return fmt.Sprintf("[@%s](https://github.com/%s)", u.Login, u.Login)
	}
	return fmt.Sprintf("@%s", u.Login)
}

// formatTime 格式化时间为标准字符串
// 格式: 2026-05-20 10:30:00 UTC
func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05 MST")
}

// formatLabels 格式化标签列表为Markdown字符串
// 空列表返回空字符串
// 格式: `bug`, `enhancement`
func formatLabels(labels []github.Label) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("`%s`", label.Name))
	}

	return strings.Join(parts, ", ")
}
