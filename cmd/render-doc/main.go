package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dalibo/ldap2pg/v6/internal/privileges"
	"github.com/gosimple/slug"
)

func main() {
	if len(os.Args) != 2 {
		slog.Error("missing template path")
		os.Exit(1)
	}
	filename := os.Args[1]
	slog.Info("Loading template.", "filename", filename)
	t := template.New(filepath.Base(filename)).Funcs(template.FuncMap{
		"slugify": func(s string) string {
			// Avoid _ which has a meaning in Markdown.
			return strings.ReplaceAll(slug.Make(s), "_", "-")
		},
		"markdown_escape": func(s string) string {
			// Escape _ as HTML entity because RTD bugs on this. See #440
			return strings.ReplaceAll(s, "_", "&#95;")
		},
	})
	t, err := t.ParseFiles(filename)
	if err != nil {
		slog.Error("parse error", "err", err)
		os.Exit(1)
	}
	if t == nil {
		slog.Error("nil")
		os.Exit(1)
	}

	data := struct {
		Groups     map[string][]any
		Privileges map[string]map[string]any
		Defaults   map[string]map[string]any
	}{
		Groups:     make(map[string][]any),
		Privileges: make(map[string]map[string]any),
		Defaults:   make(map[string]map[string]any),
	}

	for key, items := range privileges.BuiltinsProfiles {
		l := items.([]any)
		item := l[0]
		switch item.(type) {
		case string:
			data.Groups[key] = l
		default:
			if strings.HasPrefix(key, "__default") {
				data.Defaults[key] = item.(map[string]any)
			} else {
				data.Privileges[key] = item.(map[string]any)
			}
		}
	}

	err = t.Execute(os.Stdout, data)
	if err != nil {
		slog.Error("execution error", "err", err)
		os.Exit(1)
	}
}
