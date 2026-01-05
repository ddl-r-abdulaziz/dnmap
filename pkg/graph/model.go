// Package graph provides data structures and logic for building network graphs.
package graph

import "github.com/ddl-r-abdulaziz/dnmap/pkg/k8s"

// NodeType represents the type of a graph node.
type NodeType string

const (
	NodeTypeWorkload NodeType = "workload"
	NodeTypePort     NodeType = "port"
)

// Node represents a node in the network graph.
type Node struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Type      NodeType          `json:"type"`
	Namespace string            `json:"namespace"`
	Kind      string            `json:"kind"` // For workload nodes: Deployment, StatefulSet, etc.
	Parent    string            `json:"parent,omitempty"` // For port nodes: the parent workload ID
	Port      int32             `json:"port,omitempty"`
	Protocol  string            `json:"protocol,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Edge represents a connection between nodes in the network graph.
type Edge struct {
	ID       string            `json:"id"`
	Source   string            `json:"source"` // Source node ID
	Target   string            `json:"target"` // Target node ID (port node)
	Label    string            `json:"label"`
	Rule     string            `json:"rule"` // The network policy rule that allows this connection
	Policy   string            `json:"policy"` // Name of the network policy
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NetworkGraph represents the complete network graph.
type NetworkGraph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// WorkloadID generates a unique ID for a workload node.
func WorkloadID(namespace, name string) string {
	return namespace + "/" + name
}

// PortID generates a unique ID for a port node.
func PortID(workloadID string, port int32, protocol string) string {
	return workloadID + ":" + protocol + "/" + itoa(port)
}

// itoa converts int32 to string without importing strconv.
func itoa(n int32) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// NewNode creates a workload node.
func NewWorkloadNode(w k8s.Workload) Node {
	return Node{
		ID:        WorkloadID(w.Namespace, w.Name),
		Label:     w.Name,
		Type:      NodeTypeWorkload,
		Namespace: w.Namespace,
		Kind:      string(w.Type),
		Metadata:  w.Labels,
	}
}

// NewPortNode creates a port node.
func NewPortNode(workloadID string, p k8s.Port) Node {
	protocol := string(p.Protocol)
	if protocol == "" {
		protocol = "TCP"
	}
	
	label := p.Name
	if label == "" {
		label = itoa(p.ContainerPort)
	}
	
	return Node{
		ID:       PortID(workloadID, p.ContainerPort, protocol),
		Label:    label,
		Type:     NodeTypePort,
		Parent:   workloadID,
		Port:     p.ContainerPort,
		Protocol: protocol,
	}
}

