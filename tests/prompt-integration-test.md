📝 PROMPT: Write Comprehensive Static Integration Tests for Provider-Kubeadm

Context & Objective

Write static integration tests for the provider-kubeadm component that validate YIP stage generation across all supported code paths without requiring any live environment dependencies. The tests should provide complete input coverage for
all 48 core scenarios and verify that generated YIP configurations are correct.

Test Framework Requirements

Based on the Stylus testing patterns, use the following framework:
- Testing Library: Standard Go testing with Gomega assertions (g := NewWithT(t))
- File System Mocking: go-vfs for virtual filesystem operations
- Structure: Table-driven tests with comprehensive scenario coverage
- Validation: Deep YIP stage structure validation and content verification
- **Static Testing Approach**: No external dependencies, focus on logic validation
- **Helper Function Patterns**: Reusable validation functions for common test patterns

Package Structure and Imports

**Integration Tests Package Structure:**
```go
package main

import (
    "fmt"
    "strings"
    "testing"

    . "github.com/onsi/gomega"
    "github.com/kairos-io/kairos-sdk/clusterplugin"
)
```

**Unit Tests Package Structure:**
```go
package main

import (
    "testing"

    "github.com/kairos-io/kairos/provider-kubeadm/domain"
    "github.com/kairos-io/kairos/provider-kubeadm/stages"
    yip "github.com/mudler/yip/pkg/schema"
    . "github.com/onsi/gomega"
)
```

**Key Import Patterns:**
- Use `package main` for all test files to access unexported functions
- Import `github.com/onsi/gomega` for assertions
- Import `github.com/kairos-io/kairos-sdk/clusterplugin` for cluster types
- Import domain and stages packages for unit testing specific functions

Test Data Generalization Requirements

All test data must be generalized to avoid environment-specific or company-specific identifiers:

**Generalized Values to Use:**
- **Cluster Tokens**: Use `test-cluster-token-1234567890123456` instead of specific tokens
- **IP Addresses**: Use `10.0.0.1` for control plane hosts and node IPs
- **Domain Names**: Use `cluster-test.example.com` instead of company-specific domains
- **File Paths**: Use standard paths like `/opt/content/images` instead of company-specific paths
- **Network Ranges**: Use standard ranges like `192.168.0.0/16` for pod subnets

**Examples of Generalization:**
```yaml
# ❌ Avoid specific identifiers
cluster_token: 68413030465f917774b4d7c4
control_plane_host: 10.10.148.28
certSANs: ["cluster-68413030465f917774b4d7c4.proxy.dev.spectrocloud.com"]
local_images_path: /usr/local/spectrocloud/content/images

# ✅ Use generalized values
cluster_token: test-cluster-token-1234567890123456
control_plane_host: 10.0.0.1
certSANs: ["cluster-test.example.com"]
local_images_path: /opt/content/images
```

**Benefits of Generalization:**
- No company-specific or environment-specific data
- Portable across different testing environments
- Easy to understand and maintain
- Prevents accidental exposure of sensitive information
- Consistent test behavior across different setups

Static Testing Approach

Since external dependencies are not allowed, focus on testing the core logic:

**Logic Validation Patterns:**
- **Path Construction**: Test path building logic without filesystem access
- **Configuration Validation**: Test input validation and processing
- **Role-Based Behavior**: Test role-specific logic and decision making
- **Version Detection**: Test version comparison and API version logic
- **Error Handling**: Test error scenarios and edge cases

**Helper Function Patterns:**
```go
// Common validation pattern
func validatePathConstruction(t *testing.T, cluster clusterplugin.Cluster, expectedRootPath string) {
    g := NewWithT(t)
    
    // Test logic without external dependencies
    rootPath := getClusterRootPath(cluster)
    g.Expect(rootPath).To(Equal(expectedRootPath))
    
    // Test path construction
    expectedPaths := []string{
        fmt.Sprintf("%s/usr/bin/kubeadm", expectedRootPath),
        fmt.Sprintf("%s/opt/kubeadm/scripts/kube-init.sh", expectedRootPath),
    }
    
    for _, path := range expectedPaths {
        g.Expect(path).To(ContainSubstring(expectedRootPath))
    }
}
```

Coverage Verification Requirements

Include comprehensive coverage verification to ensure all code paths are tested:

