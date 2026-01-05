// Package k8s provides Kubernetes and Istio client functionality for fetching workloads and policies.
package k8s

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	securityv1beta1 "istio.io/api/security/v1beta1"
	istiotypev1beta1 "istio.io/api/type/v1beta1"
	securityclientv1 "istio.io/client-go/pkg/apis/security/v1"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
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

// PolicyType represents the type of network policy.
type PolicyType string

const (
	PolicyTypeK8sNetworkPolicy      PolicyType = "NetworkPolicy"
	PolicyTypeIstioAuthorizationPolicy PolicyType = "AuthorizationPolicy"
)

// Policy represents a unified view of network policies (both K8s NetworkPolicy and Istio AuthorizationPolicy).
type Policy struct {
	Name      string
	Namespace string
	Type      PolicyType
	// For K8s NetworkPolicy
	K8sNetworkPolicy *networkingv1.NetworkPolicy
	// For Istio AuthorizationPolicy
	IstioAuthPolicy *securityclientv1.AuthorizationPolicy
}

// Client wraps the Kubernetes and Istio clientsets.
type Client struct {
	k8sClientset   kubernetes.Interface
	istioClientset istioclient.Interface
}

// NewClient creates a new Kubernetes and Istio client using the provided kubeconfig path.
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

	k8sClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	istioClientset, err := istioclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create istio clientset: %w", err)
	}

	return &Client{
		k8sClientset:   k8sClientset,
		istioClientset: istioClientset,
	}, nil
}

// NewClientWithInterface creates a new Client with provided interfaces.
// This is useful for testing.
func NewClientWithInterface(k8s kubernetes.Interface, istio istioclient.Interface) *Client {
	return &Client{
		k8sClientset:   k8s,
		istioClientset: istio,
	}
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
		deployments, err := c.k8sClientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list deployments in namespace %s: %w", ns, err)
		}
		for _, d := range deployments.Items {
			workloads = append(workloads, deploymentToWorkload(d))
		}

		// Get StatefulSets
		statefulSets, err := c.k8sClientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list statefulsets in namespace %s: %w", ns, err)
		}
		for _, s := range statefulSets.Items {
			workloads = append(workloads, statefulSetToWorkload(s))
		}

		// Get DaemonSets
		daemonSets, err := c.k8sClientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list daemonsets in namespace %s: %w", ns, err)
		}
		for _, ds := range daemonSets.Items {
			workloads = append(workloads, daemonSetToWorkload(ds))
		}
	}

	return workloads, nil
}

// GetPolicies fetches all network policies (K8s and Istio) from the specified namespaces.
func (c *Client) GetPolicies(namespaces []string) ([]Policy, error) {
	ctx := context.Background()
	var policies []Policy

	for _, ns := range namespaces {
		// Get K8s NetworkPolicies
		netPolicies, err := c.k8sClientset.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list network policies in namespace %s: %w", ns, err)
		}
		for i := range netPolicies.Items {
			policies = append(policies, Policy{
				Name:             netPolicies.Items[i].Name,
				Namespace:        netPolicies.Items[i].Namespace,
				Type:             PolicyTypeK8sNetworkPolicy,
				K8sNetworkPolicy: &netPolicies.Items[i],
			})
		}

		// Get Istio AuthorizationPolicies
		if c.istioClientset != nil {
			authPolicies, err := c.istioClientset.SecurityV1().AuthorizationPolicies(ns).List(ctx, metav1.ListOptions{})
			if err != nil {
				// Istio might not be installed, so we just log and continue
				fmt.Printf("Warning: failed to list Istio AuthorizationPolicies in namespace %s: %v\n", ns, err)
			} else {
				for _, ap := range authPolicies.Items {
					policies = append(policies, Policy{
						Name:            ap.Name,
						Namespace:       ap.Namespace,
						Type:            PolicyTypeIstioAuthorizationPolicy,
						IstioAuthPolicy: ap,
					})
				}
			}
		}
	}

	return policies, nil
}

// GetNetworkPolicies fetches K8s NetworkPolicies from the specified namespaces.
// Deprecated: Use GetPolicies instead for unified policy access.
func (c *Client) GetNetworkPolicies(namespaces []string) ([]networkingv1.NetworkPolicy, error) {
	ctx := context.Background()
	var policies []networkingv1.NetworkPolicy

	for _, ns := range namespaces {
		policyList, err := c.k8sClientset.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list network policies in namespace %s: %w", ns, err)
		}
		policies = append(policies, policyList.Items...)
	}

	return policies, nil
}

// GetAuthorizationPolicies fetches Istio AuthorizationPolicies from the specified namespaces.
func (c *Client) GetAuthorizationPolicies(namespaces []string) ([]*securityclientv1.AuthorizationPolicy, error) {
	ctx := context.Background()
	var policies []*securityclientv1.AuthorizationPolicy

	if c.istioClientset == nil {
		return policies, nil
	}

	for _, ns := range namespaces {
		policyList, err := c.istioClientset.SecurityV1().AuthorizationPolicies(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list authorization policies in namespace %s: %w", ns, err)
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

// Helper types for Istio API access - re-exported for graph builder
type (
	// IstioAuthorizationPolicy is an alias for the Istio AuthorizationPolicy type.
	IstioAuthorizationPolicy = securityclientv1.AuthorizationPolicy
	// IstioRule is an alias for the Istio Rule type.
	IstioRule = securityv1beta1.Rule
	// IstioSource is an alias for the Istio Source type.
	IstioSource = securityv1beta1.Rule_From
	// IstioOperation is an alias for the Istio Operation type.
	IstioOperation = securityv1beta1.Rule_To
	// IstioWorkloadSelector is an alias for the Istio WorkloadSelector type.
	IstioWorkloadSelector = istiotypev1beta1.WorkloadSelector
)

// Ensure imports are used
var (
	_ runtime.Object = (*securityclientv1.AuthorizationPolicy)(nil)
)
