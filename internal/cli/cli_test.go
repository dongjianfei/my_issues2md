package cli

import (
	"bytes"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *RunOptions
		wantErr bool
	}{
		{
			name: "URL only",
			args: []string{"https://github.com/o/r/issues/1"},
			want: &RunOptions{
				URL: "https://github.com/o/r/issues/1",
			},
		},
		{
			name: "URL with output file",
			args: []string{"https://github.com/o/r/issues/1", "out.md"},
			want: &RunOptions{
				URL:        "https://github.com/o/r/issues/1",
				OutputFile: "out.md",
			},
		},
		{
			name: "enable-reactions flag",
			args: []string{"-enable-reactions", "https://github.com/o/r/issues/1"},
			want: &RunOptions{
				URL:             "https://github.com/o/r/issues/1",
				EnableReactions: true,
			},
		},
		{
			name: "enable-user-links flag",
			args: []string{"-enable-user-links", "https://github.com/o/r/issues/1"},
			want: &RunOptions{
				URL:             "https://github.com/o/r/issues/1",
				EnableUserLinks: true,
			},
		},
		{
			name: "all flags with output file",
			args: []string{"-enable-reactions", "-enable-user-links", "https://github.com/o/r/pull/99", "pr.md"},
			want: &RunOptions{
				URL:             "https://github.com/o/r/pull/99",
				OutputFile:      "pr.md",
				EnableReactions: true,
				EnableUserLinks: true,
			},
		},
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "too many positional arguments",
			args:    []string{"https://github.com/o/r/issues/1", "out.md", "extra"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"-unknown", "https://github.com/o/r/issues/1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseArgs(%v) expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseArgs(%v) unexpected error: %v", tt.args, err)
			}
			if got.URL != tt.want.URL {
				t.Errorf("URL = %q, want %q", got.URL, tt.want.URL)
			}
			if got.OutputFile != tt.want.OutputFile {
				t.Errorf("OutputFile = %q, want %q", got.OutputFile, tt.want.OutputFile)
			}
			if got.EnableReactions != tt.want.EnableReactions {
				t.Errorf("EnableReactions = %v, want %v", got.EnableReactions, tt.want.EnableReactions)
			}
			if got.EnableUserLinks != tt.want.EnableUserLinks {
				t.Errorf("EnableUserLinks = %v, want %v", got.EnableUserLinks, tt.want.EnableUserLinks)
			}
		})
	}
}

func TestRun(t *testing.T) {
	t.Run("invalid URL returns error", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &RunOptions{URL: "https://not-github.com/foo"}
		err := Run(&buf, opts)
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("empty URL returns error", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &RunOptions{URL: ""}
		err := Run(&buf, opts)
		if err == nil {
			t.Fatal("expected error for empty URL, got nil")
		}
	})

	t.Run("unsupported URL type returns error", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &RunOptions{URL: "https://github.com/owner/repo/wiki/Home"}
		err := Run(&buf, opts)
		if err == nil {
			t.Fatal("expected error for unsupported URL type, got nil")
		}
	})
}
