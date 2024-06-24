package stages

import (
	"fmt"
	"path/filepath"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
)

const (
	helperScriptPath = "opt/kubeadm/scripts"
)

func GetPreKubeadmCommandStages() yip.Stage {
	return yip.Stage{
		Name: "Run Pre Kubeadm Commands",
		Systemctl: yip.Systemctl{
			Enable: []string{"kubelet"},
		},
		Commands: []string{
			"sysctl --system",
			"modprobe overlay",
			"modprobe br_netfilter",
			"systemctl daemon-reload",
			"systemctl restart spectro-containerd",
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
	if cluster.LocalImagesPath == "" {
		cluster.LocalImagesPath = filepath.Join(cluster.ClusterRootPath, "opt/content/images")
	}

	return yip.Stage{
		Name: "Run Import Local Images",
		Commands: []string{
			fmt.Sprintf("chmod +x %s", filepath.Join(cluster.ClusterRootPath, helperScriptPath, "import.sh")),
			fmt.Sprintf("/bin/sh %s %s > /var/log/import.log", filepath.Join(cluster.ClusterRootPath, helperScriptPath, "import.sh"), cluster.LocalImagesPath),
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

func GetPreKubeadmStoreKubeadmVersionStage(rootPath string) yip.Stage {
	return yip.Stage{
		If:   fmt.Sprintf("[ ! -f %s ]", filepath.Join(rootPath, "opt/sentinel_kubeadmversion")),
		Name: "Create kubeadm sentinel version file",
		Commands: []string{
			fmt.Sprintf("kubeadm version -o short > %s", filepath.Join(rootPath, "opt/sentinel_kubeadmversion")),
		},
	}
}
