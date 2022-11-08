package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v2"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"

	configutil "k8s.io/kubernetes/cmd/kubeadm/app/util/config"

	kyaml "sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

	"github.com/kairos-io/kairos/pkg/config"
	"github.com/kairos-io/kairos/sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
)

var (
	scheme = runtime.NewScheme()
)

var configScanDir = []string{"/oem", "/usr/local/cloud-config", "/run/initramfs/live"}

func init() {
	_ = kubeadmapiv3.AddToScheme(scheme)
}

var configurationPath = "/opt/kubeadm"
var defaultKubeconfigPath = "/etc/kubernetes/admin.conf"

type KubeadmConfig struct {
	ClusterConfiguration kubeadmapiv3.ClusterConfiguration `json:"clusterConfiguration,omitempty" yaml:"clusterConfiguration,omitempty"`
	InitConfiguration    kubeadmapiv3.InitConfiguration    `json:"initConfiguration,omitempty" yaml:"initConfiguration,omitempty"`
	JoinConfiguration    kubeadmapiv3.JoinConfiguration    `json:"joinConfiguration,omitempty" yaml:"joinConfiguration,omitempty"`
}

func main() {
	plugin := clusterplugin.ClusterPlugin{
		Provider: clusterProvider,
	}

	if err := plugin.Run(); err != nil {
		logrus.Fatal(err)
	}

	c, err := config.Scan(config.Directories(configScanDir...))
	if err != nil {
		logrus.Fatal(err)
	}

	var clusterConfig clusterplugin.Config

	if err := yaml.Unmarshal([]byte(c.String()), &clusterConfig); err != nil {
		logrus.Fatal(err)
	}

	if clusterConfig.Cluster.Role != clusterplugin.RoleInit {
		return
	}

	if err := updateControlPlaneEndpoint(clusterConfig.Cluster.ControlPlaneHost); err != nil {
		logrus.Fatal(err)
	}
}

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	var stages []yip.Stage
	var kubeadmConfig KubeadmConfig

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

	if cluster.Options != "" {
		userOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
		_ = json.Unmarshal(userOptions, &kubeadmConfig)
	}

	cluster.ClusterToken = transformToken(cluster.ClusterToken)

	stages = append(stages, preStage)

	if cluster.Role == clusterplugin.RoleInit {
		stages = append(stages, getInitYipStages(cluster, kubeadmConfig.InitConfiguration, kubeadmConfig.ClusterConfiguration)...)
	} else if (cluster.Role == clusterplugin.RoleControlPlane) || (cluster.Role == clusterplugin.RoleWorker) {
		stages = append(stages, getJoinYipStages(cluster, kubeadmConfig.JoinConfiguration)...)
	}

	cfg := yip.YipConfig{
		Name: "Kubeadm Kairos Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": stages,
		},
	}

	return cfg
}

func getInitYipStages(cluster clusterplugin.Cluster, initCfg kubeadmapiv3.InitConfiguration, clusterCfg kubeadmapiv3.ClusterConfiguration) []yip.Stage {
	kubeadmCfg := getInitNodeConfiguration(cluster, initCfg, clusterCfg)
	initCmd := fmt.Sprintf("kubeadm init --config %s --upload-certs --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests", filepath.Join(configurationPath, "kubeadm.yaml"))
	return []yip.Stage{
		{
			Name: "Generate Kubeadm Init Config File",
			Files: []yip.File{
				{
					Path:        filepath.Join(configurationPath, "kubeadm.yaml"),
					Permissions: 0640,
					Content:     kubeadmCfg,
				},
			},
		},
		{
			Name: "Kubeadm Init",
			If:   "[ ! -f /opt/kubeadm.init ]",
			Commands: []string{
				fmt.Sprintf("until $(%s > /dev/null ); do echo \"failed to apply kubeadm init, will retry in 10s\"; sleep 10; done;", initCmd),
				"touch /opt/kubeadm.init",
			},
		},
	}
}

