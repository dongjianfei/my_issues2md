package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/dongjianfei/issue2md/internal/cli"
)

func main() {
	port := flag.String("port", "8080", "监听端口")
	flag.Parse()

	s := NewServer(cli.Run)

	log.Printf("issue2md web server listening on :%s", *port)
	if err := http.ListenAndServe(":"+*port, s.mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
