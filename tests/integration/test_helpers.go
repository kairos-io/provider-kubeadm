package integration

import (
	"fmt"
	"strings"
	"testing"

	yip "github.com/mudler/yip/pkg/schema"
	"github.com/onsi/gomega"
)

// Helper functions
func getStageNames(stages []yip.Stage) []string {
	names := make([]string, len(stages))
	for i, stage := range stages {
		names[i] = stage.Name
	}
	return names
}

func getRootPath(envMode string) string {
	if envMode == "agent" {
		return "/persistent/spectro"
	}
	return "/"
}

// TestScenario represents a test configuration scenario
type TestScenario struct {
	name             string
	kubeadmVersion   string
	nodeRole         string
	environmentMode  string
	proxyConfig      string
	containerRuntime string
	localImages      string
	expectedStages   int
}

// TestValidationResult holds validation results for detailed reporting
type TestValidationResult struct {
	Scenario           TestScenario
	ActualStageCount   int
	ExpectedStageCount int
	MissingStages      []string
	UnexpectedStages   []string
	ValidationErrors   []string
}

// ComprehensiveYipValidator provides detailed validation of YIP configurations
type ComprehensiveYipValidator struct {
	t testing.TB
	g *gomega.WithT
}

func NewYipValidator(t testing.TB) *ComprehensiveYipValidator {
	return &ComprehensiveYipValidator{
		t: t,
		g: gomega.NewWithT(t),
	}
}

// ValidateComprehensive performs complete validation and returns detailed results
func (v *ComprehensiveYipValidator) ValidateComprehensive(actualConfig yip.YipConfig, scenario TestScenario) *TestValidationResult {
	result := &TestValidationResult{
		Scenario:         scenario,
		ValidationErrors: []string{},
	}

	// Basic structure validation
	v.g.Expect(actualConfig.Name).To(gomega.Equal("Kubeadm Kairos Cluster Provider"))
	v.g.Expect(actualConfig.Stages).To(gomega.HaveKey("boot.before"))

	stages := actualConfig.Stages["boot.before"]
	result.ActualStageCount = len(stages)
	result.ExpectedStageCount = scenario.expectedStages

	// Stage count validation
	if result.ActualStageCount != result.ExpectedStageCount {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Stage count mismatch: expected %d, got %d", result.ExpectedStageCount, result.ActualStageCount))
	}

	// Get expected stages for this scenario
	expectedStages := v.getExpectedStagesForScenario(scenario)
	actualStageNames := getStageNames(stages)

	// Find missing and unexpected stages
	result.MissingStages = v.findMissingStages(expectedStages, actualStageNames)
	result.UnexpectedStages = v.findUnexpectedStages(expectedStages, actualStageNames)

	// Detailed stage validation
	v.validateStageContent(stages, scenario, result)

	return result
}

func (v *ComprehensiveYipValidator) getExpectedStagesForScenario(scenario TestScenario) []string {
	expectedStages := []string{
		"Run Pre Kubeadm Commands",
		"Run Pre Kubeadm Disable SwapOff",
		"Run Load Kube Images",
	}

	// Add proxy stage if proxy is configured
	if scenario.proxyConfig != "" {
		expectedStages = append([]string{"Set proxy env"}, expectedStages...)
	}

	// Add local images stage if local images are enabled
	if scenario.localImages != "" {
		expectedStages = append(expectedStages, "Run Import Local Images")
	}

	// Add role-specific stages
	switch scenario.nodeRole {
	case "init":
		expectedStages = append(expectedStages,
			"Generate Kubeadm Init Config File",
			"Run Kubeadm Init",
			"Run Post Kubeadm Init",
			"Generate Cluster Config File",
			"Generate Kubelet Config File",
			"Run Kubeadm Init Upgrade",
			"Run Kubeadm Reconfiguration",
		)
	case "controlplane":
		expectedStages = append(expectedStages,
			"Generate Kubeadm Join Config File",
			"Run Kubeadm Join",
			"Generate Cluster Config File",
			"Generate Kubelet Config File",
			"Run Kubeadm Join Upgrade",
			"Run Kubeadm Join Reconfiguration",
		)
	case "worker":
		expectedStages = append(expectedStages,
			"Generate Kubeadm Join Config File",
			"Run Kubeadm Join",
			"Run Kubeadm Join Upgrade",
			"Run Kubeadm Join Reconfiguration",
		)
	}

	return expectedStages
}

