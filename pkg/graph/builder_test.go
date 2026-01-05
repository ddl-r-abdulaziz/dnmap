package graph

import (
	"testing"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuilderMatchesSelector(t *testing.T) {
	builder := NewBuilder()

	tests := map[string]struct {
		labels   map[string]string
		selector metav1.LabelSelector
		expected bool
	}{
		"empty selector matches all": {
			labels:   map[string]string{"app": "nginx"},
			selector: metav1.LabelSelector{},
			expected: true,
		},
		"exact match": {
			labels: map[string]string{"app": "nginx", "env": "prod"},
			selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nginx"},
			},
			expected: true,
		},
		"no match": {
			labels: map[string]string{"app": "nginx"},
			selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "redis"},
			},
			expected: false,
		},
		"missing label": {
			labels: map[string]string{"app": "nginx"},
			selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nginx", "env": "prod"},
			},
			expected: false,
		},
		"in operator match": {
			labels: map[string]string{"tier": "frontend"},
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "tier", Operator: metav1.LabelSelectorOpIn, Values: []string{"frontend", "backend"}},
				},
			},
			expected: true,
		},
		"in operator no match": {
			labels: map[string]string{"tier": "database"},
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "tier", Operator: metav1.LabelSelectorOpIn, Values: []string{"frontend", "backend"}},
				},
			},
			expected: false,
		},
		"notin operator match": {
			labels: map[string]string{"env": "dev"},
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "env", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"prod", "staging"}},
				},
			},
			expected: true,
		},
		"exists operator match": {
			labels: map[string]string{"app": "nginx"},
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "app", Operator: metav1.LabelSelectorOpExists},
				},
			},
			expected: true,
		},
		"exists operator no match": {
			labels: map[string]string{"other": "value"},
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "app", Operator: metav1.LabelSelectorOpExists},
				},
			},
			expected: false,
		},
		"does not exist match": {
			labels: map[string]string{"app": "nginx"},
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "deprecated", Operator: metav1.LabelSelectorOpDoesNotExist},
				},
			},
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := builder.matchesSelector(tt.labels, tt.selector)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuilderBuild(t *testing.T) {
	builder := NewBuilder()

	tests := map[string]struct {
		workloads     []k8s.Workload
		policies      []networkingv1.NetworkPolicy
		expectedNodes int
		expectedEdges int
	}{
		"empty inputs": {
			workloads:     []k8s.Workload{},
			policies:      []networkingv1.NetworkPolicy{},
			expectedNodes: 0,
			expectedEdges: 0,
		},
		"workloads without policies": {
			workloads: []k8s.Workload{
				{
					Name:      "nginx",
					Namespace: "default",
					Type:      k8s.WorkloadTypeDeployment,
					Labels:    map[string]string{"app": "nginx"},
					Ports: []k8s.Port{
						{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP},
					},
				},
			},
			policies:      []networkingv1.NetworkPolicy{},
			expectedNodes: 2, // 1 workload + 1 port
			expectedEdges: 0,
		},
		"workload with policy allowing access": {
			workloads: []k8s.Workload{
				{
					Name:      "frontend",
					Namespace: "default",
					Type:      k8s.WorkloadTypeDeployment,
					Labels:    map[string]string{"app": "frontend"},
					Ports:     []k8s.Port{},
				},
				{
					Name:      "backend",
					Namespace: "default",
					Type:      k8s.WorkloadTypeDeployment,
					Labels:    map[string]string{"app": "backend"},
					Ports: []k8s.Port{
						{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
					},
				},
			},
			policies: []networkingv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "allow-frontend",
						Namespace: "default",
					},
					Spec: networkingv1.NetworkPolicySpec{
						PodSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "backend"},
						},
						Ingress: []networkingv1.NetworkPolicyIngressRule{
							{
								From: []networkingv1.NetworkPolicyPeer{
									{
										PodSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{"app": "frontend"},
										},
									},
								},
								Ports: []networkingv1.NetworkPolicyPort{
									{
										Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
									},
								},
							},
						},
					},
				},
			},
			expectedNodes: 3, // 2 workloads + 1 port
			expectedEdges: 1, // frontend -> backend:8080
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			graph := builder.Build(tt.workloads, tt.policies)

			if len(graph.Nodes) != tt.expectedNodes {
				t.Errorf("expected %d nodes, got %d", tt.expectedNodes, len(graph.Nodes))
			}
			if len(graph.Edges) != tt.expectedEdges {
				t.Errorf("expected %d edges, got %d", tt.expectedEdges, len(graph.Edges))
			}
		})
	}
}

func TestBuilderPortMatches(t *testing.T) {
	builder := NewBuilder()
	tcp := corev1.ProtocolTCP
	udp := corev1.ProtocolUDP

	tests := map[string]struct {
		workloadPort k8s.Port
		policyPort   networkingv1.NetworkPolicyPort
		expected     bool
	}{
		"nil port matches all": {
			workloadPort: k8s.Port{ContainerPort: 80, Protocol: corev1.ProtocolTCP},
			policyPort:   networkingv1.NetworkPolicyPort{},
			expected:     true,
		},
		"exact port match": {
			workloadPort: k8s.Port{ContainerPort: 80, Protocol: corev1.ProtocolTCP},
			policyPort: networkingv1.NetworkPolicyPort{
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 80},
				Protocol: &tcp,
			},
			expected: true,
		},
		"port number mismatch": {
			workloadPort: k8s.Port{ContainerPort: 80, Protocol: corev1.ProtocolTCP},
			policyPort: networkingv1.NetworkPolicyPort{
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				Protocol: &tcp,
			},
			expected: false,
		},
		"protocol mismatch": {
			workloadPort: k8s.Port{ContainerPort: 53, Protocol: corev1.ProtocolTCP},
			policyPort: networkingv1.NetworkPolicyPort{
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
				Protocol: &udp,
			},
			expected: false,
		},
		"named port match": {
			workloadPort: k8s.Port{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP},
			policyPort: networkingv1.NetworkPolicyPort{
				Port:     &intstr.IntOrString{Type: intstr.String, StrVal: "http"},
				Protocol: &tcp,
			},
			expected: true,
		},
		"named port mismatch": {
			workloadPort: k8s.Port{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP},
			policyPort: networkingv1.NetworkPolicyPort{
				Port:     &intstr.IntOrString{Type: intstr.String, StrVal: "https"},
				Protocol: &tcp,
			},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := builder.portMatches(tt.workloadPort, tt.policyPort)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

