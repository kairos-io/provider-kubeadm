package stages

import (
	"fmt"
	"path/filepath"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
)

const (
	helperScriptPath = "/opt/kubeadm/scripts"
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
			"systemctl restart containerd",
			"mkdir -p /etc/kubernetes/manifests",
		},
	}
}

func GetPreKubeadmImportLocalImageStage(cluster clusterplugin.Cluster) yip.Stage {
	if cluster.LocalImagesPath == "" {
		cluster.LocalImagesPath = "/opt/content/images"
	}

	return yip.Stage{
		Name: "Run Import Local Images",
		Commands: []string{
			fmt.Sprintf("chmod +x %s", filepath.Join(helperScriptPath, "import.sh")),
			fmt.Sprintf("/bin/sh %s %s > /var/log/import.log", filepath.Join(helperScriptPath, "import.sh"), cluster.LocalImagesPath),
		},
		If: fmt.Sprintf("[ -d %s ]", cluster.LocalImagesPath),
	}
}

func GetPreKubeadmImportCoreK8sImageStage() yip.Stage {
	return yip.Stage{
		Name: "Run Load Kube Images",
		Commands: []string{
			fmt.Sprintf("chmod +x %s", filepath.Join(helperScriptPath, "import.sh")),
			fmt.Sprintf("/bin/sh %s /opt/kubeadm/kube-images > /var/log/import-kube-images.log", filepath.Join(helperScriptPath, "import.sh")),
		},
	}
}

func GetPreKubeadmStoreKubeadmVersionStage() yip.Stage {
	return yip.Stage{
		If:   "[ ! -f /opt/sentinel_kubeadmversion ]",
		Name: "Create kubeadm sentinel version file",
		Commands: []string{
			"kubeadm version -o short > /opt/sentinel_kubeadmversion",
		},
	}
}
