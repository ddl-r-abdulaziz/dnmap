# dnmap - Domino Network Map


Just deploy it on your cluster

* Point your kubectx at your desired clustert
* Deploy the app
    ```
    make deploy
    ```
* visit https://<your-custer>/dnmap


---- 
# lots of lies below here

A CLI tool that generates an interactive visualization of workloads and network policies in your Kubernetes cluster. Supports both Kubernetes NetworkPolicies and Istio AuthorizationPolicies.

## Features

- **Workload Discovery**: Scans Deployments, StatefulSets, and DaemonSets across specified namespaces
- **Network Policy Analysis**: Parses Kubernetes NetworkPolicy and Istio AuthorizationPolicy resources to understand allowed traffic flows
- **Unified Policy View**: Combines K8s and Istio policies into a single network graph
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
- Optional: Istio installed for AuthorizationPolicy support

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
- **Edges** represent allowed network connections as defined by NetworkPolicies or AuthorizationPolicies
- **Tooltips** display detailed information including:
  - Workload type and namespace
  - Labels
  - The specific policy rule allowing the connection
  - Policy type (NetworkPolicy or AuthorizationPolicy)

### Color Legend

| Color | Type |
|-------|------|
| Green | Deployment |
| Purple | StatefulSet |
| Orange | DaemonSet |
| Cyan | Port |

## Supported Policies

### Kubernetes NetworkPolicy
- Pod selectors for target workloads
- Ingress rules with pod/namespace selectors
- Port specifications (named and numbered)
- Protocol specifications (TCP, UDP)

### Istio AuthorizationPolicy
- Workload selectors
- Source principals and namespaces
- Operation ports, methods, and paths
- ALLOW/DENY actions

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
├── cmd/dnmap/
│   └── main.go                      # CLI entry point
├── pkg/
│   ├── k8s/
│   │   ├── client.go                # K8s and Istio client
│   │   └── client_test.go
│   ├── graph/
│   │   ├── model.go                 # Graph data structures
│   │   ├── model_test.go
│   │   ├── builder.go               # Graph construction from policies
│   │   └── builder_test.go
│   └── render/
│       ├── html.go                  # HTML/Canvas renderer
│       ├── html_test.go
│       └── templates/
│           └── graph.html.tmpl      # Embedded HTML template
├── Makefile
├── go.mod
└── README.md
```

## How It Works

1. **Discovery**: Connects to your Kubernetes cluster and fetches workloads from the specified namespaces
2. **Policy Analysis**: Retrieves both K8s NetworkPolicy and Istio AuthorizationPolicy resources
3. **Graph Building**: 
   - Creates nodes for each workload and port
   - Analyzes K8s NetworkPolicy ingress rules to create edges
   - Analyzes Istio AuthorizationPolicy rules to create edges
   - Combines all edges with metadata about the originating policy
4. **Rendering**: Generates an interactive HTML page using embedded Go templates

## API Dependencies

- `k8s.io/api` - Kubernetes API types
- `k8s.io/client-go` - Kubernetes client library
- `istio.io/api` - Istio API types
- `istio.io/client-go` - Istio client library

## License

MIT License
