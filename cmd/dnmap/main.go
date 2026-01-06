// Package main provides the entry point for the dnmap CLI tool.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/graph"
	"github.com/ddl-r-abdulaziz/dnmap/pkg/k8s"
	"github.com/ddl-r-abdulaziz/dnmap/pkg/render"
)

const (
	defaultOutputFile = "network-map.html"
)

func main() {
	var kubeconfig string
	var outputFile string
	var namespaces string
	var serve bool
	var port string
	var refreshInterval time.Duration

	// Set up flags
	// Don't set a default kubeconfig path - let the client use standard kubectl loading rules
	// which respect KUBECONFIG env var and fall back to ~/.kube/config
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file (default: uses KUBECONFIG env or ~/.kube/config)")
	flag.StringVar(&outputFile, "output", defaultOutputFile, "output HTML file path")
	flag.StringVar(&namespaces, "namespaces", "domino-compute,domino-platform", "comma-separated list of namespaces to scan")
	flag.BoolVar(&serve, "serve", false, "serve the generated HTML via HTTP")
	flag.StringVar(&port, "port", "8080", "HTTP server port (when --serve is enabled)")
	flag.DurationVar(&refreshInterval, "refresh", 5*time.Minute, "refresh interval for regenerating the map (when --serve is enabled)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dnmap - Domino Network Map\n\n")
		fmt.Fprintf(os.Stderr, "Generates a visual graph of workloads and network policies in Kubernetes namespaces.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  dnmap [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if err := run(kubeconfig, outputFile, namespaces, serve, port, refreshInterval); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(kubeconfig, outputFile, namespaces string, serve bool, port string, refreshInterval time.Duration) error {
	// Create Kubernetes client
	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Parse namespaces
	nsList := k8s.ParseNamespaces(namespaces)

	// Generate the initial map
	if err := generateMap(client, nsList, outputFile); err != nil {
		return err
	}

	// If not serving, we're done
	if !serve {
		return nil
	}

	// Start background refresh
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Printf("Refreshing network map...\n")
			if err := generateMap(client, nsList, outputFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error refreshing map: %v\n", err)
			}
		}
	}()

	// Serve the HTML file
	dir := filepath.Dir(outputFile)
	file := filepath.Base(outputFile)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/"+file {
			http.ServeFile(w, r, outputFile)
		} else {
			http.NotFound(w, r)
		}
	})

	// Health check endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	fmt.Printf("Serving network map at http://0.0.0.0:%s/ (refresh every %v)\n", port, refreshInterval)
	fmt.Printf("Serving from directory: %s\n", dir)
	return http.ListenAndServe(":"+port, nil)
}

func generateMap(client *k8s.Client, nsList []string, outputFile string) error {
	// Fetch workloads and policies
	fmt.Printf("Scanning namespaces: %v\n", nsList)

	workloads, err := client.GetWorkloads(nsList)
	if err != nil {
		return fmt.Errorf("failed to get workloads: %w", err)
	}
	fmt.Printf("Found %d workloads\n", len(workloads))

	policies, err := client.GetPolicies(nsList)
	if err != nil {
		return fmt.Errorf("failed to get policies: %w", err)
	}

	// Count policy types
	var k8sPolicies, istioPolicies int
	for _, p := range policies {
		switch p.Type {
		case k8s.PolicyTypeK8sNetworkPolicy:
			k8sPolicies++
		case k8s.PolicyTypeIstioAuthorizationPolicy:
			istioPolicies++
		}
	}
	fmt.Printf("Found %d K8s NetworkPolicies, %d Istio AuthorizationPolicies\n", k8sPolicies, istioPolicies)

	// Build the graph
	builder := graph.NewBuilder()
	networkGraph := builder.Build(workloads, policies)
	fmt.Printf("Generated graph with %d nodes and %d edges\n", len(networkGraph.Nodes), len(networkGraph.Edges))

	// Render to HTML
	renderer, err := render.NewHTMLRenderer()
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}
	html, err := renderer.Render(networkGraph)
	if err != nil {
		return fmt.Errorf("failed to render graph: %w", err)
	}

	// Write output file
	if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Network map written to: %s\n", outputFile)
	return nil
}
