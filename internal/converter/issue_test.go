package converter

import (
	"strings"
	"testing"
	"time"

	"github.com/dongjianfei/issue2md/internal/github"
)

func TestConvertIssue(t *testing.T) {
	baseTime := time.Date(2026, 5, 15, 10, 30, 0, 0, time.UTC)
	updateTime := time.Date(2026, 5, 20, 15, 45, 0, 0, time.UTC)
	commentTime := time.Date(2026, 5, 15, 11, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		issue   *github.Issue
		options ConvertOptions
		want    []string // 必须包含的字符串片段
		notWant []string // 不应包含的字符串片段
	}{
		{
			name: "basic issue without comments and reactions",
			issue: &github.Issue{
				Number:    123,
				Title:     "Bug: Application crashes",
				Body:      "Issue body content",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/issues/123",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "johndoe",
				},
				Labels:   []github.Label{},
				Comments: []github.Comment{},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				"---",
				`title: "Bug: Application crashes"`,
				`url: "https://github.com/owner/repo/issues/123"`,
				`type: "issue"`,
				`author: "johndoe"`,
				`created_at: "2026-05-15T10:30:00Z"`,
				`updated_at: "2026-05-20T15:45:00Z"`,
				`state: "open"`,
				`comments_count: 0`,
				"# Bug: Application crashes",
				"**Author:** [@johndoe](https://github.com/johndoe)",
				"**Created:** 2026-05-15 10:30:00 UTC",
				"**Status:** Open",
				"## Description",
				"Issue body content",
			},
			notWant: []string{
				"## Comments",
				"**Reactions:**",
			},
		},
		{
			name: "issue with 2 comments",
			issue: &github.Issue{
				Number:    456,
				Title:     "Feature request",
				Body:      "Feature description",
				State:     "closed",
				HTMLURL:   "https://github.com/owner/repo/issues/456",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "alice",
				},
				Labels: []github.Label{},
				Comments: []github.Comment{
					{
						Body:      "First comment",
						CreatedAt: commentTime,
						Author: github.User{
							Login: "bob",
						},
					},
					{
						Body:      "Second comment",
						CreatedAt: commentTime.Add(time.Hour),
						Author: github.User{
							Login: "charlie",
						},
					},
				},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				"## Comments",
				"### Comment by [@bob](https://github.com/bob) on 2026-05-15 11:00:00 UTC",
				"First comment",
				"### Comment by [@charlie](https://github.com/charlie) on 2026-05-15 12:00:00 UTC",
				"Second comment",
			},
		},
		{
			name: "issue with reactions enabled",
			issue: &github.Issue{
				Number:    789,
				Title:     "Bug report",
				Body:      "Bug description",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/issues/789",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "user1",
				},
				Labels: []github.Label{},
				Reactions: github.Reaction{
					PlusOne:  3,
					MinusOne: 0,
					Laugh:    1,
					Confused: 0,
					Heart:    2,
					Hooray:   0,
					Rocket:   0,
					Eyes:     0,
				},
				Comments: []github.Comment{
					{
						Body:      "Comment with reactions",
						CreatedAt: commentTime,
						Author: github.User{
							Login: "user2",
						},
						Reactions: github.Reaction{
							PlusOne: 2,
							Heart:   1,
						},
					},
				},
			},
			options: ConvertOptions{
				EnableReactions: true,
				EnableUserLinks: true,
			},
			want: []string{
				"**Reactions:** 👍 3 | 😄 1 | ❤️ 2",
				"### Comment by [@user2](https://github.com/user2) on 2026-05-15 11:00:00 UTC",
				"Comment with reactions",
				"**Reactions:** 👍 2 | ❤️ 1",
			},
		},
		{
			name: "issue with reactions disabled",
			issue: &github.Issue{
				Number:    790,
				Title:     "Test issue",
				Body:      "Test body",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/issues/790",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "user1",
				},
				Labels: []github.Label{},
				Reactions: github.Reaction{
					PlusOne: 5,
					Heart:   3,
				},
				Comments: []github.Comment{},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				"## Description",
				"Test body",
			},
			notWant: []string{
				"**Reactions:**",
			},
		},
		{
			name: "issue with user links enabled",
			issue: &github.Issue{
				Number:    800,
				Title:     "Link test",
				Body:      "Body",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/issues/800",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "testuser",
				},
				Labels:   []github.Label{},
				Comments: []github.Comment{},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				"**Author:** [@testuser](https://github.com/testuser)",
			},
		},
		{
			name: "issue with user links disabled",
			issue: &github.Issue{
				Number:    801,
				Title:     "No link test",
				Body:      "Body",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/issues/801",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "plainuser",
				},
				Labels:   []github.Label{},
				Comments: []github.Comment{},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: false,
			},
			want: []string{
				"**Author:** @plainuser",
			},
			notWant: []string{
				"[@plainuser]",
				"(https://github.com/plainuser)",
			},
		},
		{
			name: "issue with labels",
			issue: &github.Issue{
				Number:    900,
				Title:     "Labeled issue",
				Body:      "Issue with labels",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/issues/900",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "user1",
				},
				Labels: []github.Label{
					{Name: "bug"},
					{Name: "priority-high"},
					{Name: "needs-review"},
				},
				Comments: []github.Comment{},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				`labels: ["bug", "priority-high", "needs-review"]`,
				"**Labels:** `bug`, `priority-high`, `needs-review`",
			},
		},
		{
			name: "frontmatter validation",
			issue: &github.Issue{
				Number:    1000,
				Title:     "Frontmatter test",
				Body:      "Testing frontmatter",
				State:     "closed",
				HTMLURL:   "https://github.com/owner/repo/issues/1000",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login: "author",
				},
				Labels: []github.Label{
					{Name: "documentation"},
				},
				Comments: []github.Comment{
					{Body: "Comment 1", CreatedAt: commentTime, Author: github.User{Login: "user1"}},
					{Body: "Comment 2", CreatedAt: commentTime, Author: github.User{Login: "user2"}},
				},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				"---",
				`title: "Frontmatter test"`,
				`url: "https://github.com/owner/repo/issues/1000"`,
				`type: "issue"`,
				`author: "author"`,
				`created_at: "2026-05-15T10:30:00Z"`,
				`updated_at: "2026-05-20T15:45:00Z"`,
				`state: "closed"`,
				`labels: ["documentation"]`,
				`comments_count: 2`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertIssue(tt.issue, tt.options)

			// Check all required strings are present
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("ConvertIssue() missing expected string:\nwant: %q\ngot: %s", want, got)
				}
			}

			// Check all forbidden strings are absent
			for _, notWant := range tt.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("ConvertIssue() contains forbidden string:\nnotWant: %q\ngot: %s", notWant, got)
				}
			}
		})
	}
}
