package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// graphqlResponse wraps a GraphQL response for the test server.
type graphqlResponse struct {
	Data json.RawMessage `json:"data"`
}

func TestFetchDiscussion(t *testing.T) {
	createdAt := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	commentTime1 := "2026-05-19T09:30:00Z"
	commentTime2 := "2026-05-19T10:00:00Z"

	t.Run("basic discussion with answer", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("POST /graphql", func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"discussion": map[string]interface{}{
							"number":    789,
							"title":     "How to configure DB?",
							"body":      "I need help configuring the database.",
							"url":       "https://github.com/owner/repo/discussions/789",
							"createdAt": createdAt.Format(time.RFC3339),
							"updatedAt": updatedAt.Format(time.RFC3339),
							"author":    map[string]interface{}{"login": "newuser", "url": "https://github.com/newuser"},
							"category":  map[string]interface{}{"name": "Q&A"},
							"reactionGroups": []map[string]interface{}{
								{"content": "THUMBS_UP", "reactors": map[string]interface{}{"totalCount": 2}},
								{"content": "HEART", "reactors": map[string]interface{}{"totalCount": 1}},
							},
							"comments": map[string]interface{}{
								"nodes": []interface{}{
									map[string]interface{}{
										"author":    map[string]interface{}{"login": "expert", "url": "https://github.com/expert"},
										"body":      "Set DB_HOST=localhost",
										"createdAt": commentTime1,
										"isAnswer":  true,
										"reactionGroups": []map[string]interface{}{
											{"content": "THUMBS_UP", "reactors": map[string]interface{}{"totalCount": 1}},
										},
										"replies": map[string]interface{}{
											"nodes": []interface{}{
												map[string]interface{}{
													"author":         map[string]interface{}{"login": "newuser", "url": "https://github.com/newuser"},
													"body":           "That worked!",
													"createdAt":      "2026-05-19T09:45:00Z",
													"reactionGroups": []interface{}{},
												},
											},
										},
									},
									map[string]interface{}{
										"author":    map[string]interface{}{"login": "newuser", "url": "https://github.com/newuser"},
										"body":      "Thanks!",
										"createdAt": commentTime2,
										"isAnswer":  false,
										"reactionGroups": []map[string]interface{}{
											{"content": "HOORAY", "reactors": map[string]interface{}{"totalCount": 1}},
											{"content": "HEART", "reactors": map[string]interface{}{"totalCount": 1}},
										},
										"replies": map[string]interface{}{
											"nodes": []interface{}{},
										},
									},
								},
								"pageInfo": map[string]interface{}{
									"hasNextPage": false,
									"endCursor":   "",
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestGraphQLClient(server.URL)
		disc, err := client.FetchDiscussion("owner", "repo", 789)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if disc.Title != "How to configure DB?" {
			t.Errorf("Title = %q, want %q", disc.Title, "How to configure DB?")
		}
		if disc.Author.Login != "newuser" {
			t.Errorf("Author.Login = %q, want %q", disc.Author.Login, "newuser")
		}
		if disc.Category != "Q&A" {
			t.Errorf("Category = %q, want %q", disc.Category, "Q&A")
		}
		if disc.Reactions.PlusOne != 2 {
			t.Errorf("Reactions.PlusOne = %d, want 2", disc.Reactions.PlusOne)
		}
		if disc.Reactions.Heart != 1 {
			t.Errorf("Reactions.Heart = %d, want 1", disc.Reactions.Heart)
		}

		if len(disc.Comments) != 3 {
			t.Fatalf("Comments count = %d, want 3 (2 top-level + 1 reply)", len(disc.Comments))
		}

		if !disc.Comments[0].IsAnswer {
			t.Error("Comments[0].IsAnswer = false, want true")
		}
		if disc.Comments[0].Body != "Set DB_HOST=localhost" {
			t.Errorf("Comments[0].Body = %q, want %q", disc.Comments[0].Body, "Set DB_HOST=localhost")
		}

		// Reply is flattened after its parent
		if disc.Comments[1].Body != "That worked!" {
			t.Errorf("Comments[1].Body = %q, want %q (reply)", disc.Comments[1].Body, "That worked!")
		}
		if disc.Comments[1].IsAnswer {
			t.Error("Comments[1] (reply).IsAnswer = true, want false")
		}

		if disc.Comments[2].IsAnswer {
			t.Error("Comments[2].IsAnswer = true, want false")
		}
		if disc.Comments[2].Reactions.Hooray != 1 {
			t.Errorf("Comments[2].Reactions.Hooray = %d, want 1", disc.Comments[2].Reactions.Hooray)
		}
		if disc.Comments[2].Reactions.Heart != 1 {
			t.Errorf("Comments[2].Reactions.Heart = %d, want 1", disc.Comments[2].Reactions.Heart)
		}
	})

	t.Run("discussion with comment pagination", func(t *testing.T) {
		callCount := 0
		mux := http.NewServeMux()
		mux.HandleFunc("POST /graphql", func(w http.ResponseWriter, r *http.Request) {
			callCount++

			var comments interface{}
			var pageInfo interface{}

			if callCount == 1 {
				comments = []interface{}{
					map[string]interface{}{
						"author":    map[string]interface{}{"login": "user1", "url": "https://github.com/user1"},
						"body":      "Page 1 comment",
						"createdAt": commentTime1,
						"isAnswer":  false,
						"reactionGroups": []interface{}{},
						"replies": map[string]interface{}{"nodes": []interface{}{}},
					},
				}
				pageInfo = map[string]interface{}{"hasNextPage": true, "endCursor": "cursor1"}
			} else {
				comments = []interface{}{
					map[string]interface{}{
						"author":         map[string]interface{}{"login": "user2", "url": "https://github.com/user2"},
						"body":           "Page 2 comment",
						"createdAt":      commentTime2,
						"isAnswer":       false,
						"reactionGroups": []interface{}{},
						"replies": map[string]interface{}{"nodes": []interface{}{}},
					},
				}
				pageInfo = map[string]interface{}{"hasNextPage": false, "endCursor": ""}
			}

			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"discussion": map[string]interface{}{
							"number":    100,
							"title":     "Paginated Discussion",
							"body":      "Body",
							"url":       "https://github.com/owner/repo/discussions/100",
							"createdAt": createdAt.Format(time.RFC3339),
							"updatedAt": updatedAt.Format(time.RFC3339),
							"author":    map[string]interface{}{"login": "user", "url": "https://github.com/user"},
							"category":  map[string]interface{}{"name": "General"},
							"reactionGroups": []interface{}{},
							"comments": map[string]interface{}{
								"nodes":    comments,
								"pageInfo": pageInfo,
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestGraphQLClient(server.URL)
		disc, err := client.FetchDiscussion("owner", "repo", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(disc.Comments) != 2 {
			t.Fatalf("Comments count = %d, want 2", len(disc.Comments))
		}
		if disc.Comments[0].Body != "Page 1 comment" {
			t.Errorf("Comments[0].Body = %q, want %q", disc.Comments[0].Body, "Page 1 comment")
		}
		if disc.Comments[1].Body != "Page 2 comment" {
			t.Errorf("Comments[1].Body = %q, want %q", disc.Comments[1].Body, "Page 2 comment")
		}
	})

	t.Run("discussion not found", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("POST /graphql", func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"discussion": nil,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestGraphQLClient(server.URL)
		_, err := client.FetchDiscussion("owner", "repo", 999)
		if err == nil {
			t.Fatal("expected error for missing discussion, got nil")
		}
	})
}

// newTestGraphQLClient creates a Client whose GraphQL API points at the test server.
func newTestGraphQLClient(serverURL string) *Client {
	httpClient := &http.Client{}
	return &Client{
		graphql: newGraphQLClientWithURL(httpClient, serverURL+"/graphql"),
	}
}
