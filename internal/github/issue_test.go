package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v60/github"
)

// newTestClient creates a Client whose REST API points at the given test server.
func newTestClient(serverURL string) *Client {
	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(serverURL + "/")
	return &Client{rest: client}
}

func TestFetchIssue(t *testing.T) {
	createdAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	commentTime1 := time.Date(2026, 5, 16, 8, 0, 0, 0, time.UTC)
	commentTime2 := time.Date(2026, 5, 17, 9, 0, 0, 0, time.UTC)

	t.Run("basic issue with labels and state", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
			issue := gogithub.Issue{
				Number:  intPtr(1),
				Title:   strPtr("Test Issue"),
				Body:    strPtr("Issue body content"),
				State:   strPtr("open"),
				HTMLURL: strPtr("https://github.com/owner/repo/issues/1"),
				User:    &gogithub.User{Login: strPtr("johndoe"), HTMLURL: strPtr("https://github.com/johndoe")},
				Labels: []*gogithub.Label{
					{Name: strPtr("bug")},
					{Name: strPtr("help wanted")},
				},
				CreatedAt: &gogithub.Timestamp{Time: createdAt},
				UpdatedAt: &gogithub.Timestamp{Time: updatedAt},
				Reactions: &gogithub.Reactions{
					PlusOne: intPtr(5),
					Heart:   intPtr(2),
				},
				Comments: intPtr(0),
			}
			writeJSON(w, issue)
		})
		mux.HandleFunc("GET /repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []*gogithub.IssueComment{})
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		issue, err := client.FetchIssue("owner", "repo", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if issue.Title != "Test Issue" {
			t.Errorf("Title = %q, want %q", issue.Title, "Test Issue")
		}
		if issue.Author.Login != "johndoe" {
			t.Errorf("Author.Login = %q, want %q", issue.Author.Login, "johndoe")
		}
		if issue.State != "open" {
			t.Errorf("State = %q, want %q", issue.State, "open")
		}
		if len(issue.Labels) != 2 {
			t.Errorf("Labels count = %d, want 2", len(issue.Labels))
		}
		if issue.Reactions.PlusOne != 5 {
			t.Errorf("Reactions.PlusOne = %d, want 5", issue.Reactions.PlusOne)
		}
		if issue.Reactions.Heart != 2 {
			t.Errorf("Reactions.Heart = %d, want 2", issue.Reactions.Heart)
		}
		if issue.CommentCount != 0 {
			t.Errorf("CommentCount = %d, want 0", issue.CommentCount)
		}
	})

	t.Run("issue with comments and pagination", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /repos/owner/repo/issues/2", func(w http.ResponseWriter, r *http.Request) {
			issue := gogithub.Issue{
				Number:    intPtr(2),
				Title:     strPtr("Paginated Issue"),
				Body:      strPtr("Body"),
				State:     strPtr("closed"),
				HTMLURL:   strPtr("https://github.com/owner/repo/issues/2"),
				User:      &gogithub.User{Login: strPtr("alice"), HTMLURL: strPtr("https://github.com/alice")},
				CreatedAt: &gogithub.Timestamp{Time: createdAt},
				UpdatedAt: &gogithub.Timestamp{Time: updatedAt},
				Reactions: &gogithub.Reactions{},
				Comments:  intPtr(2),
			}
			writeJSON(w, issue)
		})
		callCount := 0
		mux.HandleFunc("GET /repos/owner/repo/issues/2/comments", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				// Page 1: one comment, link to page 2
				w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/owner/repo/issues/2/comments?page=2>; rel="next"`, r.Host))
				comments := []*gogithub.IssueComment{
					{
						User:      &gogithub.User{Login: strPtr("bob"), HTMLURL: strPtr("https://github.com/bob")},
						Body:      strPtr("First comment"),
						CreatedAt: &gogithub.Timestamp{Time: commentTime1},
						Reactions: &gogithub.Reactions{PlusOne: intPtr(1)},
					},
				}
				writeJSON(w, comments)
			} else {
				// Page 2: one comment, no next link
				comments := []*gogithub.IssueComment{
					{
						User:      &gogithub.User{Login: strPtr("carol"), HTMLURL: strPtr("https://github.com/carol")},
						Body:      strPtr("Second comment"),
						CreatedAt: &gogithub.Timestamp{Time: commentTime2},
						Reactions: &gogithub.Reactions{Heart: intPtr(3)},
					},
				}
				writeJSON(w, comments)
			}
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		issue, err := client.FetchIssue("owner", "repo", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(issue.Comments) != 2 {
			t.Fatalf("Comments count = %d, want 2", len(issue.Comments))
		}
		if issue.Comments[0].Body != "First comment" {
			t.Errorf("Comments[0].Body = %q, want %q", issue.Comments[0].Body, "First comment")
		}
		if issue.Comments[1].Body != "Second comment" {
			t.Errorf("Comments[1].Body = %q, want %q", issue.Comments[1].Body, "Second comment")
		}
		if issue.Comments[0].Reactions.PlusOne != 1 {
			t.Errorf("Comments[0].Reactions.PlusOne = %d, want 1", issue.Comments[0].Reactions.PlusOne)
		}
		if issue.Comments[1].Reactions.Heart != 3 {
			t.Errorf("Comments[1].Reactions.Heart = %d, want 3", issue.Comments[1].Reactions.Heart)
		}
	})

	t.Run("issue not found 404", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /repos/owner/repo/issues/999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, gogithub.ErrorResponse{Message: "Not Found"})
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		_, err := client.FetchIssue("owner", "repo", 999)
		if err == nil {
			t.Fatal("expected error for 404, got nil")
		}
	})

	t.Run("forbidden 403", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			writeJSON(w, gogithub.ErrorResponse{Message: "Forbidden"})
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestClient(server.URL)
		_, err := client.FetchIssue("owner", "repo", 1)
		if err == nil {
			t.Fatal("expected error for 403, got nil")
		}
	})
}

// Helper functions

func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