func (v *ComprehensiveYipValidator) findMissingStages(expected, actual []string) []string {
	var missing []string
	for _, expectedStage := range expected {
		found := false
		for _, actualStage := range actual {
			if actualStage == expectedStage {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, expectedStage)
		}
	}
	return missing
}

func (v *ComprehensiveYipValidator) findUnexpectedStages(expected, actual []string) []string {
	var unexpected []string
	for _, actualStage := range actual {
		found := false
		for _, expectedStage := range expected {
			if actualStage == expectedStage {
				found = true
				break
			}
		}
		if !found {
			unexpected = append(unexpected, actualStage)
		}
	}
	return unexpected
}

func (v *ComprehensiveYipValidator) validateStageContent(stages []yip.Stage, scenario TestScenario, result *TestValidationResult) {
	for _, stage := range stages {
		switch stage.Name {
		case "Set proxy env":
			v.validateProxyStage(stage, scenario, result)
		case "Run Pre Kubeadm Commands":
			v.validatePreKubeadmCommandsStage(stage, scenario, result)
		case "Generate Kubeadm Init Config File", "Generate Kubeadm Join Config File":
			v.validateKubeadmConfigStage(stage, scenario, result)
		case "Run Kubeadm Init", "Run Kubeadm Join":
			v.validateKubeadmExecutionStage(stage, scenario, result)
		case "Run Kubeadm Init Upgrade", "Run Kubeadm Join Upgrade":
			v.validateKubeadmUpgradeStage(stage, scenario, result)
		case "Run Kubeadm Reconfiguration", "Run Kubeadm Join Reconfiguration":
			v.validateKubeadmReconfigureStage(stage, scenario, result)
		case "Run Import Local Images":
			v.validateLocalImagesStage(stage, scenario, result)
		}
	}
}

func (v *ComprehensiveYipValidator) validateProxyStage(stage yip.Stage, scenario TestScenario, result *TestValidationResult) {
	if scenario.proxyConfig == "" {
		result.ValidationErrors = append(result.ValidationErrors, "Proxy stage found but proxy not configured")
		return
	}

	if len(stage.Files) != 2 {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Proxy stage should have 2 files, got %d", len(stage.Files)))
		return
	}

	var kubeletFile, containerdFile *yip.File
	for i := range stage.Files {
		if strings.Contains(stage.Files[i].Path, "/etc/default/kubelet") {
			kubeletFile = &stage.Files[i]
		} else if strings.Contains(stage.Files[i].Path, "http-proxy.conf") {
			containerdFile = &stage.Files[i]
		}
	}

	// Validate kubelet proxy file
	if kubeletFile == nil {
		result.ValidationErrors = append(result.ValidationErrors, "Missing kubelet proxy file")
	} else {
		if kubeletFile.Permissions != 0400 {
			result.ValidationErrors = append(result.ValidationErrors,
				fmt.Sprintf("Kubelet proxy file permissions should be 0400, got %o", kubeletFile.Permissions))
		}
		if !strings.Contains(kubeletFile.Content, "HTTP_PROXY=http://proxy.example.com:8080") {
			result.ValidationErrors = append(result.ValidationErrors, "Missing HTTP_PROXY in kubelet file")
		}
	}

	// Validate containerd proxy file
	if containerdFile == nil {
		result.ValidationErrors = append(result.ValidationErrors, "Missing containerd proxy file")
	} else {
		if containerdFile.Permissions != 0400 {
			result.ValidationErrors = append(result.ValidationErrors,
				fmt.Sprintf("Containerd proxy file permissions should be 0400, got %o", containerdFile.Permissions))
		}
		if !strings.Contains(containerdFile.Content, "[Service]") {
			result.ValidationErrors = append(result.ValidationErrors, "Missing [Service] section in containerd proxy file")
		}

		// Validate service folder name in path
		expectedServiceFolder := "containerd"
		if scenario.containerRuntime == "spectro-containerd" {
			expectedServiceFolder = "spectro-containerd"
		}
		if !strings.Contains(containerdFile.Path, expectedServiceFolder+".service.d") {
			result.ValidationErrors = append(result.ValidationErrors,
				fmt.Sprintf("Containerd proxy file path should contain %s.service.d", expectedServiceFolder))
		}
	}
}

