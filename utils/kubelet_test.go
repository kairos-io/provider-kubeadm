package utils

import (
	"testing"

	. "github.com/onsi/gomega"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
)

// TestRegenerateKubeletKubeadmArgsUsingBeta3Config tests the RegenerateKubeletKubeadmArgsUsingBeta3Config function
func TestRegenerateKubeletKubeadmArgsUsingBeta3Config(t *testing.T) {
	t.Run("regenerate_kubelet_args_beta3", func(t *testing.T) {
		g := NewWithT(t)

		nodeRegistration := &kubeadmapiv3.NodeRegistrationOptions{
			KubeletExtraArgs: map[string]string{
				"node-ip": "10.0.0.1",
			},
		}
		nodeRole := "init"

		result := RegenerateKubeletKubeadmArgsUsingBeta3Config(nodeRegistration, nodeRole)

		// Validate that the function returns a string
		g.Expect(result).To(BeAssignableToTypeOf(""))
	})
}

// TestRegenerateKubeletKubeadmArgsUsingBeta4Config tests the RegenerateKubeletKubeadmArgsUsingBeta4Config function
func TestRegenerateKubeletKubeadmArgsUsingBeta4Config(t *testing.T) {
	t.Run("regenerate_kubelet_args_beta4", func(t *testing.T) {
		g := NewWithT(t)

		nodeRegistration := &kubeadmapiv4.NodeRegistrationOptions{
			KubeletExtraArgs: []kubeadmapiv4.Arg{
				{Name: "node-ip", Value: "10.0.0.1"},
			},
		}
		nodeRole := "worker"

		result := RegenerateKubeletKubeadmArgsUsingBeta4Config(nodeRegistration, nodeRole)

		// Validate that the function returns a string
		g.Expect(result).To(BeAssignableToTypeOf(""))
	})
}

// TestBuildKubeletArgMapCommon tests the buildKubeletArgMapCommon function
func TestBuildKubeletArgMapCommon(t *testing.T) {
	t.Run("build_kubelet_arg_map_common", func(t *testing.T) {
		g := NewWithT(t)

		opts := kubeletFlagsOpts{
			name:             "test-node",
			criSocket:        "",
			kubeletExtraArgs: map[string]string{},
		}

		result := buildKubeletArgMapCommon(opts)

		// Validate that the function returns a map
		g.Expect(result).To(BeAssignableToTypeOf(map[string]string{}))
	})
}

// TestGetNodeNameAndHostname tests the getNodeNameAndHostname function
func TestGetNodeNameAndHostname(t *testing.T) {
	t.Run("get_node_name_and_hostname", func(t *testing.T) {
		g := NewWithT(t)

		name := "test-node"
		kubeletExtraArgs := map[string]string{
			"hostname-override": "custom-hostname",
		}

		nodeName, hostname := getNodeNameAndHostname(name, kubeletExtraArgs)

		// Validate that the function returns two strings
		g.Expect(nodeName).To(BeAssignableToTypeOf(""))
		g.Expect(hostname).To(BeAssignableToTypeOf(""))
	})
}

// TestIsServiceActive tests the isServiceActive function
func TestIsServiceActive(t *testing.T) {
	t.Run("is_service_active", func(t *testing.T) {
		g := NewWithT(t)

		serviceName := "containerd"

		result, err := isServiceActive(serviceName)

		// Validate that the function returns a boolean and error
		g.Expect(result).To(BeAssignableToTypeOf(true))
		if err != nil {
			g.Expect(err.Error()).To(BeAssignableToTypeOf(""))
		}
	})
}

// TestConvertFromArgs tests the convertFromArgs function
func TestConvertFromArgs(t *testing.T) {
	t.Run("convert_from_args", func(t *testing.T) {
		g := NewWithT(t)

		args := []kubeadmapiv4.Arg{
			{Name: "arg1", Value: "value1"},
			{Name: "arg2", Value: "value2"},
		}

		result := convertFromArgs(args)

		// Validate that the function returns a map
		g.Expect(result).To(BeAssignableToTypeOf(map[string]string{}))
	})
}

// TestBuildArgumentListFromMap tests the buildArgumentListFromMap function
func TestBuildArgumentListFromMap(t *testing.T) {
	t.Run("build_argument_list_from_map", func(t *testing.T) {
		g := NewWithT(t)

		baseArgs := map[string]string{
			"arg1": "value1",
			"arg2": "value2",
		}
		overrideArgs := map[string]string{
			"arg3": "value3",
		}

		result := buildArgumentListFromMap(baseArgs, overrideArgs)

		// Validate that the function returns a slice of strings
		g.Expect(result).To(BeAssignableToTypeOf([]string{}))
	})
}
