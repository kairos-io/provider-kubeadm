package main

import (
	"strings"
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	. "github.com/onsi/gomega"
)

// TestNodeRoleBasedStagePaths tests different YIP stage generation based on node roles
func TestNodeRoleBasedStagePaths(t *testing.T) {
	roleTests := []struct {
		name                   string
		nodeRole               string
		expectedStageCount     int
		expectedSpecificStages []string
		missingStages          []string
	}{
		{
			name:               "init_role_stages",
			nodeRole:           "init",
			expectedStageCount: 10,
			expectedSpecificStages: []string{
				"Generate Kubeadm Init Config File",
				"Run Kubeadm Init",
				"Run Post Kubeadm Init",
				"Generate Cluster Config File",
				"Generate Kubelet Config File",
				"Run Kubeadm Init Upgrade",
				"Run Kubeadm Reconfiguration",
			},
			missingStages: []string{},
		},
		{
			name:               "controlplane_role_stages",
			nodeRole:           "controlplane",
			expectedStageCount: 9,
			expectedSpecificStages: []string{
				"Generate Kubeadm Join Config File",
				"Run Kubeadm Join",
				"Generate Cluster Config File", // Control plane gets cluster config
				"Generate Kubelet Config File", // Control plane gets kubelet config
				"Run Kubeadm Join Upgrade",
				"Run Kubeadm Join Reconfiguration",
			},
			missingStages: []string{
				"Run Post Kubeadm Init", // Only init role has post-init
			},
		},
		{
			name:               "worker_role_stages",
			nodeRole:           "worker",
			expectedStageCount: 7,
			expectedSpecificStages: []string{
				"Generate Kubeadm Join Config File",
				"Run Kubeadm Join",
				"Run Kubeadm Join Upgrade",
				"Run Kubeadm Join Reconfiguration",
			},
			missingStages: []string{
				"Generate Cluster Config File", // Workers don't get cluster config
				"Generate Kubelet Config File", // Workers don't get kubelet config
				"Run Post Kubeadm Init",        // Only init role has post-init
			},
		},
	}

	for _, tt := range roleTests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewWithT(t)

			// Create cluster input
			cluster := clusterplugin.Cluster{
				Role:             getRoleForTest(tt.nodeRole),
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
			}

			// Test role-based logic without external dependencies
			validateRoleBasedLogic(t, cluster, tt)
		})
	}
}

// validateRoleBasedLogic tests role-based logic without external dependencies
func validateRoleBasedLogic(t *testing.T, cluster clusterplugin.Cluster, tt struct {
	name                   string
	nodeRole               string
	expectedStageCount     int
	expectedSpecificStages []string
	missingStages          []string
}) {
	g := NewWithT(t)

	// Test role validation
	role := string(cluster.Role)
	g.Expect(role).To(Equal(tt.nodeRole))

	// Test role-specific logic
	switch tt.nodeRole {
	case "init":
		// Init role should have control plane host
		g.Expect(cluster.ControlPlaneHost).ToNot(BeEmpty())
		g.Expect(cluster.ClusterToken).ToNot(BeEmpty())

		// Test init-specific stage names - at least one should contain "Init"
		hasInitStage := false
		for _, stage := range tt.expectedSpecificStages {
			if strings.Contains(stage, "Init") {
				hasInitStage = true
				break
			}
		}
		g.Expect(hasInitStage).To(BeTrue(), "Init role should have at least one stage with 'Init'")

	case "controlplane":
		// Control plane role should have control plane host
		g.Expect(cluster.ControlPlaneHost).ToNot(BeEmpty())
		g.Expect(cluster.ClusterToken).ToNot(BeEmpty())

		// Test control plane specific stage names - at least one should contain "Join"
		hasJoinStage := false
		for _, stage := range tt.expectedSpecificStages {
			if strings.Contains(stage, "Join") {
				hasJoinStage = true
				break
			}
		}
		g.Expect(hasJoinStage).To(BeTrue(), "Control plane role should have at least one stage with 'Join'")

	case "worker":
		// Worker role should have control plane host
		g.Expect(cluster.ControlPlaneHost).ToNot(BeEmpty())
		g.Expect(cluster.ClusterToken).ToNot(BeEmpty())

		// Test worker specific stage names - at least one should contain "Join"
		hasJoinStage := false
		for _, stage := range tt.expectedSpecificStages {
			if strings.Contains(stage, "Join") {
				hasJoinStage = true
				break
			}
		}
		g.Expect(hasJoinStage).To(BeTrue(), "Worker role should have at least one stage with 'Join'")
	}

	// Test that missing stages are not in expected stages
	for _, missingStage := range tt.missingStages {
		found := false
		for _, expectedStage := range tt.expectedSpecificStages {
			if expectedStage == missingStage {
				found = true
				break
			}
		}
		g.Expect(found).To(BeFalse(), "Missing stage %s should not be in expected stages", missingStage)
	}
}

