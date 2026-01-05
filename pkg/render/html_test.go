package render

import (
	"strings"
	"testing"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/graph"
)

func TestHTMLRendererRender(t *testing.T) {
	renderer := NewHTMLRenderer()

	tests := map[string]struct {
		graph           *graph.NetworkGraph
		expectSubstring []string
	}{
		"empty graph": {
			graph: &graph.NetworkGraph{
				Nodes: []graph.Node{},
				Edges: []graph.Edge{},
			},
			expectSubstring: []string{
				"<!DOCTYPE html>",
				"dnmap",
				"graphData",
			},
		},
		"graph with nodes": {
			graph: &graph.NetworkGraph{
				Nodes: []graph.Node{
					{
						ID:        "default/nginx",
						Label:     "nginx",
						Type:      graph.NodeTypeWorkload,
						Namespace: "default",
						Kind:      "Deployment",
					},
					{
						ID:       "default/nginx:TCP/80",
						Label:    "http",
						Type:     graph.NodeTypePort,
						Parent:   "default/nginx",
						Port:     80,
						Protocol: "TCP",
					},
				},
				Edges: []graph.Edge{},
			},
			expectSubstring: []string{
				"nginx",
				"default/nginx",
				"TCP/80",
			},
		},
		"graph with edges": {
			graph: &graph.NetworkGraph{
				Nodes: []graph.Node{
					{ID: "default/frontend", Label: "frontend", Type: graph.NodeTypeWorkload},
					{ID: "default/backend", Label: "backend", Type: graph.NodeTypeWorkload},
					{ID: "default/backend:TCP/8080", Label: "8080", Type: graph.NodeTypePort, Parent: "default/backend"},
				},
				Edges: []graph.Edge{
					{
						ID:     "edge-0",
						Source: "default/frontend",
						Target: "default/backend:TCP/8080",
						Label:  "TCP:8080",
						Rule:   "allow-frontend",
						Policy: "default/allow-frontend",
					},
				},
			},
			expectSubstring: []string{
				"frontend",
				"backend",
				"edge-0",
				"allow-frontend",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			html, err := renderer.Render(tt.graph)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, substr := range tt.expectSubstring {
				if !strings.Contains(html, substr) {
					t.Errorf("expected HTML to contain %q", substr)
				}
			}
		})
	}
}

