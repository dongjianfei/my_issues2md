package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"

	"github.com/dongjianfei/issue2md/internal/cli"
)

//go:embed templates/*
var templateFS embed.FS

// convertFunc is the core dependency — matches cli.Run signature
type convertFunc func(w io.Writer, opts *cli.RunOptions) error

// Server holds all dependencies
type Server struct {
	convert   convertFunc
	templates *template.Template
	mux       *http.ServeMux
}

// NewServer creates a Server instance and registers routes
func NewServer(fn convertFunc) *Server {
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))

	s := &Server{
		convert:   fn,
		templates: tmpl,
		mux:       http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("POST /convert", s.handleConvert)
	s.mux.HandleFunc("POST /download", s.handleDownload)
	s.mux.HandleFunc("POST /download-all", s.handleDownloadAll)

	return s
}
