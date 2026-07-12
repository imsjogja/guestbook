// internal/handler/template.go
//
// Echo template renderer for HTML templates including HTMX partials.
package handler

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer implements echo.Renderer for HTML templates.
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer loads all HTML templates from the web/templates directory
// including nested partial templates used by HTMX.
func NewTemplateRenderer() (*TemplateRenderer, error) {
	// Parse root templates
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, err
	}

	// Parse partial templates in subdirectories (e.g. web/templates/partials/*.html)
	partialMatches, err := filepath.Glob("web/templates/*/*.html")
	if err != nil {
		return nil, err
	}
	if len(partialMatches) > 0 {
		tmpl, err = tmpl.ParseFiles(partialMatches...)
		if err != nil {
			return nil, err
		}
	}

	return &TemplateRenderer{templates: tmpl}, nil
}

// NewTemplateRendererFromFS loads templates for embedded builds.
func NewTemplateRendererFromFS() (*TemplateRenderer, error) {
	// For production builds with embedded templates,
	// use //go:embed directive. This is a placeholder
	// that falls back to file system loading.
	return NewTemplateRenderer()
}

// Render implements echo.Renderer.
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}
