package github

import "time"

// User GitHub用户
type User struct {
	Login   string
	HTMLURL string
}

// Label 标签
type Label struct {
	Name string
}

// Reaction 反应统计
type Reaction struct {
	PlusOne  int // 👍
	MinusOne int // 👎
	Laugh    int // 😄
	Confused int // 😕
	Heart    int // ❤️
	Hooray   int // 🎉
	Rocket   int // 🚀
	Eyes     int // 👀
}

// Comment 评论
type Comment struct {
	Author    User
	Body      string
	CreatedAt time.Time
	IsReview  bool
	Reactions Reaction
}

// DiscussionComment Discussion评论（支持Answer标记）
type DiscussionComment struct {
	Comment
	IsAnswer bool
}

// Issue GitHub Issue完整数据
type Issue struct {
	Number       int
	Title        string
	Author       User
	Body         string
	State        string // "open", "closed"
	Labels       []Label
	CreatedAt    time.Time
	UpdatedAt    time.Time
	HTMLURL      string
	Comments     []Comment
	Reactions    Reaction
	CommentCount int
}

// PullRequest GitHub PR完整数据
type PullRequest struct {
	Number       int
	Title        string
	Author       User
	Body         string
	State        string // "open", "closed", "merged"
	Labels       []Label
	CreatedAt    time.Time
	UpdatedAt    time.Time
	HTMLURL      string
	Comments     []Comment
	Reactions    Reaction
	CommentCount int
}

// Discussion GitHub Discussion完整数据
type Discussion struct {
	Number       int
	Title        string
	Author       User
	Body         string
	Category     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	HTMLURL      string
	Comments     []DiscussionComment
	Reactions    Reaction
	CommentCount int
}
