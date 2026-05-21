package converter

import (
	"fmt"
	"strings"

	"github.com/dongjianfei/issue2md/internal/github"
)

// ConvertPullRequest 将PR转换为Markdown字符串
func ConvertPullRequest(pr *github.PullRequest, opts ConvertOptions) string {
	var b strings.Builder

	// Frontmatter
	writePRFrontmatter(&b, pr)

	// Title
	fmt.Fprintf(&b, "# %s\n\n", pr.Title)

	// Meta info
	writePRMetaInfo(&b, pr, opts)

	b.WriteString("---\n\n")

	// Description
	b.WriteString("## Description\n\n")
	b.WriteString(pr.Body)
	b.WriteString("\n\n")

	// Reactions (on main PR body)
	if opts.EnableReactions {
		reactions := formatReactions(pr.Reactions)
		if reactions != "" {
			b.WriteString(reactions)
			b.WriteString("\n\n")
		}
	}

	b.WriteString("---\n")

	// Comments
	if len(pr.Comments) > 0 {
		b.WriteString("\n## Comments\n")
		for _, comment := range pr.Comments {
			b.WriteString("\n")
			writePRComment(&b, comment, opts)
		}
	}

	return b.String()
}

// writePRFrontmatter writes YAML frontmatter for a PullRequest.
func writePRFrontmatter(b *strings.Builder, pr *github.PullRequest) {
	b.WriteString("---\n")
	fmt.Fprintf(b, "title: %q\n", pr.Title)
	fmt.Fprintf(b, "url: %q\n", pr.HTMLURL)
	b.WriteString("type: \"pull_request\"\n")
	fmt.Fprintf(b, "author: %q\n", pr.Author.Login)
	fmt.Fprintf(b, "created_at: %q\n", pr.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(b, "updated_at: %q\n", pr.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(b, "state: %q\n", pr.State)
	writeLabelsYAML(b, pr.Labels)
	fmt.Fprintf(b, "comments_count: %d\n", pr.CommentCount)
	b.WriteString("---\n\n")
}

// writePRMetaInfo writes the meta info section for a PullRequest.
func writePRMetaInfo(b *strings.Builder, pr *github.PullRequest, opts ConvertOptions) {
	userStr := formatUser(pr.Author, opts.EnableUserLinks)
	fmt.Fprintf(b, "**Author:** %s\n", userStr)
	fmt.Fprintf(b, "**Created:** %s\n", formatTime(pr.CreatedAt))
	fmt.Fprintf(b, "**Status:** %s\n", capitalizeState(pr.State))

	labelsStr := formatLabels(pr.Labels)
	if labelsStr != "" {
		fmt.Fprintf(b, "**Labels:** %s\n", labelsStr)
	} else {
		b.WriteString("**Labels:**\n")
	}

	b.WriteString("\n")
}

// writePRComment writes a single comment section, distinguishing normal and review comments.
func writePRComment(b *strings.Builder, comment github.Comment, opts ConvertOptions) {
	userStr := formatUser(comment.Author, opts.EnableUserLinks)
	timeStr := formatTime(comment.CreatedAt)

	if comment.IsReview {
		fmt.Fprintf(b, "### Review Comment by %s on %s\n\n", userStr, timeStr)
	} else {
		fmt.Fprintf(b, "### Comment by %s on %s\n\n", userStr, timeStr)
	}

	b.WriteString(comment.Body)
	b.WriteString("\n\n")

	if opts.EnableReactions {
		reactions := formatReactions(comment.Reactions)
		if reactions != "" {
			b.WriteString(reactions)
			b.WriteString("\n\n")
		}
	}

	b.WriteString("---\n")
}
