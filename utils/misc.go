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
		return domain.DefaultRootPath
	}
	return rootpath
}

func IsKubeadmVersionGreaterThan131(rootPath string) (int, error) {
	currentVersion, err := getCurrentKubeadmVersion(rootPath)
	if err != nil {
		return 0, err
	}

	v1, err := version.ParseSemantic(currentVersion)
	if err != nil {
		return 0, err
	}

	return v1.Compare("v1.31.0")
}

func getCurrentKubeadmVersion(rootPath string) (string, error) {
	var kubeadmPath string
	if rootPath != domain.DefaultRootPath {
		kubeadmPath = fmt.Sprintf("%s/usr/bin/kubeadm", rootPath)
	} else {
		kubeadmPath = "kubeadm"
	}
	cmd := exec.Command(kubeadmPath, "version", "-o", "short")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting current kubeadm version: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}