func getJoinYipStages(cluster clusterplugin.Cluster, joinCfg kubeadmapiv3.JoinConfiguration) []yip.Stage {
	kubeadmCfg := getJoinNodeConfiguration(cluster, joinCfg)

	joinCmd := fmt.Sprintf("kubeadm join --config %s --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests", filepath.Join(configurationPath, "kubeadm.yaml"))

	return []yip.Stage{
		{
			Name: "Generate Kubeadm Join Config File",
			Files: []yip.File{
				{
					Path:        filepath.Join(configurationPath, "kubeadm.yaml"),
					Permissions: 0640,
					Content:     kubeadmCfg,
				},
			},
		},
		{
			Name: "Kubeadm Join",
			If:   "[ ! -f /opt/kubeadm.join ]",
			Commands: []string{
				fmt.Sprintf("until $(%s > /dev/null ); do echo \"failed to apply kubeadm join, will retry in 10s\"; sleep 10; done;", joinCmd),
				"touch /opt/kubeadm.join",
			},
		},
	}
}

func getInitNodeConfiguration(cluster clusterplugin.Cluster, initCfg kubeadmapiv3.InitConfiguration, clusterCfg kubeadmapiv3.ClusterConfiguration) string {
	certificateKey := getCertificateKey(cluster.ClusterToken)

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
	initCfg.LocalAPIEndpoint = kubeadmapiv3.APIEndpoint{
		AdvertiseAddress: "0.0.0.0",
	}
	clusterCfg.APIServer.CertSANs = append(clusterCfg.APIServer.CertSANs, cluster.ControlPlaneHost)

	initPrintr := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})

	_ = initPrintr.PrintObj(&clusterCfg, out)
	_ = initPrintr.PrintObj(&initCfg, out)

	return out.String()
}

func getJoinNodeConfiguration(cluster clusterplugin.Cluster, joinCfg kubeadmapiv3.JoinConfiguration) string {
	if joinCfg.Discovery.BootstrapToken == nil {
		joinCfg.Discovery.BootstrapToken = &kubeadmapiv3.BootstrapTokenDiscovery{
			Token:                    cluster.ClusterToken,
			APIServerEndpoint:        fmt.Sprintf("%s:6443", cluster.ControlPlaneHost),
			UnsafeSkipCAVerification: true,
		}
	}

	if cluster.Role == clusterplugin.RoleControlPlane {
		joinCfg.ControlPlane = &kubeadmapiv3.JoinControlPlane{
			CertificateKey: getCertificateKey(cluster.ClusterToken),
			LocalAPIEndpoint: kubeadmapiv3.APIEndpoint{
				AdvertiseAddress: "0.0.0.0",
			},
		}
	}

	joinPrinter := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})

	_ = joinPrinter.PrintObj(&joinCfg, out)

	return out.String()
}

func updateControlPlaneEndpoint(controlPlaneHost string) error {
	// create k8s client and update the config map
	clientConfig, err := clientcmd.BuildConfigFromFlags("", defaultKubeconfigPath)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	configMap, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(context.TODO(), constants.KubeadmConfigConfigMap, metav1.GetOptions{})
	if err != nil {
		return err
	}

	initcfg := &kubeadmapi.InitConfiguration{}

	// gets ClusterConfiguration from kubeadm-config
	clusterConfigurationData := configMap.Data[constants.ClusterConfigurationConfigMapKey]

	if err = runtime.DecodeInto(kubeadmscheme.Codecs.UniversalDecoder(), []byte(clusterConfigurationData), &initcfg.ClusterConfiguration); err != nil {
		return err
	}

	clusterCfg := initcfg.ClusterConfiguration
	clusterCfg.ControlPlaneEndpoint = controlPlaneHost

	clusterConfigurationYaml, err := configutil.MarshalKubeadmConfigObject(&clusterCfg)
	if err != nil {
		return err
	}

	err = apiclient.MutateConfigMap(clientset, metav1.ObjectMeta{
		Name:      constants.KubeadmConfigConfigMap,
		Namespace: metav1.NamespaceSystem,
	}, func(cm *v1.ConfigMap) error {
		cm.Data[constants.ClusterConfigurationConfigMapKey] = string(clusterConfigurationYaml)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func getCertificateKey(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

func transformToken(clusterToken string) string {
	hash := md5.New()
	hash.Write([]byte(clusterToken))
	hashString := hex.EncodeToString(hash.Sum(nil))
	return fmt.Sprintf("%s.%s", hashString[len(hashString)-6:], hashString[:16])
}
