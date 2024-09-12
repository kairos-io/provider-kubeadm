package stages

import (
	"fmt"
	"path/filepath"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/utils"
	yip "github.com/mudler/yip/pkg/schema"
)

const (
	helperScriptPath = "opt/kubeadm/scripts"
)

func GetPreKubeadmCommandStages(rootPath string) yip.Stage {
	return yip.Stage{
		Name: "Run Pre Kubeadm Commands",
		Commands: []string{
			fmt.Sprintf("/bin/bash %s %s", filepath.Join(rootPath, helperScriptPath, "kube-pre-init.sh"), rootPath),
		},
	}
}

func GetPreKubeadmSwapOffDisableStage() yip.Stage {
	return yip.Stage{
		Name: "Run Pre Kubeadm Disable SwapOff",
		Commands: []string{
			"sed -i '/ swap / s/^\\(.*\\)$/#\\1/g' /etc/fstab",
			"swapoff -a",
		},
	}
}

func GetPreKubeadmImportLocalImageStage(cluster clusterplugin.Cluster) yip.Stage {
	clusterRootPath := utils.GetClusterRootPath(cluster)

	if cluster.LocalImagesPath == "" {
		cluster.LocalImagesPath = filepath.Join(clusterRootPath, "opt/content/images")
	}

	return yip.Stage{
		Name: "Run Import Local Images",
		Commands: []string{
			fmt.Sprintf("chmod +x %s", filepath.Join(clusterRootPath, helperScriptPath, "import.sh")),
			fmt.Sprintf("/bin/sh %s %s > /var/log/import.log", filepath.Join(clusterRootPath, helperScriptPath, "import.sh"), cluster.LocalImagesPath),
		},
		If: fmt.Sprintf("[ -d %s ]", cluster.LocalImagesPath),
	}
}

func GetPreKubeadmImportCoreK8sImageStage(rootPath string) yip.Stage {
	return yip.Stage{
		Name: "Run Load Kube Images",
		Commands: []string{
			fmt.Sprintf("chmod +x %s", filepath.Join(rootPath, helperScriptPath, "import.sh")),
			fmt.Sprintf("/bin/sh %s %s %s > /var/log/import-kube-images.log", filepath.Join(rootPath, helperScriptPath, "import.sh"), filepath.Join(rootPath, "opt/kube-images"), rootPath),
		},
	}
}
