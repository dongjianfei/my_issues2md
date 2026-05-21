package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dongjianfei/issue2md/internal/converter"
	"github.com/dongjianfei/issue2md/internal/github"
	"github.com/dongjianfei/issue2md/internal/parser"
)

// RunOptions 从命令行参数解析出的运行选项
type RunOptions struct {
	URL             string
	OutputFile      string // 空字符串表示输出到stdout
	EnableReactions bool
	EnableUserLinks bool
}

// ParseArgs 解析命令行参数
// args 通常传入 os.Args[1:]
func ParseArgs(args []string) (*RunOptions, error) {
	fs := flag.NewFlagSet("issue2md", flag.ContinueOnError)

	var opts RunOptions
	fs.BoolVar(&opts.EnableReactions, "enable-reactions", false, "include reactions in output")
	fs.BoolVar(&opts.EnableUserLinks, "enable-user-links", false, "render usernames as GitHub profile links")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	positional := fs.Args()
	if len(positional) < 1 {
		return nil, fmt.Errorf("usage: issue2md [flags] <url> [output_file]")
	}
	if len(positional) > 2 {
		return nil, fmt.Errorf("too many arguments: expected at most 2 positional arguments (url, output_file)")
	}

	opts.URL = positional[0]
	if len(positional) == 2 {
		opts.OutputFile = positional[1]
	}

	return &opts, nil
}

// Run 执行主流程：解析URL → 获取数据 → 转换Markdown → 输出
func Run(w io.Writer, opts *RunOptions) error {
	parsed, err := parser.ParseURL(opts.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	client := github.NewClient(token)

	convertOpts := converter.ConvertOptions{
		EnableReactions: opts.EnableReactions,
		EnableUserLinks: opts.EnableUserLinks,
	}

	var markdown string

	switch parsed.ContentType {
	case parser.TypeIssue:
		issue, err := client.FetchIssue(parsed.Owner, parsed.Repo, parsed.Number)
		if err != nil {
			return fmt.Errorf("fetch issue: %w", err)
		}
		markdown = converter.ConvertIssue(issue, convertOpts)

	case parser.TypePR:
		pr, err := client.FetchPullRequest(parsed.Owner, parsed.Repo, parsed.Number)
		if err != nil {
			return fmt.Errorf("fetch pull request: %w", err)
		}
		markdown = converter.ConvertPullRequest(pr, convertOpts)

	case parser.TypeDiscussion:
		disc, err := client.FetchDiscussion(parsed.Owner, parsed.Repo, parsed.Number)
		if err != nil {
			return fmt.Errorf("fetch discussion: %w", err)
		}
		markdown = converter.ConvertDiscussion(disc, convertOpts)
	}

	_, err = fmt.Fprint(w, markdown)
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
