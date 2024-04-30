package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gosimple/slug"

	"github.com/dalibo/ldap2pg/internal/config"
)

func main() {
	if 2 != len(os.Args) {
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
		Groups   map[string][]interface{}
		Refs     map[string]map[string]interface{}
		Defaults map[string]map[string]interface{}
	}{
		Groups:   make(map[string][]interface{}),
		Refs:     make(map[string]map[string]interface{}),
		Defaults: make(map[string]map[string]interface{}),
	}

	for key, items := range config.BuiltinsProfiles {
		l := items.([]interface{})
		item := l[0]
		switch item.(type) {
		case string:
			data.Groups[key] = l
		default:
			if strings.HasPrefix(key, "__default") {
				data.Defaults[key] = item.(map[string]interface{})
			} else {
				data.Refs[key] = item.(map[string]interface{})
			}
		}
	}

	err = t.Execute(os.Stdout, data)
	if err != nil {
		slog.Error("execution error", "err", err)
		os.Exit(1)
	}
}
