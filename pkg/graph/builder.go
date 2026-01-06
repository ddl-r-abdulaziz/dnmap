package graph

import (
	"fmt"
	"strings"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/k8s"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// Builder constructs network graphs from Kubernetes resources.
type Builder struct {
	namespaceLabels map[string]map[string]string // namespace name -> labels
}

// NewBuilder creates a new graph builder.
func NewBuilder() *Builder {
	return &Builder{
		namespaceLabels: make(map[string]map[string]string),
	}
}

// WithNamespaceLabels sets the namespace labels for proper namespace selector matching.
func (b *Builder) WithNamespaceLabels(namespaces []k8s.NamespaceInfo) *Builder {
	for _, ns := range namespaces {
		b.namespaceLabels[ns.Name] = ns.Labels
	}
	return b
}

// Build constructs a NetworkGraph from workloads and policies.
func (b *Builder) Build(workloads []k8s.Workload, policies []k8s.Policy) *NetworkGraph {
	graph := &NetworkGraph{
		Nodes:          make([]Node, 0),
		Edges:          make([]Edge, 0),
		WarningDetails: make([]WarningDetail, 0),
	}

	// Build maps for quick lookup
	workloadMap := make(map[string]k8s.Workload)     // workloadID -> Workload
	workloadsByNS := make(map[string][]k8s.Workload) // namespace -> []Workload
	portNodes := make(map[string]Node)               // portID -> Node
	nodeIndex := make(map[string]int)                // nodeID -> index in graph.Nodes

	// Track warnings per workload (for node-level display)
	workloadWarnings := make(map[string]map[WarningType]bool) // workloadID -> set of warnings

	// Create nodes for each workload and its ports
	for _, w := range workloads {
		wID := WorkloadID(w.Namespace, w.Name)
		workloadMap[wID] = w
		workloadsByNS[w.Namespace] = append(workloadsByNS[w.Namespace], w)
		workloadWarnings[wID] = make(map[WarningType]bool)

		// Add workload node
		nodeIndex[wID] = len(graph.Nodes)
		graph.Nodes = append(graph.Nodes, NewWorkloadNode(w))

		// Add port nodes
		for _, p := range w.Ports {
			portNode := NewPortNode(wID, p)
			graph.Nodes = append(graph.Nodes, portNode)
			portNodes[portNode.ID] = portNode
		}
	}

	// Process policies to create edges and detect warnings
	edgeID := 0
	for _, policy := range policies {
		switch policy.Type {
		case k8s.PolicyTypeK8sNetworkPolicy:
			if policy.K8sNetworkPolicy != nil {
				edges, warnings, details := b.processK8sNetworkPolicyWithWarnings(policy.K8sNetworkPolicy, workloadsByNS, workloadMap, &edgeID)
				graph.Edges = append(graph.Edges, edges...)
				graph.WarningDetails = append(graph.WarningDetails, details...)
				// Merge warnings for node display
				for wID, warnSet := range warnings {
					for warn := range warnSet {
						workloadWarnings[wID][warn] = true
					}
				}
			}
		case k8s.PolicyTypeIstioAuthorizationPolicy:
			if policy.IstioAuthPolicy != nil {
				edges := b.processIstioAuthPolicy(policy.IstioAuthPolicy, workloadsByNS, &edgeID)
				graph.Edges = append(graph.Edges, edges...)
			}
		}
	}

	// Apply warnings to workload nodes
	for wID, warnSet := range workloadWarnings {
		if idx, ok := nodeIndex[wID]; ok && len(warnSet) > 0 {
			warnings := make([]WarningType, 0, len(warnSet))
			for warn := range warnSet {
				warnings = append(warnings, warn)
			}
			graph.Nodes[idx].Warnings = warnings
		}
	}

	return graph
}

// BuildFromNetworkPolicies constructs a NetworkGraph using only K8s NetworkPolicies.
// This is for backwards compatibility.
func (b *Builder) BuildFromNetworkPolicies(workloads []k8s.Workload, netPolicies []networkingv1.NetworkPolicy) *NetworkGraph {
	policies := make([]k8s.Policy, 0, len(netPolicies))
	for i := range netPolicies {
		policies = append(policies, k8s.Policy{
			Name:             netPolicies[i].Name,
			Namespace:        netPolicies[i].Namespace,
			Type:             k8s.PolicyTypeK8sNetworkPolicy,
			K8sNetworkPolicy: &netPolicies[i],
		})
	}
	return b.Build(workloads, policies)
}

// processK8sNetworkPolicy processes a K8s NetworkPolicy and returns edges.
func (b *Builder) processK8sNetworkPolicy(policy *networkingv1.NetworkPolicy, workloadsByNS map[string][]k8s.Workload, edgeID *int) []Edge {
	var edges []Edge

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

				// Generate policy YAML once per policy (elide managedFields)
				policyYAML := ""
				policyCopy := policy.DeepCopy()
				policyCopy.ManagedFields = nil
				if yamlBytes, err := yaml.Marshal(policyCopy); err == nil {
					policyYAML = string(yamlBytes)
				}

				for _, port := range allowedPorts {
					protocol := string(port.Protocol)
					if protocol == "" {
						protocol = "TCP"
					}
					portID := PortID(targetWID, port.ContainerPort, protocol)

					edge := Edge{
						ID:         fmt.Sprintf("edge-%d", *edgeID),
						Source:     sourceWID,
						Target:     portID,
						Label:      fmt.Sprintf("%s:%d", protocol, port.ContainerPort),
						Rule:       b.formatK8sRule(ingressRule, ruleIdx),
						Policy:     policy.Namespace + "/" + policy.Name,
						PolicyYAML: policyYAML,
						Metadata: map[string]string{
							"policyType": "NetworkPolicy",
							"ruleType":   "ingress",
						},
					}
					edges = append(edges, edge)
					*edgeID++
				}
			}
		}
	}

	return edges
}

