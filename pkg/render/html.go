// Package render provides HTML rendering functionality for network graphs.
package render

import (
	"bytes"
	"embed"
	"encoding/json"
	"text/template"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/graph"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// HTMLRenderer renders network graphs to interactive HTML pages.
type HTMLRenderer struct {
	tmpl *template.Template
}

// NewHTMLRenderer creates a new HTML renderer.
func NewHTMLRenderer() (*HTMLRenderer, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/graph.html.tmpl")
	if err != nil {
		return nil, err
	}
	return &HTMLRenderer{tmpl: tmpl}, nil
}

// Render converts a NetworkGraph to an interactive HTML page.
func (r *HTMLRenderer) Render(g *graph.NetworkGraph) (string, error) {
	graphJSON, err := json.Marshal(g)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, map[string]string{
		"GraphData": string(graphJSON),
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}