**Coverage Matrix Validation:**
- **Core Scenarios**: 48 scenarios (2×3×2×2×2 matrix)
- **Integration Test Files**: 5 files covering different aspects
- **Unit Test Files**: 4 files covering main functions
- **Critical Code Paths**: 10+ identified paths
- **Function Coverage**: All main and stages package functions

**Coverage Verification Test:**
```go
func TestComprehensiveCoverageVerification(t *testing.T) {
    g := NewWithT(t)
    
    // Verify core matrix coverage: 2×3×2×2×2 = 48 scenarios
    expectedCoreScenarios := len(kubeadmVersions) * len(nodeRoles) * 
                           len(environmentModes) * len(proxyConfigs) * 
                           len(containerRuntimes)
    
    g.Expect(expectedCoreScenarios).To(Equal(48))
    
    // Verify test file coverage
    testFilesCoverage := map[string][]string{
        "version_paths_test.go": {"kubeadm version detection", "API handling"},
        "role_paths_test.go": {"role stage generation", "command generation"},
        // ... more coverage mappings
    }
}
```

Code Path Matrix to Cover

Create integration tests covering all combinations of:

| Dimension           | Values                                    | Description                                   |
  |---------------------|-------------------------------------------|-----------------------------------------------|
| Kubeadm Version     | v1beta3 (< 1.31.0), v1beta4 (≥ 1.31.0)    | Version detection determines API version path |
| Node Role           | init, controlplane, worker                | Role determines YIP stage sequences           |
| Environment Mode    | agent (STYLUS_ROOT), appliance (standard) | Path handling and service detection           |
| Proxy Configuration | with-proxy, without-proxy                 | Environment variable handling                 |
| Container Runtime   | spectro-containerd, standard-containerd   | Socket detection logic                        |

Total Test Scenarios: 2 × 3 × 2 × 2 × 2 = 48 scenarios

Test Implementation Structure

1. Main Test Function Pattern:

func TestProviderKubeadmYipStageGeneration(t *testing.T) {
g := NewWithT(t)

      tests := []struct {
          name                string
          kubeadmVersion      string  // "1.30.0" or "1.31.0"
          nodeRole           string  // "init", "controlplane", "worker"
          environmentMode    string  // "agent", "appliance"
          proxyConfig        bool    // true/false
          containerRuntime   string  // "spectro-containerd", "containerd"
          userOptions        string  // Custom kubeadm config YAML
          expectedStages     []expectedStage
          expectedFiles      []expectedFile
          expectedCommands   []expectedCommand
          wantErr           bool
      }{
          // All 48 test scenarios here
      }

      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              // Test implementation
          })
      }
}

2. Virtual Filesystem Setup Pattern:

// Setup virtual filesystem for each test scenario
func setupTestFileSystem(kubeadmVersion, environmentMode string) (afero.Fs, func(), error) {
var rootPath string
if environmentMode == "agent" {
rootPath = "/persistent/spectro"
} else {
rootPath = "/"
}

      fileSystem := map[string]interface{}{
          // Kubeadm binary with version
          filepath.Join(rootPath, "usr/bin/kubeadm"): createMockKubeadmBinary(kubeadmVersion),

          // Scripts directory
          filepath.Join(rootPath, "opt/kubeadm/scripts/kube-init.sh"):    mockScript("kube-init"),
          filepath.Join(rootPath, "opt/kubeadm/scripts/kube-join.sh"):    mockScript("kube-join"),
          filepath.Join(rootPath, "opt/kubeadm/scripts/kube-reset.sh"):   mockScript("kube-reset"),
          filepath.Join(rootPath, "opt/kubeadm/scripts/kube-pre-init.sh"): mockScript("kube-pre-init"),

          // Service detection files
          "/run/spectro/containerd/containerd.sock": []byte(""),  // For spectro-containerd
          "/run/containerd/containerd.sock": []byte(""),          // For standard containerd

          // Optional directories
          filepath.Join(rootPath, "opt/content/images/"): nil,    // For image import tests
      }

      vfsTest, cleanup, err := vfst.NewTestFS(fileSystem)
      return vfsTest, cleanup, err
}

3. Mock Cluster Input Generation:

func createClusterInput(scenario testScenario) clusterplugin.Cluster {
cluster := clusterplugin.Cluster{
Role: getClusterRole(scenario.nodeRole),
ControlPlaneHost: "10.0.0.1",  // Generalized IP address
ClusterToken: "test-cluster-token-1234567890123456",  // Generalized token
Options: scenario.userOptions,
}

      // Environment configuration
      if scenario.proxyConfig {
          cluster.Env = map[string]string{
              "HTTP_PROXY":  "http://proxy.example.com:8080",
              "HTTPS_PROXY": "https://proxy.example.com:8080",
              "NO_PROXY":    ".svc,.svc.cluster.local",
          }
      }

      // Provider options for service detection
      if scenario.containerRuntime == "spectro-containerd" {
          cluster.ProviderOptions = map[string]string{
              "spectro-containerd-service-name": "true",
          }
      }

      // Root path for agent mode
      if scenario.environmentMode == "agent" {
          cluster.ProviderOptions["cluster_root_path"] = "/persistent/spectro"
      }

      return cluster
}

4. YIP Stage Validation Functions:

func validateYipStages(t *testing.T, actualConfig yip.YipConfig, expectedScenario testScenario) {
g := NewWithT(t)

      // Validate overall structure
      g.Expect(actualConfig.Name).To(Equal("Kubeadm Kairos Cluster Provider"))
      g.Expect(actualConfig.Stages).To(HaveKey("boot.before"))

      stages := actualConfig.Stages["boot.before"]

      // Validate stage count based on role
      expectedStageCount := getExpectedStageCount(expectedScenario.nodeRole)
      g.Expect(len(stages)).To(Equal(expectedStageCount))

      // Validate pre-stages (common to all roles)
      validatePreStages(t, stages[0:5], expectedScenario)

      // Validate role-specific stages
      switch expectedScenario.nodeRole {
      case "init":
          validateInitStages(t, stages[5:], expectedScenario)
      case "controlplane", "worker":
          validateJoinStages(t, stages[5:], expectedScenario)
      }
}

func validatePreStages(t *testing.T, stages []yip.Stage, scenario testScenario) {
g := NewWithT(t)

      // Validate proxy stage
      proxyStage := findStageByName(stages, "Set proxy env")
      if scenario.proxyConfig {
          g.Expect(proxyStage).ToNot(BeNil())
          validateProxyStageFiles(t, proxyStage.Files, scenario)
      }

      // Validate pre-init commands stage
      preInitStage := findStageByName(stages, "Run Pre Kubeadm Commands")
      g.Expect(preInitStage).ToNot(BeNil())
      g.Expect(preInitStage.Commands).To(HaveLen(1))
      expectedCommand := fmt.Sprintf("/bin/bash %s/opt/kubeadm/scripts/kube-pre-init.sh %s",
          getRootPath(scenario.environmentMode), getRootPath(scenario.environmentMode))
      g.Expect(preInitStage.Commands[0]).To(Equal(expectedCommand))
}

func validateInitStages(t *testing.T, stages []yip.Stage, scenario testScenario) {
g := NewWithT(t)

      // Validate init config generation
      configStage := findStageByName(stages, "Generate Kubeadm Init Config File")
      g.Expect(configStage).ToNot(BeNil())
      g.Expect(configStage.Files).To(HaveLen(1))

      configFile := configStage.Files[0]
      expectedPath := fmt.Sprintf("%s/opt/kubeadm/kubeadm.yaml", getRootPath(scenario.environmentMode))
      g.Expect(configFile.Path).To(Equal(expectedPath))
      g.Expect(configFile.Permissions).To(Equal(uint32(0640)))

      // Validate kubeadm version-specific content
      if scenario.kubeadmVersion >= "1.31.0" {
          g.Expect(configFile.Content).To(ContainSubstring("apiVersion: kubeadm.k8s.io/v1beta4"))
      } else {
          g.Expect(configFile.Content).To(ContainSubstring("apiVersion: kubeadm.k8s.io/v1beta3"))
      }

      // Validate init execution stage
      initStage := findStageByName(stages, "Run Kubeadm Init")
      g.Expect(initStage).ToNot(BeNil())

      // Validate proxy-aware command construction
      if scenario.proxyConfig {
          expectedProxyArgs := "true http://proxy.example.com:8080 https://proxy.example.com:8080"
          g.Expect(initStage.Commands[0]).To(ContainSubstring(expectedProxyArgs))
      }
}

5. Key Test Scenarios to Implement:

// High-priority test cases
var criticalScenarios = []testScenario{
{
name: "v1beta3_init_agent_with_proxy_spectro_containerd",
kubeadmVersion: "1.30.0",
nodeRole: "init",
environmentMode: "agent",
proxyConfig: true,
containerRuntime: "spectro-containerd",
userOptions: `
  clusterConfiguration:
    networking:
      serviceSubnet: "10.96.0.0/16"
      podSubnet: "192.168.0.0/16"
  initConfiguration:
    nodeRegistration:
      kubeletExtraArgs:
        node-ip: "10.0.0.1"`,
},

      {
          name: "v1beta4_controlplane_appliance_no_proxy_standard_containerd",
          kubeadmVersion: "1.31.0",
          nodeRole: "controlplane",
          environmentMode: "appliance",
          proxyConfig: false,
          containerRuntime: "containerd",
          userOptions: `
clusterConfiguration:
networking:
serviceSubnet: "10.96.0.0/16"
podSubnet: "192.168.0.0/16"
joinConfiguration:
nodeRegistration:
kubeletExtraArgs:
- name: node-ip
value: "10.0.0.2"`,
},

      // Add all 48 scenarios...
}

6. Error Scenario Testing:

func TestProviderKubeadmErrorScenarios(t *testing.T) {
g := NewWithT(t)

      errorTests := []struct {
          name            string
          mockSetup       func() afero.Fs
          clusterInput    clusterplugin.Cluster
          expectedError   string
      }{
          {
              name: "kubeadm_binary_missing",
              mockSetup: func() afero.Fs {
                  // Setup filesystem without kubeadm binary
                  fileSystem := map[string]interface{}{
                      "/usr/bin/": nil, // Empty directory
                  }
                  vfsTest, _, _ := vfst.NewTestFS(fileSystem)
                  return vfsTest
              },
              expectedError: "failed to check if kubeadm version is greater than 131",
          },

          {
              name: "invalid_user_options_yaml",
              clusterInput: clusterplugin.Cluster{
                  Role: clusterplugin.RoleInit,
                  Options: "invalid: yaml: content: [",  // Invalid YAML
              },
              expectedError: "failed to parse config",
          },
      }

      for _, tt := range errorTests {
          t.Run(tt.name, func(t *testing.T) {
              // Test error handling
          })
      }
}

7. Test Execution Framework:

func TestMain(m *testing.M) {
// Setup test environment
setupGlobalTestEnvironment()

      // Run tests
      code := m.Run()

      // Cleanup
      cleanupGlobalTestEnvironment()

      os.Exit(code)
}

func setupGlobalTestEnvironment() {
// Initialize logging for tests
log.InitLogger("/tmp/provider-kubeadm-test.log")

      // Set test-friendly defaults
      os.Setenv("TEST_MODE", "true")
}

Validation Requirements

YIP Stage Structure Validation:

- Verify correct stage names and execution order
- Validate file paths are properly constructed for agent/appliance modes
- Confirm proxy environment variables are correctly injected
- Check script command arguments match expected patterns

Configuration Content Validation:

- Validate kubeadm YAML content matches expected API version (v1beta3/v1beta4)
- Verify kubelet configuration includes correct node-ip handling
- Check certificate and token handling for different roles
- Validate service folder logic (spectro-containerd vs containerd)

Path Resolution Validation:

- Confirm STYLUS_ROOT handling in agent mode
- Verify standard path usage in appliance mode
- Check script path construction consistency
- Validate binary path resolution logic

Expected Deliverables

1. Complete test file: provider_kubeadm_integration_test.go with all 48 scenarios
2. Helper functions: Mock setup, validation utilities, test data generators
3. Test data: Generalized kubeadm configurations for different scenarios (no environment-specific data)
4. Documentation: Test scenario coverage matrix and validation approach

Success Criteria

- 100% code path coverage across all identified scenarios
- Static execution with no external dependencies or live environments
- Comprehensive validation of generated YIP stage structures
- Clear test failure messages for debugging configuration issues
- Maintainable test structure that can be easily extended for new scenarios
- All test data generalized with no environment-specific or company-specific identifiers
- Portable test configurations that work across different environments
- Comprehensive coverage verification with 48+ test scenarios

This prompt provides the complete framework for writing static integration tests that thoroughly validate provider-kubeadm YIP stage generation across all supported deployment scenarios.

Input Yaml File to provider - /Users/rishi/work/src/provider-kubeadm/tests/data/k8s-1.30.11-input.yaml