// processK8sNetworkPolicyWithWarnings processes a K8s NetworkPolicy and returns edges, warnings, and warning details.
func (b *Builder) processK8sNetworkPolicyWithWarnings(policy *networkingv1.NetworkPolicy, workloadsByNS map[string][]k8s.Workload, workloadMap map[string]k8s.Workload, edgeID *int) ([]Edge, map[string]map[WarningType]bool, []WarningDetail) {
	var edges []Edge
	var warningDetails []WarningDetail
	warnings := make(map[string]map[WarningType]bool)

	policyFullName := policy.Namespace + "/" + policy.Name

	// Find workloads that this policy applies to (targets)
	targetWorkloads := b.findMatchingWorkloads(policy.Namespace, policy.Spec.PodSelector, workloadsByNS)

	// Initialize warnings map for target workloads
	for _, targetW := range targetWorkloads {
		wID := WorkloadID(targetW.Namespace, targetW.Name)
		if warnings[wID] == nil {
			warnings[wID] = make(map[WarningType]bool)
		}
	}

	// Process ingress rules
	for ruleIdx, ingressRule := range policy.Spec.Ingress {
		// Check for warnings
		hasNoPorts := len(ingressRule.Ports) == 0
		hasNoSelector := len(ingressRule.From) == 0

		// Find source workloads allowed by this rule
		sourceWorkloads := b.findSourceWorkloads(policy.Namespace, ingressRule.From, workloadsByNS)

		// For each target workload
		for _, targetW := range targetWorkloads {
			targetWID := WorkloadID(targetW.Namespace, targetW.Name)

			// Add warnings for this workload and collect details
			if hasNoPorts {
				if !warnings[targetWID][WarningNoPorts] {
					warnings[targetWID][WarningNoPorts] = true
					warningDetails = append(warningDetails, WarningDetail{
						WorkloadID:   targetWID,
						WorkloadName: targetW.Name,
						Namespace:    targetW.Namespace,
						PolicyName:   policyFullName,
						WarningType:  WarningNoPorts,
					})
				}
			}
			if hasNoSelector {
				if !warnings[targetWID][WarningNoSelector] {
					warnings[targetWID][WarningNoSelector] = true
					warningDetails = append(warningDetails, WarningDetail{
						WorkloadID:   targetWID,
						WorkloadName: targetW.Name,
						Namespace:    targetW.Namespace,
						PolicyName:   policyFullName,
						WarningType:  WarningNoSelector,
					})
				}
			}

			// Determine which ports are allowed
			allowedPorts := b.getAllowedPorts(targetW, ingressRule.Ports)

			// Create edges from each source to each allowed port
			for _, sourceW := range sourceWorkloads {
				sourceWID := WorkloadID(sourceW.Namespace, sourceW.Name)

				// Don't create self-referencing edges
				if sourceWID == targetWID {
					continue
				}

				// Generate policy YAML once per policy (elide managedFields)
				policyYAML := ""
				policyCopy := policy.DeepCopy()
				policyCopy.ManagedFields = nil
				if yamlBytes, err := yaml.Marshal(policyCopy); err == nil {
					policyYAML = string(yamlBytes)
				}

				for _, port := range allowedPorts {
					protocol := string(port.Protocol)
					if protocol == "" {
						protocol = "TCP"
					}
					portID := PortID(targetWID, port.ContainerPort, protocol)

					edge := Edge{
						ID:         fmt.Sprintf("edge-%d", *edgeID),
						Source:     sourceWID,
						Target:     portID,
						Label:      fmt.Sprintf("%s:%d", protocol, port.ContainerPort),
						Rule:       b.formatK8sRule(ingressRule, ruleIdx),
						Policy:     policyFullName,
						PolicyYAML: policyYAML,
						Metadata: map[string]string{
							"policyType": "NetworkPolicy",
							"ruleType":   "ingress",
						},
					}
					edges = append(edges, edge)
					*edgeID++
				}
			}
		}
	}

	return edges, warnings, warningDetails
}

