package converter

import (
	"fmt"
	"strings"

	"github.com/dongjianfei/issue2md/internal/github"
)

// ConvertIssue 将Issue转换为Markdown字符串
func ConvertIssue(issue *github.Issue, opts ConvertOptions) string {
	var b strings.Builder

	// Frontmatter
	writeFrontmatter(&b, issue)

	// Title
	fmt.Fprintf(&b, "# %s\n\n", issue.Title)

	// Meta info
	writeMetaInfo(&b, issue, opts)

	b.WriteString("---\n\n")

	// Description
	b.WriteString("## Description\n\n")
	b.WriteString(issue.Body)
	b.WriteString("\n\n")

	// Reactions (on main issue body)
	if opts.EnableReactions {
		reactions := formatReactions(issue.Reactions)
		if reactions != "" {
			b.WriteString(reactions)
			b.WriteString("\n\n")
		}
	}

	b.WriteString("---\n")

	// Comments
	if len(issue.Comments) > 0 {
		b.WriteString("\n## Comments\n")
		for _, comment := range issue.Comments {
			b.WriteString("\n")
			writeComment(&b, comment, opts)
		}
	}

	return b.String()
}

// writeFrontmatter writes YAML frontmatter for an Issue.
func writeFrontmatter(b *strings.Builder, issue *github.Issue) {
	b.WriteString("---\n")
	fmt.Fprintf(b, "title: %q\n", issue.Title)
	fmt.Fprintf(b, "url: %q\n", issue.HTMLURL)
	b.WriteString("type: \"issue\"\n")
	fmt.Fprintf(b, "author: %q\n", issue.Author.Login)
	fmt.Fprintf(b, "created_at: %q\n", issue.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(b, "updated_at: %q\n", issue.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(b, "state: %q\n", issue.State)
	writeLabelsYAML(b, issue.Labels)
	fmt.Fprintf(b, "comments_count: %d\n", len(issue.Comments))
	b.WriteString("---\n\n")
}

// writeLabelsYAML writes labels as a YAML array.
func writeLabelsYAML(b *strings.Builder, labels []github.Label) {
	if len(labels) == 0 {
		b.WriteString("labels: []\n")
		return
	}
	var parts []string
	for _, l := range labels {
		parts = append(parts, fmt.Sprintf("%q", l.Name))
	}
	fmt.Fprintf(b, "labels: [%s]\n", strings.Join(parts, ", "))
}

// writeMetaInfo writes the meta info section (Author, Created, Status, Labels).
func writeMetaInfo(b *strings.Builder, issue *github.Issue, opts ConvertOptions) {
	userStr := formatUser(issue.Author, opts.EnableUserLinks)
	fmt.Fprintf(b, "**Author:** %s  \n", userStr)
	fmt.Fprintf(b, "**Created:** %s  \n", formatTime(issue.CreatedAt))
	fmt.Fprintf(b, "**Status:** %s  \n", capitalizeState(issue.State))

	labelsStr := formatLabels(issue.Labels)
	if labelsStr != "" {
		fmt.Fprintf(b, "**Labels:** %s\n", labelsStr)
	}

	b.WriteString("\n")
}

// capitalizeState capitalizes the first letter of the state.
func capitalizeState(state string) string {
	if len(state) == 0 {
		return state
	}
	return strings.ToUpper(state[:1]) + state[1:]
}

// writeComment writes a single comment section.
func writeComment(b *strings.Builder, comment github.Comment, opts ConvertOptions) {
	userStr := formatUser(comment.Author, opts.EnableUserLinks)
	timeStr := formatTime(comment.CreatedAt)
	fmt.Fprintf(b, "### Comment by %s on %s\n\n", userStr, timeStr)
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
