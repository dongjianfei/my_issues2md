package converter

import (
	"strings"
	"testing"
	"time"

	"github.com/dongjianfei/issue2md/internal/github"
)

func TestConvertPullRequest(t *testing.T) {
	baseTime := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	updateTime := time.Date(2026, 5, 20, 16, 0, 0, 0, time.UTC)
	commentTime := time.Date(2026, 5, 18, 10, 30, 0, 0, time.UTC)
	reviewTime := time.Date(2026, 5, 18, 10, 35, 0, 0, time.UTC)

	tests := []struct {
		name    string
		pr      *github.PullRequest
		options ConvertOptions
		want    []string
		notWant []string
	}{
		{
			name: "basic merged PR",
			pr: &github.PullRequest{
				Number:    456,
				Title:     "feat: Add user authentication",
				Body:      "PR description",
				State:     "merged",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login:   "developer",
					HTMLURL: "https://github.com/developer",
				},
				HTMLURL:      "https://github.com/owner/repo/pull/456",
				Labels:       []github.Label{{Name: "enhancement"}},
				Comments:     []github.Comment{},
				CommentCount: 0,
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				"---",
				`title: "feat: Add user authentication"`,
				`url: "https://github.com/owner/repo/pull/456"`,
				`type: "pull_request"`,
				`author: "developer"`,
				`created_at: "2026-05-18T09:00:00Z"`,
				`updated_at: "2026-05-20T16:00:00Z"`,
				`state: "merged"`,
				`labels: ["enhancement"]`,
				`comments_count: 0`,
				"# feat: Add user authentication",
				"[@developer](https://github.com/developer)",
				"2026-05-18 09:00:00 UTC",
				"**Status:** Merged",
				"`enhancement`",
				"## Description",
				"PR description",
			},
			notWant: []string{
				"## Comments",
				"**Reactions:**",
			},
		},
		{
			name: "PR with mixed comments (normal + review)",
			pr: &github.PullRequest{
				Number:    789,
				Title:     "fix: Resolve memory leak",
				Body:      "Fixed memory leak in cache layer",
				State:     "closed",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login:   "developer",
					HTMLURL: "https://github.com/developer",
				},
				HTMLURL: "https://github.com/owner/repo/pull/789",
				Labels:  []github.Label{},
				Comments: []github.Comment{
					{
						Author: github.User{
							Login:   "reviewer1",
							HTMLURL: "https://github.com/reviewer1",
						},
						Body:      "Looks good overall!",
						CreatedAt: commentTime,
						IsReview:  false,
					},
					{
						Author: github.User{
							Login:   "reviewer1",
							HTMLURL: "https://github.com/reviewer1",
						},
						Body:      "Consider adding rate limiting.",
						CreatedAt: reviewTime,
						IsReview:  true,
					},
				},
				CommentCount: 2,
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				`type: "pull_request"`,
				`state: "closed"`,
				`comments_count: 2`,
				"**Status:** Closed",
				"## Comments",
				"### Comment by [@reviewer1](https://github.com/reviewer1) on 2026-05-18 10:30:00 UTC",
				"Looks good overall!",
				"### Review Comment by [@reviewer1](https://github.com/reviewer1) on 2026-05-18 10:35:00 UTC",
				"Consider adding rate limiting.",
			},
			notWant: []string{
				"**Reactions:**",
			},
		},
		{
			name: "PR with reactions enabled",
			pr: &github.PullRequest{
				Number:    101,
				Title:     "feat: New feature",
				Body:      "Description",
				State:     "open",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login:   "dev",
					HTMLURL: "https://github.com/dev",
				},
				HTMLURL:      "https://github.com/owner/repo/pull/101",
				Labels:       []github.Label{},
				Comments:     []github.Comment{},
				CommentCount: 0,
				Reactions: github.Reaction{
					PlusOne: 5,
					Heart:   2,
				},
			},
			options: ConvertOptions{
				EnableReactions: true,
				EnableUserLinks: true,
			},
			want: []string{
				`type: "pull_request"`,
				`state: "open"`,
				"**Status:** Open",
				"**Reactions:**",
			},
			notWant: []string{},
		},
		{
			name: "PR with reactions disabled",
			pr: &github.PullRequest{
				Number:    102,
				Title:     "feat: Another feature",
				Body:      "Description",
				State:     "open",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login:   "dev",
					HTMLURL: "https://github.com/dev",
				},
				HTMLURL:      "https://github.com/owner/repo/pull/102",
				Labels:       []github.Label{},
				Comments:     []github.Comment{},
				CommentCount: 0,
				Reactions: github.Reaction{
					PlusOne: 5,
					Heart:   2,
				},
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: true,
			},
			want: []string{
				`type: "pull_request"`,
			},
			notWant: []string{
				"**Reactions:**",
			},
		},
		{
			name: "PR with userlinks disabled",
			pr: &github.PullRequest{
				Number:    202,
				Title:     "chore: Update deps",
				Body:      "Updated dependencies",
				State:     "merged",
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Author: github.User{
					Login:   "maintainer",
					HTMLURL: "https://github.com/maintainer",
				},
				HTMLURL: "https://github.com/owner/repo/pull/202",
				Labels:  []github.Label{},
				Comments: []github.Comment{
					{
						Author: github.User{
							Login:   "bot",
							HTMLURL: "https://github.com/bot",
						},
						Body:      "CI passed",
						CreatedAt: commentTime,
						IsReview:  false,
					},
				},
				CommentCount: 1,
			},
			options: ConvertOptions{
				EnableReactions: false,
				EnableUserLinks: false,
			},
			want: []string{
				`type: "pull_request"`,
				`author: "maintainer"`,
				`state: "merged"`,
				"**Status:** Merged",
				"**Author:** @maintainer",
				"### Comment by @bot on 2026-05-18 10:30:00 UTC",
				"CI passed",
			},
			notWant: []string{
				"[@maintainer]",
				"[@bot]",
				"**Reactions:**",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertPullRequest(tt.pr, tt.options)

			// Check all required strings are present
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("ConvertPullRequest() missing expected string:\nwant: %q\ngot:\n%s", want, got)
				}
			}

			// Check all forbidden strings are absent
			for _, notWant := range tt.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("ConvertPullRequest() contains forbidden string:\nnotWant: %q\ngot:\n%s", notWant, got)
				}
			}
		})
	}
}