// processIstioAuthPolicy processes an Istio AuthorizationPolicy and returns edges.
func (b *Builder) processIstioAuthPolicy(policy *k8s.IstioAuthorizationPolicy, workloadsByNS map[string][]k8s.Workload, edgeID *int) []Edge {
	var edges []Edge

	if policy == nil {
		return edges
	}

	// Find workloads that this policy applies to using the selector
	var targetWorkloads []k8s.Workload
	if policy.Spec.GetSelector() != nil && len(policy.Spec.GetSelector().GetMatchLabels()) > 0 {
		targetWorkloads = b.findWorkloadsByLabels(policy.Namespace, policy.Spec.GetSelector().GetMatchLabels(), workloadsByNS)
	} else {
		// No selector means all workloads in the namespace
		targetWorkloads = workloadsByNS[policy.Namespace]
	}

	// Process rules
	for ruleIdx, rule := range policy.Spec.GetRules() {
		if rule == nil {
			continue
		}

		// Find source workloads from the 'from' section
		sourceWorkloads := b.findIstioSourceWorkloads(policy.Namespace, rule.GetFrom(), workloadsByNS)

		// Get operations (ports) from the 'to' section
		allowedPorts := b.getIstioAllowedPorts(rule.GetTo())

		// For each target workload
		for _, targetW := range targetWorkloads {
			targetWID := WorkloadID(targetW.Namespace, targetW.Name)

			// If no specific ports in the rule, use all ports of the workload
			targetPorts := allowedPorts
			if len(targetPorts) == 0 {
				for _, p := range targetW.Ports {
					targetPorts = append(targetPorts, int(p.ContainerPort))
				}
			}

			// Generate policy YAML once per policy (elide managedFields)
			policyYAML := ""
			policyCopy := policy.DeepCopy()
			policyCopy.ManagedFields = nil
			if yamlBytes, err := yaml.Marshal(policyCopy); err == nil {
				policyYAML = string(yamlBytes)
			}

			// Create edges from each source to each allowed port
			for _, sourceW := range sourceWorkloads {
				sourceWID := WorkloadID(sourceW.Namespace, sourceW.Name)

				// Don't create self-referencing edges
				if sourceWID == targetWID {
					continue
				}

				for _, port := range targetPorts {
					protocol := "TCP" // Istio primarily uses TCP
					portID := PortID(targetWID, int32(port), protocol)

					edge := Edge{
						ID:         fmt.Sprintf("edge-%d", *edgeID),
						Source:     sourceWID,
						Target:     portID,
						Label:      fmt.Sprintf("%s:%d", protocol, port),
						Rule:       b.formatIstioRule(rule, ruleIdx),
						Policy:     policy.Namespace + "/" + policy.Name,
						PolicyYAML: policyYAML,
						Metadata: map[string]string{
							"policyType": "AuthorizationPolicy",
							"action":     policy.Spec.GetAction().String(),
						},
					}
					edges = append(edges, edge)
					*edgeID++
				}
			}
		}
	}

	return edges
}

