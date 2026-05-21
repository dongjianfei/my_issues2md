package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/dongjianfei/issue2md/internal/cli"
)

func main() {
	opts, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 先生成到内存buffer，成功后再写文件（避免失败时清空已有文件）
	var buf bytes.Buffer
	if err := cli.Run(&buf, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 成功后再写入目标
	if opts.OutputFile != "" {
		if err := os.WriteFile(opts.OutputFile, buf.Bytes(), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write output file: %v\n", err)
			os.Exit(1)
		}
	} else {
		os.Stdout.Write(buf.Bytes())
	}
}
