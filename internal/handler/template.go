// internal/handler/template.go
//
// Echo template renderer for HTML views.
package handler

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer implements echo.Renderer for HTML templates.
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer loads all HTML templates from the web/templates directory.
func NewTemplateRenderer() (*TemplateRenderer, error) {
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, err
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
