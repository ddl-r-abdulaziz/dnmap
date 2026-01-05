// Package main provides the entry point for the dnmap CLI tool.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/graph"
	"github.com/ddl-r-abdulaziz/dnmap/pkg/k8s"
	"github.com/ddl-r-abdulaziz/dnmap/pkg/render"
	"k8s.io/client-go/util/homedir"
)

const (
	defaultOutputFile = "network-map.html"
)

func main() {
	var kubeconfig string
	var outputFile string
	var namespaces string

	// Set up flags
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file")
	}
	flag.StringVar(&outputFile, "output", defaultOutputFile, "output HTML file path")
	flag.StringVar(&namespaces, "namespaces", "domino-compute,domino-platform", "comma-separated list of namespaces to scan")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dnmap - Domino Network Map\n\n")
		fmt.Fprintf(os.Stderr, "Generates a visual graph of workloads and network policies in Kubernetes namespaces.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  dnmap [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if err := run(kubeconfig, outputFile, namespaces); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(kubeconfig, outputFile, namespaces string) error {
	// Create Kubernetes client
	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Parse namespaces
	nsList := k8s.ParseNamespaces(namespaces)

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
