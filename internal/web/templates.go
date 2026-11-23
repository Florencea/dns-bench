package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/Florencea/dns-bench/internal/dns"
	"github.com/Florencea/dns-bench/internal/i18n"
	webassets "github.com/Florencea/dns-bench/web"
)

// ServerCardView represents one resolver card rendered on the index page.
type ServerCardView struct {
	Index  int
	Server dns.Server
	Output string
}

// IndexPage holds all template data for the main benchmark page.
type IndexPage struct {
	Lang            string
	SwitchLocale    string
	CurrentURI      string
	T               i18n.Messages
	Mode            string
	HostInput       string
	QueriedHost     string
	ErrorMessage    string
	HasResult       bool
	TotalDurationMS int64
	Cards           []ServerCardView
}

// TemplateSet holds pre-parsed embedded HTML5 templates.
type TemplateSet struct {
	index *template.Template
}

// NewTemplateSet parses all embedded templates at startup to fail fast on syntax issues.
func NewTemplateSet() (*TemplateSet, error) {
	tmpl, err := template.New("layout").ParseFS(
		webassets.Assets,
		"templates/layout.html",
		"templates/index.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &TemplateSet{index: tmpl}, nil
}

// RenderIndex buffers and writes the rendered index HTML page.
func (ts *TemplateSet) RenderIndex(w http.ResponseWriter, status int, data IndexPage) {
	var buf bytes.Buffer
	if err := ts.index.ExecuteTemplate(&buf, "layout", data); err != nil {
		http.Error(w, "Template render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// StaticFS returns the embedded static filesystem rooted at web/static.
func StaticFS() (fs.FS, error) {
	return fs.Sub(webassets.Assets, "static")
}
