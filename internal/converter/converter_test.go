package converter

import (
	"testing"
	"time"

	"github.com/dongjianfei/issue2md/internal/github"
)

func TestFormatReactions(t *testing.T) {
	tests := []struct {
		name     string
		reaction github.Reaction
		want     string
	}{
		{
			name:     "all zero reactions",
			reaction: github.Reaction{},
			want:     "",
		},
		{
			name: "single reaction",
			reaction: github.Reaction{
				PlusOne: 5,
			},
			want: "**Reactions:** 👍 5",
		},
		{
			name: "multiple reactions",
			reaction: github.Reaction{
				PlusOne: 5,
				Heart:   3,
			},
			want: "**Reactions:** 👍 5 | ❤️ 3",
		},
		{
			name: "all reactions present",
			reaction: github.Reaction{
				PlusOne:  10,
				MinusOne: 2,
				Laugh:    5,
				Confused: 1,
				Heart:    8,
				Hooray:   3,
				Rocket:   6,
				Eyes:     4,
			},
			want: "**Reactions:** 👍 10 | 👎 2 | 😄 5 | 😕 1 | ❤️ 8 | 🎉 3 | 🚀 6 | 👀 4",
		},
		{
			name: "sparse reactions",
			reaction: github.Reaction{
				Laugh:  2,
				Rocket: 1,
			},
			want: "**Reactions:** 😄 2 | 🚀 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatReactions(tt.reaction)
			if got != tt.want {
				t.Errorf("formatReactions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatUser(t *testing.T) {
	tests := []struct {
		name        string
		user        github.User
		enableLinks bool
		want        string
	}{
		{
			name: "without links",
			user: github.User{
				Login: "octocat",
			},
			enableLinks: false,
			want:        "@octocat",
		},
		{
			name: "with links",
			user: github.User{
				Login: "octocat",
			},
			enableLinks: true,
			want:        "[@octocat](https://github.com/octocat)",
		},
		{
			name: "empty username without links",
			user: github.User{
				Login: "",
			},
			enableLinks: false,
			want:        "@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUser(tt.user, tt.enableLinks)
			if got != tt.want {
				t.Errorf("formatUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "specific time",
			time: time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC),
			want: "2026-05-20 10:30:00 UTC",
		},
		{
			name: "zero time",
			time: time.Time{},
			want: "0001-01-01 00:00:00 UTC",
		},
		{
			name: "different timezone converted to UTC",
			time: time.Date(2026, 5, 20, 18, 30, 0, 0, time.FixedZone("CST", 8*3600)),
			want: "2026-05-20 10:30:00 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.time)
			if got != tt.want {
				t.Errorf("formatTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels []github.Label
		want   string
	}{
		{
			name:   "empty labels",
			labels: []github.Label{},
			want:   "",
		},
		{
			name:   "nil labels",
			labels: nil,
			want:   "",
		},
		{
			name: "single label",
			labels: []github.Label{
				{Name: "bug"},
			},
			want: "`bug`",
		},
		{
			name: "multiple labels",
			labels: []github.Label{
				{Name: "bug"},
				{Name: "enhancement"},
			},
			want: "`bug`, `enhancement`",
		},
		{
			name: "three labels",
			labels: []github.Label{
				{Name: "bug"},
				{Name: "enhancement"},
				{Name: "documentation"},
			},
			want: "`bug`, `enhancement`, `documentation`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLabels(tt.labels)
			if got != tt.want {
				t.Errorf("formatLabels() = %q, want %q", got, tt.want)
			}
		})
	}
}