func (v *ComprehensiveYipValidator) validatePreKubeadmCommandsStage(stage yip.Stage, scenario TestScenario, result *TestValidationResult) {
	if len(stage.Commands) != 1 {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Pre-kubeadm commands stage should have 1 command, got %d", len(stage.Commands)))
		return
	}

	expectedRootPath := getRootPath(scenario.environmentMode)
	expectedCommand := fmt.Sprintf("/bin/bash %s/opt/kubeadm/scripts/kube-pre-init.sh %s", expectedRootPath, expectedRootPath)

	if stage.Commands[0] != expectedCommand {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Pre-kubeadm command mismatch: expected %s, got %s", expectedCommand, stage.Commands[0]))
	}
}

func (v *ComprehensiveYipValidator) validateKubeadmConfigStage(stage yip.Stage, scenario TestScenario, result *TestValidationResult) {
	if len(stage.Files) != 1 {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Kubeadm config stage should have 1 file, got %d", len(stage.Files)))
		return
	}

	configFile := stage.Files[0]
	expectedRootPath := getRootPath(scenario.environmentMode)
	expectedPath := fmt.Sprintf("%s/opt/kubeadm/kubeadm.yaml", expectedRootPath)

	if configFile.Path != expectedPath {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Config file path mismatch: expected %s, got %s", expectedPath, configFile.Path))
	}

	if configFile.Permissions != 0640 {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Config file permissions should be 0640, got %o", configFile.Permissions))
	}

	// Validate API version in content
	expectedAPIVersion := "v1beta3"
	if strings.Contains(scenario.kubeadmVersion, "1.31") || strings.Contains(scenario.kubeadmVersion, "1.32") {
		expectedAPIVersion = "v1beta4"
	}

	if !strings.Contains(configFile.Content, fmt.Sprintf("apiVersion: kubeadm.k8s.io/%s", expectedAPIVersion)) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Config file should contain API version %s", expectedAPIVersion))
	}
}

func (v *ComprehensiveYipValidator) validateKubeadmExecutionStage(stage yip.Stage, scenario TestScenario, result *TestValidationResult) {
	if len(stage.Commands) == 0 {
		result.ValidationErrors = append(result.ValidationErrors, "Kubeadm execution stage should have commands")
		return
	}

	command := stage.Commands[0]
	expectedRootPath := getRootPath(scenario.environmentMode)

	// Validate script path in command
	expectedScriptPath := fmt.Sprintf("%s/opt/kubeadm/scripts/kube-", expectedRootPath)
	if !strings.Contains(command, expectedScriptPath) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Command should contain script path %s", expectedScriptPath))
	}

	// Validate proxy parameters if proxy is configured
	if scenario.proxyConfig != "" {
		if !strings.Contains(command, "true") || !strings.Contains(command, "proxy.example.com") {
			result.ValidationErrors = append(result.ValidationErrors, "Missing proxy parameters in kubeadm command")
		}
	}

	// Validate conditional execution (If clause)
	expectedCondition := fmt.Sprintf("[ ! -f %s/opt/kubeadm", expectedRootPath)
	if stage.If != "" && !strings.Contains(stage.If, expectedCondition) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Stage condition should check for completion file in %s/opt/kubeadm", expectedRootPath))
	}
}

func (v *ComprehensiveYipValidator) validateKubeadmUpgradeStage(stage yip.Stage, scenario TestScenario, result *TestValidationResult) {
	if len(stage.Commands) == 0 {
		result.ValidationErrors = append(result.ValidationErrors, "Kubeadm upgrade stage should have commands")
		return
	}

	command := stage.Commands[0]
	expectedRootPath := getRootPath(scenario.environmentMode)

	// Validate script path
	expectedScriptPath := fmt.Sprintf("%s/opt/kubeadm/scripts/kube-upgrade.sh", expectedRootPath)
	if !strings.Contains(command, expectedScriptPath) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Upgrade command should contain script path %s", expectedScriptPath))
	}

	// Validate node role parameter
	if !strings.Contains(command, scenario.nodeRole) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Upgrade command should contain node role %s", scenario.nodeRole))
	}
}