// findWorkloadsByLabels finds workloads that match the given labels.
func (b *Builder) findWorkloadsByLabels(namespace string, labels map[string]string, workloadsByNS map[string][]k8s.Workload) []k8s.Workload {
	var result []k8s.Workload
	workloads := workloadsByNS[namespace]

	for _, w := range workloads {
		if b.labelsMatch(w.Labels, labels) {
			result = append(result, w)
		}
	}
	return result
}

// labelsMatch checks if workload labels contain all the required labels.
func (b *Builder) labelsMatch(workloadLabels, requiredLabels map[string]string) bool {
	for key, value := range requiredLabels {
		if workloadLabels[key] != value {
			return false
		}
	}
	return true
}

// findIstioSourceWorkloads finds workloads allowed by Istio 'from' rules.
func (b *Builder) findIstioSourceWorkloads(policyNamespace string, from []*k8s.IstioSource, workloadsByNS map[string][]k8s.Workload) []k8s.Workload {
	var result []k8s.Workload
	seen := make(map[string]bool)

	// If 'from' is empty, all sources are allowed (ALLOW action)
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

	for _, f := range from {
		if f == nil || f.GetSource() == nil {
			continue
		}

		source := f.GetSource()

		// Check principals (service accounts)
		if len(source.GetPrincipals()) > 0 {
			// Principals are in the format: cluster.local/ns/<namespace>/sa/<serviceaccount>
			// For simplicity, we match workloads in namespaces mentioned in principals
			for _, principal := range source.GetPrincipals() {
				ns := extractNamespaceFromPrincipal(principal)
				if ns != "" {
					for _, w := range workloadsByNS[ns] {
						wID := WorkloadID(w.Namespace, w.Name)
						if !seen[wID] {
							result = append(result, w)
							seen[wID] = true
						}
					}
				}
			}
		}

		// Check namespaces
		if len(source.GetNamespaces()) > 0 {
			for _, ns := range source.GetNamespaces() {
				for _, w := range workloadsByNS[ns] {
					wID := WorkloadID(w.Namespace, w.Name)
					if !seen[wID] {
						result = append(result, w)
						seen[wID] = true
					}
				}
			}
		}

		// If no specific principals or namespaces, check all workloads
		if len(source.GetPrincipals()) == 0 && len(source.GetNamespaces()) == 0 {
			for _, workloads := range workloadsByNS {
				for _, w := range workloads {
					wID := WorkloadID(w.Namespace, w.Name)
					if !seen[wID] {
						result = append(result, w)
						seen[wID] = true
					}
				}
			}
		}
	}

	return result
}

