package converter

import (
	"testing"
	"time"

	"github.com/dongjianfei/issue2md/internal/github"
)

func TestConvertDiscussion(t *testing.T) {
	baseTime := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
	updateTime := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	commentTime1 := time.Date(2026, 5, 19, 9, 30, 0, 0, time.UTC)
	commentTime2 := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		discussion *github.Discussion
		opts       ConvertOptions
		want       string
	}{
		{
			name: "basic discussion with category",
			discussion: &github.Discussion{
				Number:    789,
				Title:     "How to configure database connection?",
				Body:      "Question body",
				HTMLURL:   "https://github.com/owner/repo/discussions/789",
				Author:    github.User{Login: "newuser"},
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Category:  "Q&A",
			},
			opts: ConvertOptions{},
			want: `---
title: "How to configure database connection?"
url: "https://github.com/owner/repo/discussions/789"
type: "discussion"
author: "newuser"
created_at: "2026-05-19T08:00:00Z"
updated_at: "2026-05-20T12:00:00Z"
state: "open"
labels: []
comments_count: 0
---

# How to configure database connection?

**Author:** @newuser
**Created:** 2026-05-19 08:00:00 UTC
**Status:** Open
**Category:** Q&A

---

## Question

Question body

---
`,
		},
		{
			name: "discussion with accepted answer",
			discussion: &github.Discussion{
				Number:    789,
				Title:     "How to configure database connection?",
				Body:      "Question body",
				HTMLURL:   "https://github.com/owner/repo/discussions/789",
				Author:    github.User{Login: "newuser"},
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Category:  "Q&A",
				Comments: []github.DiscussionComment{
					{
						Comment: github.Comment{
							Body:      "You need to set DB_HOST=localhost",
							Author:    github.User{Login: "expert"},
							CreatedAt: commentTime1,
						},
						IsAnswer: true,
					},
					{
						Comment: github.Comment{
							Body:      "Perfect! Thanks!",
							Author:    github.User{Login: "newuser"},
							CreatedAt: commentTime2,
							Reactions: github.Reaction{Hooray: 1, Heart: 1},
						},
					},
				},
			},
			opts: ConvertOptions{
				EnableReactions: true,
				EnableUserLinks: true,
			},
			want: `---
title: "How to configure database connection?"
url: "https://github.com/owner/repo/discussions/789"
type: "discussion"
author: "newuser"
created_at: "2026-05-19T08:00:00Z"
updated_at: "2026-05-20T12:00:00Z"
state: "open"
labels: []
comments_count: 2
---

# How to configure database connection?

**Author:** [@newuser](https://github.com/newuser)
**Created:** 2026-05-19 08:00:00 UTC
**Status:** Open
**Category:** Q&A

---

## Question

Question body

---

## Comments

### Comment by [@expert](https://github.com/expert) on 2026-05-19 09:30:00 UTC

> ✅ **Accepted Answer**

You need to set DB_HOST=localhost

---

### Comment by [@newuser](https://github.com/newuser) on 2026-05-19 10:00:00 UTC

Perfect! Thanks!

**Reactions:** ❤️ 1 | 🎉 1

---
`,
		},
		{
			name: "discussion without answer - all comments normal",
			discussion: &github.Discussion{
				Number:    790,
				Title:     "Feature request",
				Body:      "Please add feature X",
				HTMLURL:   "https://github.com/owner/repo/discussions/790",
				Author:    github.User{Login: "user1"},
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Category:  "Ideas",
				Comments: []github.DiscussionComment{
					{
						Comment: github.Comment{
							Body:      "Good idea!",
							Author:    github.User{Login: "user2"},
							CreatedAt: commentTime1,
						},
					},
					{
						Comment: github.Comment{
							Body:      "I agree",
							Author:    github.User{Login: "user3"},
							CreatedAt: commentTime2,
						},
					},
				},
			},
			opts: ConvertOptions{},
			want: `---
title: "Feature request"
url: "https://github.com/owner/repo/discussions/790"
type: "discussion"
author: "user1"
created_at: "2026-05-19T08:00:00Z"
updated_at: "2026-05-20T12:00:00Z"
state: "open"
labels: []
comments_count: 2
---

# Feature request

**Author:** @user1
**Created:** 2026-05-19 08:00:00 UTC
**Status:** Open
**Category:** Ideas

---

## Question

Please add feature X

---

## Comments

### Comment by @user2 on 2026-05-19 09:30:00 UTC

Good idea!

---

### Comment by @user3 on 2026-05-19 10:00:00 UTC

I agree

---
`,
		},
		{
			name: "discussion with reactions disabled",
			discussion: &github.Discussion{
				Number:    791,
				Title:     "Test discussion",
				Body:      "Test body",
				HTMLURL:   "https://github.com/owner/repo/discussions/791",
				Author:    github.User{Login: "user1"},
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Category:  "General",
				Comments: []github.DiscussionComment{
					{
						Comment: github.Comment{
							Body:      "Comment with reactions",
							Author:    github.User{Login: "user2"},
							CreatedAt: commentTime1,
							Reactions: github.Reaction{Heart: 5},
						},
					},
				},
			},
			opts: ConvertOptions{
				EnableReactions: false,
			},
			want: `---
title: "Test discussion"
url: "https://github.com/owner/repo/discussions/791"
type: "discussion"
author: "user1"
created_at: "2026-05-19T08:00:00Z"
updated_at: "2026-05-20T12:00:00Z"
state: "open"
labels: []
comments_count: 1
---

# Test discussion

**Author:** @user1
**Created:** 2026-05-19 08:00:00 UTC
**Status:** Open
**Category:** General

---

## Question

Test body

---

## Comments

### Comment by @user2 on 2026-05-19 09:30:00 UTC

Comment with reactions

---
`,
		},
		{
			name: "discussion with user links enabled",
			discussion: &github.Discussion{
				Number:    792,
				Title:     "Another discussion",
				Body:      "Body text",
				HTMLURL:   "https://github.com/owner/repo/discussions/792",
				Author:    github.User{Login: "author1"},
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Category:  "Q&A",
				Comments: []github.DiscussionComment{
					{
						Comment: github.Comment{
							Body:      "Reply",
							Author:    github.User{Login: "commenter1"},
							CreatedAt: commentTime1,
						},
					},
				},
			},
			opts: ConvertOptions{
				EnableUserLinks: true,
			},
			want: `---
title: "Another discussion"
url: "https://github.com/owner/repo/discussions/792"
type: "discussion"
author: "author1"
created_at: "2026-05-19T08:00:00Z"
updated_at: "2026-05-20T12:00:00Z"
state: "open"
labels: []
comments_count: 1
---

# Another discussion

**Author:** [@author1](https://github.com/author1)
**Created:** 2026-05-19 08:00:00 UTC
**Status:** Open
**Category:** Q&A

---

## Question

Body text

---

## Comments

### Comment by [@commenter1](https://github.com/commenter1) on 2026-05-19 09:30:00 UTC

Reply

---
`,
		},
		{
			name: "discussion with user links disabled",
			discussion: &github.Discussion{
				Number:    793,
				Title:     "No links discussion",
				Body:      "Body text",
				HTMLURL:   "https://github.com/owner/repo/discussions/793",
				Author:    github.User{Login: "author1"},
				CreatedAt: baseTime,
				UpdatedAt: updateTime,
				Category:  "Q&A",
				Comments: []github.DiscussionComment{
					{
						Comment: github.Comment{
							Body:      "Reply",
							Author:    github.User{Login: "commenter1"},
							CreatedAt: commentTime1,
						},
					},
				},
			},
			opts: ConvertOptions{
				EnableUserLinks: false,
			},
			want: `---
title: "No links discussion"
url: "https://github.com/owner/repo/discussions/793"
type: "discussion"
author: "author1"
created_at: "2026-05-19T08:00:00Z"
updated_at: "2026-05-20T12:00:00Z"
state: "open"
labels: []
comments_count: 1
---

# No links discussion

**Author:** @author1
**Created:** 2026-05-19 08:00:00 UTC
**Status:** Open
**Category:** Q&A

---

## Question

Body text

---

## Comments

### Comment by @commenter1 on 2026-05-19 09:30:00 UTC

Reply

---
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertDiscussion(tt.discussion, tt.opts)
			if got != tt.want {
				t.Errorf("ConvertDiscussion() mismatch\nGot:\n%s\nWant:\n%s", got, tt.want)
			}
		})
	}
}
