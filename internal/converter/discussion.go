package converter

import (
	"fmt"
	"strings"

	"github.com/dongjianfei/issue2md/internal/github"
)

// ConvertDiscussion 将Discussion转换为Markdown字符串
func ConvertDiscussion(disc *github.Discussion, opts ConvertOptions) string {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", disc.Title))
	b.WriteString(fmt.Sprintf("url: %q\n", disc.HTMLURL))
	b.WriteString("type: \"discussion\"\n")
	b.WriteString(fmt.Sprintf("author: %q\n", disc.Author.Login))
	b.WriteString(fmt.Sprintf("created_at: %q\n", disc.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")))
	b.WriteString(fmt.Sprintf("updated_at: %q\n", disc.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")))
	b.WriteString("state: \"open\"\n")
	b.WriteString("labels: []\n")
	b.WriteString(fmt.Sprintf("comments_count: %d\n", len(disc.Comments)))
	b.WriteString("---\n")

	// Title
	b.WriteString(fmt.Sprintf("\n# %s\n", disc.Title))

	// Meta info
	b.WriteString(fmt.Sprintf("\n**Author:** %s\n", formatUser(disc.Author, opts.EnableUserLinks)))
	b.WriteString(fmt.Sprintf("**Created:** %s\n", formatTime(disc.CreatedAt)))
	b.WriteString("**Status:** Open\n")
	b.WriteString(fmt.Sprintf("**Category:** %s\n", disc.Category))

	b.WriteString("\n---\n")

	// Question (main body)
	b.WriteString("\n## Question\n")
	b.WriteString(fmt.Sprintf("\n%s\n", disc.Body))

	// Reactions (on main discussion body)
	if opts.EnableReactions {
		reactions := formatReactions(disc.Reactions)
		if reactions != "" {
			b.WriteString(fmt.Sprintf("\n%s\n", reactions))
		}
	}

	b.WriteString("\n---\n")

	// Comments
	if len(disc.Comments) > 0 {
		b.WriteString("\n## Comments\n")

		for _, dc := range disc.Comments {
			b.WriteString(fmt.Sprintf("\n### Comment by %s on %s\n",
				formatUser(dc.Author, opts.EnableUserLinks),
				formatTime(dc.CreatedAt)))

			// Answer marker
			if dc.IsAnswer {
				b.WriteString("\n> ✅ **Accepted Answer**\n")
			}

			b.WriteString(fmt.Sprintf("\n%s\n", dc.Body))

			// Reactions
			if opts.EnableReactions {
				reactions := formatReactions(dc.Reactions)
				if reactions != "" {
					b.WriteString(fmt.Sprintf("\n%s\n", reactions))
				}
			}

			b.WriteString("\n---\n")
		}
	}

	return b.String()
}