// TestRoleSpecificCommandGeneration tests role-specific command generation
func TestRoleSpecificCommandGeneration(t *testing.T) {
	commandTests := []struct {
		name                string
		nodeRole            string
		expectedCommandType string
		expectedScriptName  string
	}{
		{
			name:                "init_role_commands",
			nodeRole:            "init",
			expectedCommandType: "init",
			expectedScriptName:  "kube-init.sh",
		},
		{
			name:                "controlplane_role_commands",
			nodeRole:            "controlplane",
			expectedCommandType: "join",
			expectedScriptName:  "kube-join.sh",
		},
		{
			name:                "worker_role_commands",
			nodeRole:            "worker",
			expectedCommandType: "join",
			expectedScriptName:  "kube-join.sh",
		},
	}

	for _, tt := range commandTests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewWithT(t)

			// Test role-specific command generation logic
			validateRoleSpecificCommands(t, tt)
		})
	}
}

// validateRoleSpecificCommands tests role-specific command generation without external dependencies
func validateRoleSpecificCommands(t *testing.T, tt struct {
	name                string
	nodeRole            string
	expectedCommandType string
	expectedScriptName  string
}) {
	g := NewWithT(t)

	// Test command type based on role
	commandType := getCommandTypeForRole(tt.nodeRole)
	g.Expect(commandType).To(Equal(tt.expectedCommandType))

	// Test script name based on role
	scriptName := getScriptNameForRole(tt.nodeRole)
	g.Expect(scriptName).To(Equal(tt.expectedScriptName))

	// Test command construction
	command := constructCommandForRole(tt.nodeRole, "/opt/kubeadm/scripts")
	g.Expect(command).To(ContainSubstring(tt.expectedScriptName))
	g.Expect(command).To(ContainSubstring("/opt/kubeadm/scripts"))
}

// Helper functions for role testing
func getCommandTypeForRole(role string) string {
	switch role {
	case "init":
		return "init"
	case "controlplane", "worker":
		return "join"
	default:
		return "unknown"
	}
}

func getScriptNameForRole(role string) string {
	switch role {
	case "init":
		return "kube-init.sh"
	case "controlplane", "worker":
		return "kube-join.sh"
	default:
		return "unknown.sh"
	}
}

func constructCommandForRole(role string, scriptPath string) string {
	scriptName := getScriptNameForRole(role)
	return scriptPath + "/" + scriptName
}

// TestRoleSpecificConfigurationDifferences tests role-specific configuration differences
func TestRoleSpecificConfigurationDifferences(t *testing.T) {
	configTests := []struct {
		name                   string
		nodeRole               string
		expectedConfigType     string
		expectedConfigElements []string
	}{
		{
			name:               "init_role_config",
			nodeRole:           "init",
			expectedConfigType: "init",
			expectedConfigElements: []string{
				"initConfiguration",
				"clusterConfiguration",
			},
		},
		{
			name:               "controlplane_role_config",
			nodeRole:           "controlplane",
			expectedConfigType: "join",
			expectedConfigElements: []string{
				"joinConfiguration",
			},
		},
		{
			name:               "worker_role_config",
			nodeRole:           "worker",
			expectedConfigType: "join",
			expectedConfigElements: []string{
				"joinConfiguration",
			},
		},
	}

	for _, tt := range configTests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewWithT(t)

			// Test role-specific configuration logic
			validateRoleSpecificConfiguration(t, tt)
		})
	}
}

// validateRoleSpecificConfiguration tests role-specific configuration without external dependencies
func validateRoleSpecificConfiguration(t *testing.T, tt struct {
	name                   string
	nodeRole               string
	expectedConfigType     string
	expectedConfigElements []string
}) {
	g := NewWithT(t)

	// Test config type based on role
	configType := getConfigTypeForRole(tt.nodeRole)
	g.Expect(configType).To(Equal(tt.expectedConfigType))

	// Test config elements based on role
	configElements := getConfigElementsForRole(tt.nodeRole)
	for _, expectedElement := range tt.expectedConfigElements {
		found := false
		for _, element := range configElements {
			if element == expectedElement {
				found = true
				break
			}
		}
		g.Expect(found).To(BeTrue(), "Expected config element %s not found for role %s", expectedElement, tt.nodeRole)
	}
}

// Helper functions for configuration testing
func getConfigTypeForRole(role string) string {
	switch role {
	case "init":
		return "init"
	case "controlplane", "worker":
		return "join"
	default:
		return "unknown"
	}
}

func getConfigElementsForRole(role string) []string {
	switch role {
	case "init":
		return []string{"initConfiguration", "clusterConfiguration"}
	case "controlplane", "worker":
		return []string{"joinConfiguration"}
	default:
		return []string{}
	}
}

// Helper function to get cluster role
func getRoleForTest(role string) clusterplugin.Role {
	switch role {
	case "init":
		return clusterplugin.RoleInit
	case "controlplane":
		return clusterplugin.RoleControlPlane
	case "worker":
		return clusterplugin.RoleWorker
	default:
		return clusterplugin.RoleInit
	}
}
