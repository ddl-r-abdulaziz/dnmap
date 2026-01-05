package k8s

import (
	"testing"
)

func TestParseNamespaces(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"single namespace": {
			input:    "default",
			expected: []string{"default"},
		},
		"two namespaces": {
			input:    "domino-compute,domino-platform",
			expected: []string{"domino-compute", "domino-platform"},
		},
		"namespaces with spaces": {
			input:    " ns1 , ns2 , ns3 ",
			expected: []string{"ns1", "ns2", "ns3"},
		},
		"empty string": {
			input:    "",
			expected: []string{},
		},
		"only commas": {
			input:    ",,",
			expected: []string{},
		},
		"mixed empty and valid": {
			input:    "ns1,,ns2",
			expected: []string{"ns1", "ns2"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := ParseNamespaces(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d namespaces, got %d", len(tt.expected), len(result))
				return
			}
			for i, ns := range result {
				if ns != tt.expected[i] {
					t.Errorf("expected namespace[%d] = %q, got %q", i, tt.expected[i], ns)
				}
			}
		})
	}
}