// extractNamespaceFromPrincipal extracts namespace from an Istio principal.
func extractNamespaceFromPrincipal(principal string) string {
	// Format: cluster.local/ns/<namespace>/sa/<serviceaccount>
	parts := strings.Split(principal, "/")
	for i, part := range parts {
		if part == "ns" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// getIstioAllowedPorts extracts allowed ports from Istio 'to' operations.
func (b *Builder) getIstioAllowedPorts(to []*k8s.IstioOperation) []int {
	var ports []int
	seen := make(map[int]bool)

	for _, t := range to {
		if t == nil || t.GetOperation() == nil {
			continue
		}
		for _, portStr := range t.GetOperation().GetPorts() {
			port := 0
			fmt.Sscanf(portStr, "%d", &port)
			if port > 0 && !seen[port] {
				ports = append(ports, port)
				seen[port] = true
			}
		}
	}

	return ports
}

// formatIstioRule creates a human-readable description of an Istio rule.
func (b *Builder) formatIstioRule(rule *k8s.IstioRule, idx int) string {
	var parts []string

	// Describe sources
	if len(rule.GetFrom()) == 0 {
		parts = append(parts, "from: all")
	} else {
		var sources []string
		for _, f := range rule.GetFrom() {
			if f != nil && f.GetSource() != nil {
				source := f.GetSource()
				if len(source.GetPrincipals()) > 0 {
					sources = append(sources, fmt.Sprintf("principals: %v", source.GetPrincipals()))
				}
				if len(source.GetNamespaces()) > 0 {
					sources = append(sources, fmt.Sprintf("namespaces: %v", source.GetNamespaces()))
				}
			}
		}
		if len(sources) > 0 {
			parts = append(parts, "from: "+strings.Join(sources, ", "))
		}
	}

	// Describe operations
	if len(rule.GetTo()) == 0 {
		parts = append(parts, "to: all")
	} else {
		var ops []string
		for _, t := range rule.GetTo() {
			if t != nil && t.GetOperation() != nil {
				op := t.GetOperation()
				if len(op.GetPorts()) > 0 {
					ops = append(ops, fmt.Sprintf("ports: %v", op.GetPorts()))
				}
				if len(op.GetMethods()) > 0 {
					ops = append(ops, fmt.Sprintf("methods: %v", op.GetMethods()))
				}
				if len(op.GetPaths()) > 0 {
					ops = append(ops, fmt.Sprintf("paths: %v", op.GetPaths()))
				}
			}
		}
		if len(ops) > 0 {
			parts = append(parts, "to: "+strings.Join(ops, ", "))
		}
	}

	return fmt.Sprintf("AuthzPolicy Rule %d: %s", idx+1, strings.Join(parts, "; "))
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

	// Filter namespaces by their labels matching the selector
	var namespaces []string
	for ns := range workloadsByNS {
		nsLabels := b.namespaceLabels[ns]
		if b.namespaceMatchesSelector(nsLabels, *peer.NamespaceSelector) {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces
}

// namespaceMatchesSelector checks if namespace labels match the given LabelSelector.
func (b *Builder) namespaceMatchesSelector(nsLabels map[string]string, selector metav1.LabelSelector) bool {
	// Check MatchLabels
	for key, value := range selector.MatchLabels {
		if nsLabels[key] != value {
			return false
		}
	}

	// Check MatchExpressions
	for _, expr := range selector.MatchExpressions {
		labelValue, hasLabel := nsLabels[expr.Key]

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

// formatK8sRule creates a human-readable description of a K8s NetworkPolicy ingress rule.
func (b *Builder) formatK8sRule(rule networkingv1.NetworkPolicyIngressRule, idx int) string {
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

	return fmt.Sprintf("NetworkPolicy Rule %d: %s", idx+1, strings.Join(parts, "; "))
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
	} else if peer.PodSelector != nil {
		// No namespace selector with a pod selector means same namespace as the policy
		parts = append(parts, "namespaces: same as policy")
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
