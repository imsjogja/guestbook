// internal/handler/embed.go
//
// Embeds web assets into the binary for production builds.
// Use 'go build -tags embed' to use embedded assets.
// Default (no tag) loads from filesystem for development.

//go:build embed

package handler

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed all:web/templates/*.html
var templatesFS embed.FS

//go:embed all:web/static
var staticFS embed.FS

// NewTemplateRendererFromEmbed loads templates from embedded filesystem.
func NewTemplateRendererFromEmbed() (*TemplateRenderer, error) {
	tmplFS, err := fs.Sub(templatesFS, "web/templates")
	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFS(tmplFS, "*.html")
	if err != nil {
		return nil, err
	}

	return &TemplateRenderer{templates: tmpl}, nil
}

// StaticFS returns the embedded static file system.
func StaticFS() (fs.FS, error) {
	return fs.Sub(staticFS, "web/static")
}
