package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
)

// GraphQL query structs for Discussion

type discussionQuery struct {
	Repository struct {
		Discussion *discussionNode `graphql:"discussion(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

type discussionNode struct {
	Number    int
	Title     string
	Body      string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
	Author    struct {
		Login string
		URL   string
	}
	Category struct {
		Name string
	}
	ReactionGroups []reactionGroup
	Comments       struct {
		Nodes    []discussionCommentNode
		PageInfo struct {
			HasNextPage bool
			EndCursor   string
		}
	} `graphql:"comments(first: 100, after: $commentsCursor)"`
}

type reactionGroup struct {
	Content string
	Reactors struct {
		TotalCount int
	}
}

type discussionCommentNode struct {
	Author struct {
		Login string
		URL   string
	}
	Body           string
	CreatedAt      time.Time
	IsAnswer       bool
	ReactionGroups []reactionGroup
	Replies        struct {
		Nodes []discussionReplyNode
	} `graphql:"replies(first: 100)"`
}

type discussionReplyNode struct {
	Author struct {
		Login string
		URL   string
	}
	Body           string
	CreatedAt      time.Time
	ReactionGroups []reactionGroup
}

// FetchDiscussion 获取Discussion完整数据（含Answer标记）
func (c *Client) FetchDiscussion(owner, repo string, number int) (*Discussion, error) {
	ctx := context.Background()

	var q discussionQuery
	variables := map[string]interface{}{
		"owner":          githubv4.String(owner),
		"repo":           githubv4.String(repo),
		"number":         githubv4.Int(number),
		"commentsCursor": (*githubv4.String)(nil),
	}

	err := c.graphql.Query(ctx, &q, variables)
	if err != nil {
		return nil, fmt.Errorf("discussion %s/%s#%d: %w", owner, repo, number, err)
	}

	if q.Repository.Discussion == nil {
		return nil, fmt.Errorf("discussion %s/%s#%d not found (404). Please check the URL and ensure the resource exists", owner, repo, number)
	}

	d := q.Repository.Discussion

	disc := &Discussion{
		Number:    d.Number,
		Title:     d.Title,
		Author:    User{Login: d.Author.Login, HTMLURL: d.Author.URL},
		Body:      d.Body,
		Category:  d.Category.Name,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		HTMLURL:   d.URL,
		Reactions: reactionsFromGroups(d.ReactionGroups),
	}

	// Collect first page of comments (with replies flattened)
	for _, c := range d.Comments.Nodes {
		disc.Comments = append(disc.Comments, toDiscussionComment(c))
		for _, r := range c.Replies.Nodes {
			disc.Comments = append(disc.Comments, toDiscussionReply(r))
		}
	}

	// Paginate remaining comments
	for d.Comments.PageInfo.HasNextPage {
		cursor := githubv4.String(d.Comments.PageInfo.EndCursor)
		variables["commentsCursor"] = &cursor

		var nextQ discussionQuery
		err := c.graphql.Query(ctx, &nextQ, variables)
		if err != nil {
			return nil, fmt.Errorf("fetch discussion comments page %s/%s#%d: %w", owner, repo, number, err)
		}

		if nextQ.Repository.Discussion == nil {
			break
		}

		nd := nextQ.Repository.Discussion
		for _, c := range nd.Comments.Nodes {
			disc.Comments = append(disc.Comments, toDiscussionComment(c))
			for _, r := range c.Replies.Nodes {
				disc.Comments = append(disc.Comments, toDiscussionReply(r))
			}
		}
		d.Comments.PageInfo = nd.Comments.PageInfo
	}

	disc.CommentCount = len(disc.Comments)
	return disc, nil
}

func toDiscussionComment(n discussionCommentNode) DiscussionComment {
	return DiscussionComment{
		Comment: Comment{
			Author:    User{Login: n.Author.Login, HTMLURL: n.Author.URL},
			Body:      n.Body,
			CreatedAt: n.CreatedAt,
			Reactions: reactionsFromGroups(n.ReactionGroups),
		},
		IsAnswer: n.IsAnswer,
	}
}

func toDiscussionReply(n discussionReplyNode) DiscussionComment {
	return DiscussionComment{
		Comment: Comment{
			Author:    User{Login: n.Author.Login, HTMLURL: n.Author.URL},
			Body:      n.Body,
			CreatedAt: n.CreatedAt,
			Reactions: reactionsFromGroups(n.ReactionGroups),
		},
		IsAnswer: false,
	}
}

func reactionsFromGroups(groups []reactionGroup) Reaction {
	var r Reaction
	for _, g := range groups {
		switch g.Content {
		case "THUMBS_UP":
			r.PlusOne = g.Reactors.TotalCount
		case "THUMBS_DOWN":
			r.MinusOne = g.Reactors.TotalCount
		case "LAUGH":
			r.Laugh = g.Reactors.TotalCount
		case "CONFUSED":
			r.Confused = g.Reactors.TotalCount
		case "HEART":
			r.Heart = g.Reactors.TotalCount
		case "HOORAY":
			r.Hooray = g.Reactors.TotalCount
		case "ROCKET":
			r.Rocket = g.Reactors.TotalCount
		case "EYES":
			r.Eyes = g.Reactors.TotalCount
		}
	}
	return r
}
