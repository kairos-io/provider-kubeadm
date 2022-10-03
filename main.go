package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmapiv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

	"gopkg.in/yaml.v2"

	"github.com/c3os-io/c3os/sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = kubeadmapiv1.AddToScheme(scheme)
}

var configurationPath = "/opt/kubeadm"

func main() {
	plugin := clusterplugin.ClusterPlugin{
		Provider: clusterProvider,
	}

	if err := plugin.Run(); err != nil {
		logrus.Fatal(err)
	}
}

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {

	var stages []yip.Stage

	preStage := yip.Stage{
		Systemctl: yip.Systemctl{
			Enable: []string{"kubelet"},
			Start:  []string{"containerd"},
		},
		Commands: []string{
			"sysctl --system",
			"modprobe overlay",
			"modprobe br_netfilter",
		},
	}

	if !bootstraputil.IsValidBootstrapToken(cluster.ClusterToken) {
		logrus.Fatalf("the bootstrap token %s is not of the form %s", cluster.ClusterToken, bootstrapapi.BootstrapTokenPattern)
		return yip.YipConfig{}
	}

	stages = append(stages, preStage)

	if cluster.Role == clusterplugin.RoleInit {
		stages = append(stages, getInitYipStages(cluster)...)
	} else if (cluster.Role == clusterplugin.RoleControlPlane) || (cluster.Role == clusterplugin.RoleWorker) {
		stages = append(stages, getJoinYipStages(cluster)...)
	}

	cfg := yip.YipConfig{
		Name: "Kubeadm C3OS Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": stages,
		},
	}

	return cfg
}

func getInitYipStages(cluster clusterplugin.Cluster) []yip.Stage {
	kubeadmCfg := getInitNodeConfiguration(cluster)
	return []yip.Stage{
		{
			Name: "Init Kubeadm",
			If:   "[ ! -f /opt/kubeadm.init ]",
			Files: []yip.File{
				{
					Path:        filepath.Join(configurationPath, "kubeadm.yaml"),
					Permissions: 0640,
					Content:     kubeadmCfg,
				},
			},
			Commands: []string{
				fmt.Sprintf("kubeadm init --config %s --upload-certs", filepath.Join(configurationPath, "kubeadm.yaml")),
			},
		},
	}
}

func getJoinYipStages(cluster clusterplugin.Cluster) []yip.Stage {
	kubeadmCfg := getJoinNodeConfiguration(cluster)

	joinCmd := fmt.Sprintf("kubeadm join --config %s --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests", filepath.Join(configurationPath, "kubeadm.yaml"))
	if cluster.Role == clusterplugin.RoleControlPlane {
		joinCmd = joinCmd + " --control-plane"
	}

	return []yip.Stage{
		{
			Name: "Join Kubeadm",
			If:   "[ ! -f /opt/kubeadm.join ]",
			Files: []yip.File{
				{
					Path:        filepath.Join(configurationPath, "kubeadm.yaml"),
					Permissions: 0640,
					Content:     kubeadmCfg,
				},
			},
		},
		{
			Name: "Kubeadm join control plane",
			Commands: []string{
				joinCmd,
			},
		},
	}
}

func getInitNodeConfiguration(cluster clusterplugin.Cluster) string {
	certificateKey := getCertificateKey(cluster.ClusterToken)

	var initCfg kubeadmapiv1.InitConfiguration
	var clusterCfg kubeadmapiv1.ClusterConfiguration

	_ = yaml.Unmarshal([]byte(cluster.Options), &initCfg)
	_ = yaml.Unmarshal([]byte(cluster.Options), &clusterCfg)

	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(cluster.ClusterToken)

	initCfg.BootstrapTokens = []bootstraptokenv1.BootstrapToken{
		{
			Token: &bootstraptokenv1.BootstrapTokenString{
				ID:     substrs[1],
				Secret: substrs[2],
			},
		},
	}
	initCfg.CertificateKey = certificateKey

	initPrintr := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})

	_ = initPrintr.PrintObj(&clusterCfg, out)
	_ = initPrintr.PrintObj(&initCfg, out)

	return out.String()
}

func getJoinNodeConfiguration(cluster clusterplugin.Cluster) string {
	var joinCfg kubeadmapiv1.JoinConfiguration
	_ = yaml.Unmarshal([]byte(cluster.Options), &joinCfg)

	if joinCfg.Discovery.BootstrapToken == nil {
		joinCfg.Discovery.BootstrapToken = &kubeadmapiv1.BootstrapTokenDiscovery{
			Token:                    cluster.ClusterToken,
			APIServerEndpoint:        cluster.ControlPlaneHost,
			UnsafeSkipCAVerification: true,
		}
	}

	if cluster.Role == clusterplugin.RoleControlPlane {
		joinCfg.ControlPlane = &kubeadmapiv1.JoinControlPlane{
			CertificateKey: getCertificateKey(cluster.ClusterToken),
		}
	}

	joinPrinter := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})

	_ = joinPrinter.PrintObj(&joinCfg, out)

	return out.String()
}

func getCertificateKey(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