func (v *ComprehensiveYipValidator) validateKubeadmReconfigureStage(stage yip.Stage, scenario TestScenario, result *TestValidationResult) {
	if len(stage.Commands) == 0 {
		result.ValidationErrors = append(result.ValidationErrors, "Kubeadm reconfigure stage should have commands")
		return
	}

	command := stage.Commands[0]
	expectedRootPath := getRootPath(scenario.environmentMode)

	// Validate script path
	expectedScriptPath := fmt.Sprintf("%s/opt/kubeadm/scripts/kube-reconfigure.sh", expectedRootPath)
	if !strings.Contains(command, expectedScriptPath) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Reconfigure command should contain script path %s", expectedScriptPath))
	}

	// Validate parameters count
	paramCount := len(strings.Fields(command))
	expectedParamCount := 6 // Base parameters
	if scenario.proxyConfig != "" {
		expectedParamCount = 9 // Additional proxy parameters
	}

	if paramCount < expectedParamCount {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Reconfigure command should have at least %d parameters, got %d", expectedParamCount, paramCount))
	}
}

func (v *ComprehensiveYipValidator) validateLocalImagesStage(stage yip.Stage, scenario TestScenario, result *TestValidationResult) {
	if scenario.localImages == "" {
		result.ValidationErrors = append(result.ValidationErrors, "Local images stage found but local images not configured")
		return
	}

	if len(stage.Commands) == 0 {
		result.ValidationErrors = append(result.ValidationErrors, "Local images stage should have commands")
		return
	}

	command := stage.Commands[0]
	expectedRootPath := getRootPath(scenario.environmentMode)

	// Validate script path
	expectedScriptPath := fmt.Sprintf("%s/opt/kubeadm/scripts/import.sh", expectedRootPath)
	if !strings.Contains(command, expectedScriptPath) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Import command should contain script path %s", expectedScriptPath))
	}

	// Validate conditional execution based on local images path
	expectedImagesPath := fmt.Sprintf("%s/opt/content/images", expectedRootPath)
	if stage.If != "" && !strings.Contains(stage.If, expectedImagesPath) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Local images stage condition should check for %s", expectedImagesPath))
	}
}

// PrintValidationSummary prints a comprehensive validation summary
func (v *ComprehensiveYipValidator) PrintValidationSummary(results []*TestValidationResult) {
	fmt.Printf("\n=== Provider-Kubeadm Integration Test Validation Summary ===\n\n")

	passedTests := 0
	failedTests := 0

	for _, result := range results {
		if len(result.ValidationErrors) == 0 && len(result.MissingStages) == 0 && len(result.UnexpectedStages) == 0 {
			passedTests++
			fmt.Printf("✅ %s - PASSED\n", result.Scenario.name)
		} else {
			failedTests++
			fmt.Printf("❌ %s - FAILED\n", result.Scenario.name)

			if result.ActualStageCount != result.ExpectedStageCount {
				fmt.Printf("   Stage Count: Expected %d, Got %d\n", result.ExpectedStageCount, result.ActualStageCount)
			}

			if len(result.MissingStages) > 0 {
				fmt.Printf("   Missing Stages: %s\n", strings.Join(result.MissingStages, ", "))
			}

			if len(result.UnexpectedStages) > 0 {
				fmt.Printf("   Unexpected Stages: %s\n", strings.Join(result.UnexpectedStages, ", "))
			}

			if len(result.ValidationErrors) > 0 {
				fmt.Printf("   Validation Errors:\n")
				for _, err := range result.ValidationErrors {
					fmt.Printf("     - %s\n", err)
				}
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total Tests: %d\n", len(results))
	fmt.Printf("Passed: %d\n", passedTests)
	fmt.Printf("Failed: %d\n", failedTests)
	fmt.Printf("Coverage: %.1f%%\n", float64(passedTests)/float64(len(results))*100)

	if failedTests == 0 {
		fmt.Printf("🎉 All tests passed! Provider-kubeadm has 100%% YIP stage generation coverage.\n")
	}
}
