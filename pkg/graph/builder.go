package graph

import (
	"fmt"
	"strings"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/k8s"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder constructs network graphs from Kubernetes resources.
type Builder struct{}

// NewBuilder creates a new graph builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build constructs a NetworkGraph from workloads and network policies.
func (b *Builder) Build(workloads []k8s.Workload, policies []networkingv1.NetworkPolicy) *NetworkGraph {
	graph := &NetworkGraph{
		Nodes: make([]Node, 0),
		Edges: make([]Edge, 0),
	}

	// Build maps for quick lookup
	workloadMap := make(map[string]k8s.Workload)      // workloadID -> Workload
	workloadsByNS := make(map[string][]k8s.Workload)  // namespace -> []Workload
	portNodes := make(map[string]Node)                // portID -> Node

	// Create nodes for each workload and its ports
	for _, w := range workloads {
		wID := WorkloadID(w.Namespace, w.Name)
		workloadMap[wID] = w
		workloadsByNS[w.Namespace] = append(workloadsByNS[w.Namespace], w)

		// Add workload node
		graph.Nodes = append(graph.Nodes, NewWorkloadNode(w))

		// Add port nodes
		for _, p := range w.Ports {
			portNode := NewPortNode(wID, p)
			graph.Nodes = append(graph.Nodes, portNode)
			portNodes[portNode.ID] = portNode
		}
	}

	// Process network policies to create edges
	edgeID := 0
	for _, policy := range policies {
		// Find workloads that this policy applies to (targets)
		targetWorkloads := b.findMatchingWorkloads(policy.Namespace, policy.Spec.PodSelector, workloadsByNS)

		// Process ingress rules
		for ruleIdx, ingressRule := range policy.Spec.Ingress {
			// Find source workloads allowed by this rule
			sourceWorkloads := b.findSourceWorkloads(policy.Namespace, ingressRule.From, workloadsByNS)

			// For each target workload
			for _, targetW := range targetWorkloads {
				targetWID := WorkloadID(targetW.Namespace, targetW.Name)

				// Determine which ports are allowed
				allowedPorts := b.getAllowedPorts(targetW, ingressRule.Ports)

				// Create edges from each source to each allowed port
				for _, sourceW := range sourceWorkloads {
					sourceWID := WorkloadID(sourceW.Namespace, sourceW.Name)

					// Don't create self-referencing edges
					if sourceWID == targetWID {
						continue
					}

					for _, port := range allowedPorts {
						protocol := string(port.Protocol)
						if protocol == "" {
							protocol = "TCP"
						}
						portID := PortID(targetWID, port.ContainerPort, protocol)

						edge := Edge{
							ID:     fmt.Sprintf("edge-%d", edgeID),
							Source: sourceWID,
							Target: portID,
							Label:  fmt.Sprintf("%s:%d", protocol, port.ContainerPort),
							Rule:   b.formatRule(ingressRule, ruleIdx),
							Policy: policy.Namespace + "/" + policy.Name,
							Metadata: map[string]string{
								"ruleType": "ingress",
							},
						}
						graph.Edges = append(graph.Edges, edge)
						edgeID++
					}
				}
			}
		}
	}

	return graph
}

// findMatchingWorkloads finds workloads that match the given label selector in the specified namespace.
func (b *Builder) findMatchingWorkloads(namespace string, selector metav1.LabelSelector, workloadsByNS map[string][]k8s.Workload) []k8s.Workload {
	var result []k8s.Workload
	workloads := workloadsByNS[namespace]

	for _, w := range workloads {
		if b.matchesSelector(w.Labels, selector) {
			result = append(result, w)
		}
	}
	return result
}

// findSourceWorkloads finds workloads that are allowed as sources by the ingress rule.
func (b *Builder) findSourceWorkloads(policyNamespace string, from []networkingv1.NetworkPolicyPeer, workloadsByNS map[string][]k8s.Workload) []k8s.Workload {
	var result []k8s.Workload
	seen := make(map[string]bool)

	// If 'from' is empty, all sources are allowed
	if len(from) == 0 {
		for _, workloads := range workloadsByNS {
			for _, w := range workloads {
				wID := WorkloadID(w.Namespace, w.Name)
				if !seen[wID] {
					result = append(result, w)
					seen[wID] = true
				}
			}
		}
		return result
	}

	for _, peer := range from {
		// Determine which namespaces to check
		namespaces := b.getNamespacesForPeer(policyNamespace, peer, workloadsByNS)

		for _, ns := range namespaces {
			workloads := workloadsByNS[ns]
			for _, w := range workloads {
				// Check if pod selector matches (if specified)
				if peer.PodSelector != nil {
					if !b.matchesSelector(w.Labels, *peer.PodSelector) {
						continue
					}
				}

				wID := WorkloadID(w.Namespace, w.Name)
				if !seen[wID] {
					result = append(result, w)
					seen[wID] = true
				}
			}
		}
	}

	return result
}

// getNamespacesForPeer determines which namespaces are relevant for a NetworkPolicyPeer.
func (b *Builder) getNamespacesForPeer(policyNamespace string, peer networkingv1.NetworkPolicyPeer, workloadsByNS map[string][]k8s.Workload) []string {
	if peer.NamespaceSelector == nil {
		// No namespace selector means same namespace as the policy
		return []string{policyNamespace}
	}

	// If namespace selector is empty ({}), it matches all namespaces
	if len(peer.NamespaceSelector.MatchLabels) == 0 && len(peer.NamespaceSelector.MatchExpressions) == 0 {
		namespaces := make([]string, 0, len(workloadsByNS))
		for ns := range workloadsByNS {
			namespaces = append(namespaces, ns)
		}
		return namespaces
	}

	// For simplicity, we check all namespaces we know about
	// In a real implementation, we would query namespace labels
	namespaces := make([]string, 0, len(workloadsByNS))
	for ns := range workloadsByNS {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// matchesSelector checks if labels match a LabelSelector.
func (b *Builder) matchesSelector(labels map[string]string, selector metav1.LabelSelector) bool {
	// Check MatchLabels
	for key, value := range selector.MatchLabels {
		if labels[key] != value {
			return false
		}
	}

	// Check MatchExpressions
	for _, expr := range selector.MatchExpressions {
		labelValue, hasLabel := labels[expr.Key]

		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			if !hasLabel {
				return false
			}
			found := false
			for _, v := range expr.Values {
				if labelValue == v {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case metav1.LabelSelectorOpNotIn:
			if hasLabel {
				for _, v := range expr.Values {
					if labelValue == v {
						return false
					}
				}
			}
		case metav1.LabelSelectorOpExists:
			if !hasLabel {
				return false
			}
		case metav1.LabelSelectorOpDoesNotExist:
			if hasLabel {
				return false
			}
		}
	}

	return true
}

// getAllowedPorts determines which ports are allowed by the ingress rule.
func (b *Builder) getAllowedPorts(w k8s.Workload, policyPorts []networkingv1.NetworkPolicyPort) []k8s.Port {
	// If no ports are specified, all ports are allowed
	if len(policyPorts) == 0 {
		return w.Ports
	}

	var allowed []k8s.Port
	for _, wPort := range w.Ports {
		for _, pPort := range policyPorts {
			if b.portMatches(wPort, pPort) {
				allowed = append(allowed, wPort)
				break
			}
		}
	}
	return allowed
}

// portMatches checks if a workload port matches a policy port specification.
func (b *Builder) portMatches(wPort k8s.Port, pPort networkingv1.NetworkPolicyPort) bool {
	// Check protocol
	if pPort.Protocol != nil && *pPort.Protocol != wPort.Protocol {
		return false
	}

	// Check port number/name
	if pPort.Port != nil {
		// IntOrString can be an int or a string (port name)
		if pPort.Port.Type == 0 { // Int
			if int32(pPort.Port.IntVal) != wPort.ContainerPort {
				return false
			}
		} else { // String (port name)
			if pPort.Port.StrVal != wPort.Name {
				return false
			}
		}
	}

	return true
}

// formatRule creates a human-readable description of an ingress rule.
func (b *Builder) formatRule(rule networkingv1.NetworkPolicyIngressRule, idx int) string {
	var parts []string

	// Describe sources
	if len(rule.From) == 0 {
		parts = append(parts, "from: all")
	} else {
		var sources []string
		for _, from := range rule.From {
			sources = append(sources, b.formatPeer(from))
		}
		parts = append(parts, "from: "+strings.Join(sources, ", "))
	}

	// Describe ports
	if len(rule.Ports) == 0 {
		parts = append(parts, "ports: all")
	} else {
		var ports []string
		for _, p := range rule.Ports {
			ports = append(ports, b.formatPolicyPort(p))
		}
		parts = append(parts, "ports: "+strings.Join(ports, ", "))
	}

	return fmt.Sprintf("Ingress Rule %d: %s", idx+1, strings.Join(parts, "; "))
}

// formatPeer creates a human-readable description of a NetworkPolicyPeer.
func (b *Builder) formatPeer(peer networkingv1.NetworkPolicyPeer) string {
	var parts []string

	if peer.PodSelector != nil {
		if len(peer.PodSelector.MatchLabels) == 0 && len(peer.PodSelector.MatchExpressions) == 0 {
			parts = append(parts, "pods: all")
		} else {
			parts = append(parts, fmt.Sprintf("pods: %v", peer.PodSelector.MatchLabels))
		}
	}

	if peer.NamespaceSelector != nil {
		if len(peer.NamespaceSelector.MatchLabels) == 0 && len(peer.NamespaceSelector.MatchExpressions) == 0 {
			parts = append(parts, "namespaces: all")
		} else {
			parts = append(parts, fmt.Sprintf("namespaces: %v", peer.NamespaceSelector.MatchLabels))
		}
	}

	if peer.IPBlock != nil {
		parts = append(parts, fmt.Sprintf("cidr: %s", peer.IPBlock.CIDR))
	}

	if len(parts) == 0 {
		return "any"
	}
	return strings.Join(parts, ", ")
}

// formatPolicyPort creates a human-readable description of a NetworkPolicyPort.
func (b *Builder) formatPolicyPort(p networkingv1.NetworkPolicyPort) string {
	protocol := "TCP"
	if p.Protocol != nil {
		protocol = string(*p.Protocol)
	}

	if p.Port == nil {
		return protocol + "/*"
	}

	if p.Port.Type == 0 {
		return fmt.Sprintf("%s/%d", protocol, p.Port.IntVal)
	}
	return fmt.Sprintf("%s/%s", protocol, p.Port.StrVal)
}

