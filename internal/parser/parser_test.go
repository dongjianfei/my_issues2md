package parser

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name        string
		rawURL      string
		want        *ParsedURL
		wantErr     bool
	}{
		{
			name:   "valid issue URL",
			rawURL: "https://github.com/golang/go/issues/123",
			want: &ParsedURL{
				Owner:       "golang",
				Repo:        "go",
				Number:      123,
				ContentType: TypeIssue,
				RawURL:      "https://github.com/golang/go/issues/123",
			},
		},
		{
			name:   "valid PR URL",
			rawURL: "https://github.com/owner/repo/pull/456",
			want: &ParsedURL{
				Owner:       "owner",
				Repo:        "repo",
				Number:      456,
				ContentType: TypePR,
				RawURL:      "https://github.com/owner/repo/pull/456",
			},
		},
		{
			name:   "valid discussion URL",
			rawURL: "https://github.com/owner/repo/discussions/789",
			want: &ParsedURL{
				Owner:       "owner",
				Repo:        "repo",
				Number:      789,
				ContentType: TypeDiscussion,
				RawURL:      "https://github.com/owner/repo/discussions/789",
			},
		},
		{
			name:   "URL with trailing slash",
			rawURL: "https://github.com/owner/repo/issues/42/",
			want: &ParsedURL{
				Owner:       "owner",
				Repo:        "repo",
				Number:      42,
				ContentType: TypeIssue,
				RawURL:      "https://github.com/owner/repo/issues/42/",
			},
		},
		{
			name:   "URL with anchor",
			rawURL: "https://github.com/owner/repo/issues/10#issuecomment-123456",
			want: &ParsedURL{
				Owner:       "owner",
				Repo:        "repo",
				Number:      10,
				ContentType: TypeIssue,
				RawURL:      "https://github.com/owner/repo/issues/10#issuecomment-123456",
			},
		},
		{
			name:   "URL with query params",
			rawURL: "https://github.com/owner/repo/pull/99?diff=unified",
			want: &ParsedURL{
				Owner:       "owner",
				Repo:        "repo",
				Number:      99,
				ContentType: TypePR,
				RawURL:      "https://github.com/owner/repo/pull/99?diff=unified",
			},
		},
		{
			name:    "empty string",
			rawURL:  "",
			wantErr: true,
		},
		{
			name:    "non-GitHub URL",
			rawURL:  "https://gitlab.com/owner/repo/issues/1",
			wantErr: true,
		},
		{
			name:    "missing number",
			rawURL:  "https://github.com/owner/repo/issues/",
			wantErr: true,
		},
		{
			name:    "number is not numeric",
			rawURL:  "https://github.com/owner/repo/issues/abc",
			wantErr: true,
		},
		{
			name:    "unsupported path type",
			rawURL:  "https://github.com/owner/repo/wiki/Home",
			wantErr: true,
		},
		{
			name:    "too few path segments",
			rawURL:  "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "extra path segments after number",
			rawURL:  "https://github.com/owner/repo/issues/1/extra",
			wantErr: true,
		},
		{
			name:    "number is zero",
			rawURL:  "https://github.com/owner/repo/issues/0",
			wantErr: true,
		},
		{
			name:    "number is negative",
			rawURL:  "https://github.com/owner/repo/issues/-1",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			rawURL:  "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseURL(%q) expected error, got nil", tt.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseURL(%q) unexpected error: %v", tt.rawURL, err)
			}
			if got.Owner != tt.want.Owner {
				t.Errorf("Owner = %q, want %q", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.want.Repo)
			}
			if got.Number != tt.want.Number {
				t.Errorf("Number = %d, want %d", got.Number, tt.want.Number)
			}
			if got.ContentType != tt.want.ContentType {
				t.Errorf("ContentType = %q, want %q", got.ContentType, tt.want.ContentType)
			}
			if got.RawURL != tt.want.RawURL {
				t.Errorf("RawURL = %q, want %q", got.RawURL, tt.want.RawURL)
			}
		})
	}
}
