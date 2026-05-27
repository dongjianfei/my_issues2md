package github

import (
	"context"
	"fmt"

	gogithub "github.com/google/go-github/v60/github"
)

// FetchIssue 获取Issue完整数据（含所有评论）
func (c *Client) FetchIssue(owner, repo string, number int) (*Issue, error) {
	ctx := context.Background()

	ghIssue, _, err := c.rest.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, wrapAPIError(err, fmt.Sprintf("issue %s/%s#%d", owner, repo, number))
	}

	issue := &Issue{
		Number:       number,
		Title:        ghIssue.GetTitle(),
		Author:       userFromGH(ghIssue.GetUser()),
		Body:         ghIssue.GetBody(),
		State:        ghIssue.GetState(),
		HTMLURL:      ghIssue.GetHTMLURL(),
		CreatedAt:    ghIssue.GetCreatedAt().Time,
		UpdatedAt:    ghIssue.GetUpdatedAt().Time,
		Reactions:    reactionsFromGH(ghIssue.GetReactions()),
		CommentCount: ghIssue.GetComments(),
	}

	for _, l := range ghIssue.Labels {
		issue.Labels = append(issue.Labels, Label{Name: l.GetName()})
	}

	comments, err := c.fetchIssueComments(ctx, owner, repo, number)
	if err != nil {
		return nil, wrapAPIError(err, fmt.Sprintf("issue comments %s/%s#%d", owner, repo, number))
	}
	issue.Comments = comments

	return issue, nil
}

func (c *Client) fetchIssueComments(ctx context.Context, owner, repo string, number int) ([]Comment, error) {
	var allComments []Comment

	opts := &gogithub.IssueListCommentsOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}

	for {
		ghComments, resp, err := c.rest.Issues.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("list comments: %w", err)
		}

		for _, gc := range ghComments {
			allComments = append(allComments, Comment{
				Author:    userFromGH(gc.GetUser()),
				Body:      gc.GetBody(),
				CreatedAt: gc.GetCreatedAt().Time,
				IsReview:  false,
				Reactions: reactionsFromGH(gc.GetReactions()),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

func userFromGH(u *gogithub.User) User {
	if u == nil {
		return User{}
	}
	return User{
		Login:   u.GetLogin(),
		HTMLURL: u.GetHTMLURL(),
	}
}

func reactionsFromGH(r *gogithub.Reactions) Reaction {
	if r == nil {
		return Reaction{}
	}
	return Reaction{
		PlusOne:  r.GetPlusOne(),
		MinusOne: r.GetMinusOne(),
		Laugh:    r.GetLaugh(),
		Confused: r.GetConfused(),
		Heart:    r.GetHeart(),
		Hooray:   r.GetHooray(),
		Rocket:   r.GetRocket(),
		Eyes:     r.GetEyes(),
	}
}
