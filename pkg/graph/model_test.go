package graph

import (
	"testing"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

func TestWorkloadID(t *testing.T) {
	tests := map[string]struct {
		namespace string
		name      string
		expected  string
	}{
		"simple": {
			namespace: "default",
			name:      "nginx",
			expected:  "default/nginx",
		},
		"with dashes": {
			namespace: "domino-compute",
			name:      "my-service",
			expected:  "domino-compute/my-service",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := WorkloadID(tt.namespace, tt.name)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPortID(t *testing.T) {
	tests := map[string]struct {
		workloadID string
		port       int32
		protocol   string
		expected   string
	}{
		"tcp port": {
			workloadID: "default/nginx",
			port:       80,
			protocol:   "TCP",
			expected:   "default/nginx:TCP/80",
		},
		"udp port": {
			workloadID: "default/dns",
			port:       53,
			protocol:   "UDP",
			expected:   "default/dns:UDP/53",
		},
		"high port": {
			workloadID: "ns/app",
			port:       8443,
			protocol:   "TCP",
			expected:   "ns/app:TCP/8443",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := PortID(tt.workloadID, tt.port, tt.protocol)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := map[string]struct {
		input    int32
		expected string
	}{
		"zero": {
			input:    0,
			expected: "0",
		},
		"positive": {
			input:    80,
			expected: "80",
		},
		"large positive": {
			input:    8443,
			expected: "8443",
		},
		"negative": {
			input:    -1,
			expected: "-1",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := itoa(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNewWorkloadNode(t *testing.T) {
	tests := map[string]struct {
		workload     k8s.Workload
		expectedID   string
		expectedType NodeType
		expectedKind string
	}{
		"deployment": {
			workload: k8s.Workload{
				Name:      "nginx",
				Namespace: "default",
				Type:      k8s.WorkloadTypeDeployment,
				Labels:    map[string]string{"app": "nginx"},
			},
			expectedID:   "default/nginx",
			expectedType: NodeTypeWorkload,
			expectedKind: "Deployment",
		},
		"statefulset": {
			workload: k8s.Workload{
				Name:      "postgres",
				Namespace: "db",
				Type:      k8s.WorkloadTypeStatefulSet,
				Labels:    map[string]string{"app": "postgres"},
			},
			expectedID:   "db/postgres",
			expectedType: NodeTypeWorkload,
			expectedKind: "StatefulSet",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			node := NewWorkloadNode(tt.workload)
			if node.ID != tt.expectedID {
				t.Errorf("expected ID %q, got %q", tt.expectedID, node.ID)
			}
			if node.Type != tt.expectedType {
				t.Errorf("expected Type %q, got %q", tt.expectedType, node.Type)
			}
			if node.Kind != tt.expectedKind {
				t.Errorf("expected Kind %q, got %q", tt.expectedKind, node.Kind)
			}
		})
	}
}

func TestNewPortNode(t *testing.T) {
	tests := map[string]struct {
		workloadID   string
		port         k8s.Port
		expectedID   string
		expectedPort int32
	}{
		"named port": {
			workloadID: "default/nginx",
			port: k8s.Port{
				Name:          "http",
				ContainerPort: 80,
				Protocol:      corev1.ProtocolTCP,
			},
			expectedID:   "default/nginx:TCP/80",
			expectedPort: 80,
		},
		"unnamed port": {
			workloadID: "default/app",
			port: k8s.Port{
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
			expectedID:   "default/app:TCP/8080",
			expectedPort: 8080,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			node := NewPortNode(tt.workloadID, tt.port)
			if node.ID != tt.expectedID {
				t.Errorf("expected ID %q, got %q", tt.expectedID, node.ID)
			}
			if node.Port != tt.expectedPort {
				t.Errorf("expected Port %d, got %d", tt.expectedPort, node.Port)
			}
			if node.Type != NodeTypePort {
				t.Errorf("expected Type %q, got %q", NodeTypePort, node.Type)
			}
		})
	}
}

