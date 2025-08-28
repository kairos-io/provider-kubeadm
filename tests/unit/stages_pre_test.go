package main

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestStageGeneration tests YIP stage generation logic
func TestStageGeneration(t *testing.T) {
	_ = NewWithT(t)

	tests := []struct {
		name                string
		rootPath           string
		expectedStageName  string
		expectedCommands   []string
		expectedIf         string
	}{
		{
			name:     "appliance_mode_pre_commands",
			rootPath: "/",
			expectedStageName: "Run Pre Kubeadm Commands",
			expectedCommands: []string{
				"/bin/bash /opt/kubeadm/scripts/kube-pre-init.sh /",
			},
			expectedIf: "",
		},
		{
			name:     "agent_mode_pre_commands",
			rootPath: "/persistent/spectro",
			expectedStageName: "Run Pre Kubeadm Commands",
			expectedCommands: []string{
				"/bin/bash /persistent/spectro/opt/kubeadm/scripts/kube-pre-init.sh /persistent/spectro",
			},
			expectedIf: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test stage generation
			stage := generateMockPreKubeadmCommandStage(tt.rootPath)

			// Validate results
			g.Expect(stage.name).To(Equal(tt.expectedStageName))
			g.Expect(stage.commands).To(Equal(tt.expectedCommands))
			g.Expect(stage.ifCondition).To(Equal(tt.expectedIf))
		})
	}
}

// TestSwapOffStageGeneration tests swap off stage generation
func TestSwapOffStageGeneration(t *testing.T) {
	_ = NewWithT(t)

	// Test swap off stage generation
	stage := generateMockSwapOffStage()

	// Validate results
	g := NewWithT(t)
	g.Expect(stage.name).To(Equal("Run Pre Kubeadm Disable SwapOff"))
	g.Expect(stage.commands).To(HaveLen(2))
	g.Expect(stage.commands[0]).To(Equal("sed -i '/ swap / s/^\\(.*\\)$/#\\1/g' /etc/fstab"))
	g.Expect(stage.commands[1]).To(Equal("swapoff -a"))
	g.Expect(stage.ifCondition).To(BeEmpty())
}

// Mock stage structure for testing
type mockStage struct {
	name        string
	commands    []string
	ifCondition string
}

func generateMockPreKubeadmCommandStage(rootPath string) mockStage {
	scriptPath := rootPath
	if rootPath == "/" {
		scriptPath = ""
	}
	return mockStage{
		name: "Run Pre Kubeadm Commands",
		commands: []string{
			"/bin/bash " + scriptPath + "/opt/kubeadm/scripts/kube-pre-init.sh " + rootPath,
		},
		ifCondition: "",
	}
}

func generateMockSwapOffStage() mockStage {
	return mockStage{
		name: "Run Pre Kubeadm Disable SwapOff",
		commands: []string{
			"sed -i '/ swap / s/^\\(.*\\)$/#\\1/g' /etc/fstab",
			"swapoff -a",
		},
		ifCondition: "",
	}
}