# dnmap - Domino Network Map

A CLI tool that generates an interactive visualization of workloads and network policies in your Kubernetes cluster.

## Features

- **Workload Discovery**: Scans Deployments, StatefulSets, and DaemonSets across specified namespaces
- **Network Policy Analysis**: Parses NetworkPolicy resources to understand allowed traffic flows
- **Interactive Visualization**: Generates a single-page HTML with:
  - Drag-and-drop nodes
  - Physics-based layout with force-directed graph
  - Zoom and pan navigation
  - Search functionality
  - Minimap for orientation
  - Detailed tooltips showing workload metadata and policy rules
  - PNG export

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/ddl-r-abdulaziz/dnmap.git
cd dnmap

# Build the binary
make build

# The binary will be in bin/dnmap
```

### Prerequisites

- Go 1.21+
- Access to a Kubernetes cluster (via kubeconfig)

## Usage

```bash
# Basic usage - scans domino-compute and domino-platform namespaces
dnmap

# Specify output file
dnmap -output my-network-map.html

# Scan different namespaces
dnmap -namespaces default,kube-system

# Use a specific kubeconfig
dnmap -kubeconfig /path/to/kubeconfig
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-kubeconfig` | `~/.kube/config` | Path to kubeconfig file |
| `-output` | `network-map.html` | Output HTML file path |
| `-namespaces` | `domino-compute,domino-platform` | Comma-separated list of namespaces to scan |

## Output

The tool generates a single HTML file containing an interactive network graph:

- **Nodes** represent workloads (Deployments, StatefulSets, DaemonSets)
- **Small circles** attached to nodes represent exposed ports
- **Edges** represent allowed network connections as defined by NetworkPolicies
- **Tooltips** display detailed information including:
  - Workload type and namespace
  - Labels
  - The specific NetworkPolicy rule allowing the connection

### Color Legend

| Color | Type |
|-------|------|
| Green | Deployment |
| Purple | StatefulSet |
| Orange | DaemonSet |
| Cyan | Port |

## Development

```bash
# Run all checks
make all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt

# See all available targets
make help
```

## Project Structure

```
dnmap/
├── cmd/
│   └── dnmap/
│       └── main.go          # CLI entry point
├── pkg/
│   ├── k8s/
│   │   ├── client.go        # Kubernetes client
│   │   └── client_test.go
│   ├── graph/
│   │   ├── model.go         # Graph data structures
│   │   ├── model_test.go
│   │   ├── builder.go       # Graph construction logic
│   │   └── builder_test.go
│   └── render/
│       ├── html.go          # HTML/Canvas renderer
│       └── html_test.go
├── Makefile
├── go.mod
└── README.md
```

## How It Works

1. **Discovery**: Connects to your Kubernetes cluster and fetches workloads from the specified namespaces
2. **Policy Analysis**: Retrieves NetworkPolicy resources and analyzes their rules
3. **Graph Building**: Creates nodes for each workload and port, then creates edges based on NetworkPolicy ingress rules
4. **Rendering**: Generates an interactive HTML page with embedded JavaScript for visualization

## License

MIT License

