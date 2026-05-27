package github

import (
	"context"
	"fmt"
	"sort"

	gogithub "github.com/google/go-github/v60/github"
)

// FetchPullRequest 获取PR完整数据（含普通评论+Review评论，已按时间排序）
func (c *Client) FetchPullRequest(owner, repo string, number int) (*PullRequest, error) {
	ctx := context.Background()

	ghPR, _, err := c.rest.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, wrapAPIError(err, fmt.Sprintf("pull request %s/%s#%d", owner, repo, number))
	}

	state := ghPR.GetState()
	if ghPR.GetMerged() {
		state = "merged"
	}

	pr := &PullRequest{
		Number:    number,
		Title:     ghPR.GetTitle(),
		Author:    userFromGH(ghPR.GetUser()),
		Body:      ghPR.GetBody(),
		State:     state,
		HTMLURL:   ghPR.GetHTMLURL(),
		CreatedAt: ghPR.GetCreatedAt().Time,
		UpdatedAt: ghPR.GetUpdatedAt().Time,
	}

	for _, l := range ghPR.Labels {
		pr.Labels = append(pr.Labels, Label{Name: l.GetName()})
	}

	// Fetch reactions from the Issues API (PRs are issues)
	ghIssue, _, err := c.rest.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, wrapAPIError(err, fmt.Sprintf("pull request reactions %s/%s#%d", owner, repo, number))
	}
	pr.Reactions = reactionsFromGH(ghIssue.GetReactions())

	// Fetch issue comments (普通评论)
	issueComments, err := c.fetchIssueComments(ctx, owner, repo, number)
	if err != nil {
		return nil, wrapAPIError(err, fmt.Sprintf("pull request comments %s/%s#%d", owner, repo, number))
	}

	// Fetch review comments (代码审查评论)
	reviewComments, err := c.fetchReviewComments(ctx, owner, repo, number)
	if err != nil {
		return nil, wrapAPIError(err, fmt.Sprintf("pull request review comments %s/%s#%d", owner, repo, number))
	}

	// Merge and sort by time
	allComments := append(issueComments, reviewComments...)
	sort.Slice(allComments, func(i, j int) bool {
		return allComments[i].CreatedAt.Before(allComments[j].CreatedAt)
	})

	pr.Comments = allComments
	pr.CommentCount = len(allComments)

	return pr, nil
}

func (c *Client) fetchReviewComments(ctx context.Context, owner, repo string, number int) ([]Comment, error) {
	var allComments []Comment

	opts := &gogithub.PullRequestListCommentsOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}

	for {
		ghComments, resp, err := c.rest.PullRequests.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("list review comments: %w", err)
		}

		for _, gc := range ghComments {
			allComments = append(allComments, Comment{
				Author:    userFromGH(gc.GetUser()),
				Body:      gc.GetBody(),
				CreatedAt: gc.GetCreatedAt().Time,
				IsReview:  true,
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
