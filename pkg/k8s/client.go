// Package k8s provides Kubernetes client functionality for fetching workloads and network policies.
package k8s

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// WorkloadType represents the type of Kubernetes workload.
type WorkloadType string

const (
	WorkloadTypeDeployment  WorkloadType = "Deployment"
	WorkloadTypeStatefulSet WorkloadType = "StatefulSet"
	WorkloadTypeDaemonSet   WorkloadType = "DaemonSet"
	WorkloadTypePod         WorkloadType = "Pod"
)

// Port represents a container port exposed by a workload.
type Port struct {
	Name          string
	ContainerPort int32
	Protocol      corev1.Protocol
}

// Workload represents a Kubernetes workload (Deployment, StatefulSet, DaemonSet, or standalone Pod).
type Workload struct {
	Name      string
	Namespace string
	Type      WorkloadType
	Labels    map[string]string
	Ports     []Port
}

// Client wraps the Kubernetes clientset.
type Client struct {
	clientset kubernetes.Interface
}

// NewClient creates a new Kubernetes client using the provided kubeconfig path.
// If kubeconfig is empty, it attempts to use in-cluster config.
func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

// NewClientWithInterface creates a new Client with a provided kubernetes.Interface.
// This is useful for testing.
func NewClientWithInterface(clientset kubernetes.Interface) *Client {
	return &Client{clientset: clientset}
}

// ParseNamespaces parses a comma-separated list of namespaces.
func ParseNamespaces(namespaces string) []string {
	parts := strings.Split(namespaces, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetWorkloads fetches all workloads from the specified namespaces.
func (c *Client) GetWorkloads(namespaces []string) ([]Workload, error) {
	ctx := context.Background()
	var workloads []Workload

	for _, ns := range namespaces {
		// Get Deployments
		deployments, err := c.clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list deployments in namespace %s: %w", ns, err)
		}
		for _, d := range deployments.Items {
			workloads = append(workloads, deploymentToWorkload(d))
		}

		// Get StatefulSets
		statefulSets, err := c.clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list statefulsets in namespace %s: %w", ns, err)
		}
		for _, s := range statefulSets.Items {
			workloads = append(workloads, statefulSetToWorkload(s))
		}

		// Get DaemonSets
		daemonSets, err := c.clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list daemonsets in namespace %s: %w", ns, err)
		}
		for _, ds := range daemonSets.Items {
			workloads = append(workloads, daemonSetToWorkload(ds))
		}
	}

	return workloads, nil
}

// GetNetworkPolicies fetches all network policies from the specified namespaces.
func (c *Client) GetNetworkPolicies(namespaces []string) ([]networkingv1.NetworkPolicy, error) {
	ctx := context.Background()
	var policies []networkingv1.NetworkPolicy

	for _, ns := range namespaces {
		policyList, err := c.clientset.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list network policies in namespace %s: %w", ns, err)
		}
		policies = append(policies, policyList.Items...)
	}

	return policies, nil
}

func deploymentToWorkload(d appsv1.Deployment) Workload {
	return Workload{
		Name:      d.Name,
		Namespace: d.Namespace,
		Type:      WorkloadTypeDeployment,
		Labels:    d.Spec.Template.Labels,
		Ports:     extractPorts(d.Spec.Template.Spec.Containers),
	}
}

func statefulSetToWorkload(s appsv1.StatefulSet) Workload {
	return Workload{
		Name:      s.Name,
		Namespace: s.Namespace,
		Type:      WorkloadTypeStatefulSet,
		Labels:    s.Spec.Template.Labels,
		Ports:     extractPorts(s.Spec.Template.Spec.Containers),
	}
}

func daemonSetToWorkload(ds appsv1.DaemonSet) Workload {
	return Workload{
		Name:      ds.Name,
		Namespace: ds.Namespace,
		Type:      WorkloadTypeDaemonSet,
		Labels:    ds.Spec.Template.Labels,
		Ports:     extractPorts(ds.Spec.Template.Spec.Containers),
	}
}

func extractPorts(containers []corev1.Container) []Port {
	var ports []Port
	for _, c := range containers {
		for _, p := range c.Ports {
			protocol := p.Protocol
			if protocol == "" {
				protocol = corev1.ProtocolTCP
			}
			ports = append(ports, Port{
				Name:          p.Name,
				ContainerPort: p.ContainerPort,
				Protocol:      protocol,
			})
		}
	}
	return ports
}

