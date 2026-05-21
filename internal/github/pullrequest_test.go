package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v60/github"
)

func TestFetchPullRequest(t *testing.T) {
	createdAt := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 20, 16, 0, 0, 0, time.UTC)
	commentTime1 := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	reviewTime1 := time.Date(2026, 5, 18, 10, 30, 0, 0, time.UTC)
	commentTime2 := time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC)

	t.Run("basic merged PR", func(t *testing.T) {
		mux := http.NewServeMux()
		merged := true
		mux.HandleFunc("GET /repos/owner/repo/pulls/10", func(w http.ResponseWriter, r *http.Request) {
			pr := gogithub.PullRequest{
				Number:  intPtr(10),
				Title:   strPtr("feat: Add auth"),
				Body:    strPtr("PR body"),
				State:   strPtr("closed"),
				Merged:  &merged,
				HTMLURL: strPtr("https://github.com/owner/repo/pull/10"),
				User:    &gogithub.User{Login: strPtr("dev"), HTMLURL: strPtr("https://github.com/dev")},
				Labels: []*gogithub.Label{
					{Name: strPtr("enhancement")},
				},
				CreatedAt: &gogithub.Timestamp{Time: createdAt},
				UpdatedAt: &gogithub.Timestamp{Time: updatedAt},
			}
			writeJSON(w, pr)
		})
		// PR reactions are fetched via Issues API
		mux.HandleFunc("GET /repos/owner/repo/issues/10", func(w http.ResponseWriter, r *http.Request) {
			issue := gogithub.Issue{
				Number:    intPtr(10),
				Reactions: &gogithub.Reactions{PlusOne: intPtr(3)},
			}
			writeJSON(w, issue)
		})
		mux.HandleFunc("GET /repos/owner/repo/issues/10/comments", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []*gogithub.IssueComment{})
		})
		mux.HandleFunc("GET /repos/owner/repo/pulls/10/comments", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []*gogithub.PullRequestComment{})
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		pr, err := client.FetchPullRequest("owner", "repo", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pr.Title != "feat: Add auth" {
			t.Errorf("Title = %q, want %q", pr.Title, "feat: Add auth")
		}
		if pr.State != "merged" {
			t.Errorf("State = %q, want %q", pr.State, "merged")
		}
		if pr.Reactions.PlusOne != 3 {
			t.Errorf("Reactions.PlusOne = %d, want 3", pr.Reactions.PlusOne)
		}
		if len(pr.Labels) != 1 || pr.Labels[0].Name != "enhancement" {
			t.Errorf("Labels = %v, want [enhancement]", pr.Labels)
		}
	})

	t.Run("PR with mixed comments sorted by time", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /repos/owner/repo/pulls/20", func(w http.ResponseWriter, r *http.Request) {
			pr := gogithub.PullRequest{
				Number:    intPtr(20),
				Title:     strPtr("PR with comments"),
				Body:      strPtr("Body"),
				State:     strPtr("open"),
				HTMLURL:   strPtr("https://github.com/owner/repo/pull/20"),
				User:      &gogithub.User{Login: strPtr("dev"), HTMLURL: strPtr("https://github.com/dev")},
				CreatedAt: &gogithub.Timestamp{Time: createdAt},
				UpdatedAt: &gogithub.Timestamp{Time: updatedAt},
			}
			writeJSON(w, pr)
		})
		mux.HandleFunc("GET /repos/owner/repo/issues/20", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, gogithub.Issue{Number: intPtr(20), Reactions: &gogithub.Reactions{}})
		})
		// Issue comments (普通评论): at 10:00 and 14:00
		mux.HandleFunc("GET /repos/owner/repo/issues/20/comments", func(w http.ResponseWriter, r *http.Request) {
			comments := []*gogithub.IssueComment{
				{
					User:      &gogithub.User{Login: strPtr("alice"), HTMLURL: strPtr("https://github.com/alice")},
					Body:      strPtr("Looks good!"),
					CreatedAt: &gogithub.Timestamp{Time: commentTime1},
					Reactions: &gogithub.Reactions{},
				},
				{
					User:      &gogithub.User{Login: strPtr("dev"), HTMLURL: strPtr("https://github.com/dev")},
					Body:      strPtr("Thanks!"),
					CreatedAt: &gogithub.Timestamp{Time: commentTime2},
					Reactions: &gogithub.Reactions{},
				},
			}
			writeJSON(w, comments)
		})
		// Review comments (代码审查评论): at 10:30 (between the two issue comments)
		mux.HandleFunc("GET /repos/owner/repo/pulls/20/comments", func(w http.ResponseWriter, r *http.Request) {
			comments := []*gogithub.PullRequestComment{
				{
					User:      &gogithub.User{Login: strPtr("reviewer"), HTMLURL: strPtr("https://github.com/reviewer")},
					Body:      strPtr("Add rate limiting here"),
					CreatedAt: &gogithub.Timestamp{Time: reviewTime1},
					Reactions: &gogithub.Reactions{PlusOne: intPtr(2)},
				},
			}
			writeJSON(w, comments)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		pr, err := client.FetchPullRequest("owner", "repo", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(pr.Comments) != 3 {
			t.Fatalf("Comments count = %d, want 3", len(pr.Comments))
		}

		// Verify time-sorted order: 10:00, 10:30, 14:00
		if pr.Comments[0].Body != "Looks good!" {
			t.Errorf("Comments[0].Body = %q, want %q", pr.Comments[0].Body, "Looks good!")
		}
		if pr.Comments[0].IsReview {
			t.Error("Comments[0].IsReview = true, want false")
		}

		if pr.Comments[1].Body != "Add rate limiting here" {
			t.Errorf("Comments[1].Body = %q, want %q", pr.Comments[1].Body, "Add rate limiting here")
		}
		if !pr.Comments[1].IsReview {
			t.Error("Comments[1].IsReview = false, want true")
		}

		if pr.Comments[2].Body != "Thanks!" {
			t.Errorf("Comments[2].Body = %q, want %q", pr.Comments[2].Body, "Thanks!")
		}
		if pr.Comments[2].IsReview {
			t.Error("Comments[2].IsReview = true, want false")
		}
	})

	t.Run("PR with only review comments", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /repos/owner/repo/pulls/30", func(w http.ResponseWriter, r *http.Request) {
			pr := gogithub.PullRequest{
				Number:    intPtr(30),
				Title:     strPtr("Review only"),
				Body:      strPtr("Body"),
				State:     strPtr("open"),
				HTMLURL:   strPtr("https://github.com/owner/repo/pull/30"),
				User:      &gogithub.User{Login: strPtr("dev"), HTMLURL: strPtr("https://github.com/dev")},
				CreatedAt: &gogithub.Timestamp{Time: createdAt},
				UpdatedAt: &gogithub.Timestamp{Time: updatedAt},
			}
			writeJSON(w, pr)
		})
		mux.HandleFunc("GET /repos/owner/repo/issues/30", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, gogithub.Issue{Number: intPtr(30), Reactions: &gogithub.Reactions{}})
		})
		mux.HandleFunc("GET /repos/owner/repo/issues/30/comments", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []*gogithub.IssueComment{})
		})
		mux.HandleFunc("GET /repos/owner/repo/pulls/30/comments", func(w http.ResponseWriter, r *http.Request) {
			comments := []*gogithub.PullRequestComment{
				{
					User:      &gogithub.User{Login: strPtr("reviewer"), HTMLURL: strPtr("https://github.com/reviewer")},
					Body:      strPtr("Review comment"),
					CreatedAt: &gogithub.Timestamp{Time: reviewTime1},
					Reactions: &gogithub.Reactions{},
				},
			}
			writeJSON(w, comments)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		pr, err := client.FetchPullRequest("owner", "repo", 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(pr.Comments) != 1 {
			t.Fatalf("Comments count = %d, want 1", len(pr.Comments))
		}
		if !pr.Comments[0].IsReview {
			t.Error("Comments[0].IsReview = false, want true")
		}
	})

	t.Run("PR not found 404", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /repos/owner/repo/pulls/999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, gogithub.ErrorResponse{Message: "Not Found"})
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		_, err := client.FetchPullRequest("owner", "repo", 999)
		if err == nil {
			t.Fatal("expected error for 404, got nil")
		}
	})
}
