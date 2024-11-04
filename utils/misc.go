package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/apimachinery/pkg/util/version"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
)

func GetClusterRootPath(cluster clusterplugin.Cluster) string {
	rootpath := cluster.ProviderOptions[domain.ClusterRootPath]
	if rootpath == "" {
		return "/"
	}
	return rootpath
}

func IsKubeadmVersionGreaterThan131() (int, error) {
	currentVersion, err := getCurrentKubeadmVersion()
	if err != nil {
		return 0, err
	}

	v1, err := version.ParseSemantic(currentVersion)
	if err != nil {
		return 0, err
	}

	return v1.Compare("v1.31.0")
}

func getCurrentKubeadmVersion() (string, error) {
	cmd := exec.Command("kubeadm", "version", "-o", "short")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting current kubeadm version: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}
