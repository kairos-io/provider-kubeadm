package utils

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	. "github.com/onsi/gomega"
)

// TestGetNoProxyConfig tests the GetNoProxyConfig function
func TestGetNoProxyConfig(t *testing.T) {
	tests := []struct {
		name           string
		clusterCtx     *domain.ClusterContext
		expectedResult string
	}{
		{
			name: "with_all_network_configs",
			clusterCtx: &domain.ClusterContext{
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "192.168.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local",
		},
		{
			name: "with_only_cluster_cidr",
			clusterCtx: &domain.ClusterContext{
				ControlPlaneHost: "192.168.1.100",
				ClusterCidr:      "10.244.0.0/16",
			},
			expectedResult: "10.244.0.0/16,.svc,.svc.cluster,.svc.cluster.local",
		},
		{
			name: "with_only_service_cidr",
			clusterCtx: &domain.ClusterContext{
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "172.20.0.0/16",
			},
			expectedResult: ",172.20.0.0/16,.svc,.svc.cluster,.svc.cluster.local",
		},
		{
			name:           "empty_cluster_context",
			clusterCtx:     &domain.ClusterContext{},
			expectedResult: ",.svc,.svc.cluster,.svc.cluster.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetNoProxyConfig(tt.clusterCtx)

			g.Expect(result).To(Equal(tt.expectedResult))
		})
	}
}

// TestIsProxyConfigured tests the IsProxyConfigured function
func TestIsProxyConfigured(t *testing.T) {
	tests := []struct {
		name           string
		proxyMap       map[string]string
		expectedResult bool
	}{
		{
			name: "http_proxy_only",
			proxyMap: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
			expectedResult: true,
		},
		{
			name: "https_proxy_only",
			proxyMap: map[string]string{
				"HTTPS_PROXY": "https://proxy.example.com:8080",
			},
			expectedResult: true,
		},
		{
			name: "both_proxies",
			proxyMap: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://proxy.example.com:8080",
			},
			expectedResult: true,
		},
		{
			name: "no_proxy_only",
			proxyMap: map[string]string{
				"NO_PROXY": "localhost,127.0.0.1",
			},
			expectedResult: false,
		},
		{
			name:           "empty_map",
			proxyMap:       map[string]string{},
			expectedResult: false,
		},
		{
			name:           "nil_map",
			proxyMap:       nil,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := IsProxyConfigured(tt.proxyMap)

			g.Expect(result).To(Equal(tt.expectedResult))
		})
	}
}

// TestGetDefaultNoProxy tests the GetDefaultNoProxy function
func TestGetDefaultNoProxy(t *testing.T) {
	tests := []struct {
		name           string
		clusterCtx     *domain.ClusterContext
		expectedResult string
	}{
		{
			name: "with_all_network_configs",
			clusterCtx: &domain.ClusterContext{
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "192.168.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local",
		},
		{
			name: "with_only_cluster_cidr",
			clusterCtx: &domain.ClusterContext{
				ControlPlaneHost: "192.168.1.100",
				ClusterCidr:      "10.244.0.0/16",
			},
			expectedResult: "10.244.0.0/16,.svc,.svc.cluster,.svc.cluster.local",
		},
		{
			name:           "empty_cluster_context",
			clusterCtx:     &domain.ClusterContext{},
			expectedResult: ",.svc,.svc.cluster,.svc.cluster.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetDefaultNoProxy(tt.clusterCtx)

			g.Expect(result).To(Equal(tt.expectedResult))
		})
	}
}
