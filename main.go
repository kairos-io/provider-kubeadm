package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kyaml "sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
)

const (
	kubeletEnvConfigPath = "/etc/default"
	envPrefix            = "Environment="
	systemdDir           = "/etc/systemd/system/containerd.service.d"
	kubeletServiceName   = "kubelet"
	containerdEnv        = "http-proxy.conf"
	K8sNoProxy           = ".svc,.svc.cluster,.svc.cluster.local"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = kubeadmapiv3.AddToScheme(scheme)
}

var configurationPath = "/opt/kubeadm"
var helperScriptPath = "/opt/kubeadm/scripts"

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
}

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	var stages []yip.Stage
	var kubeadmConfig KubeadmConfig
	var importStage yip.Stage

	if cluster.Options != "" {
		userOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
		_ = json.Unmarshal(userOptions, &kubeadmConfig)
	}

	preStage := []yip.Stage{
		{
			Name: "Set proxy env",
			Files: []yip.File{
				{
					Path:        filepath.Join(kubeletEnvConfigPath, kubeletServiceName),
					Permissions: 0400,
					Content:     kubeletProxyEnv(kubeadmConfig.ClusterConfiguration, cluster.Env),
				},
				{
					Path:        filepath.Join(systemdDir, containerdEnv),
					Permissions: 0400,
					Content:     containerdProxyEnv(kubeadmConfig.ClusterConfiguration, cluster.Env),
				},
			},
		},
		{
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
			},
		},
	}

	if cluster.ImportLocalImages {
		if cluster.LocalImagesPath == "" {
			cluster.LocalImagesPath = "/opt/content/images"
		}

		importStage = yip.Stage{
			Commands: []string{
				fmt.Sprintf("chmod +x %s", filepath.Join(helperScriptPath, "import.sh")),
				fmt.Sprintf("/bin/sh %s %s > /var/log/import.log", filepath.Join(helperScriptPath, "import.sh"), cluster.LocalImagesPath),
			},
			If: fmt.Sprintf("[ -d %s ]", cluster.LocalImagesPath),
		}
		preStage = append(preStage, importStage)
	}

	// import k8s images
	preStage = append(preStage, yip.Stage{
		Name: "Run Load Kube Images",
		Commands: []string{
			fmt.Sprintf("chmod +x %s", filepath.Join(helperScriptPath, "import.sh")),
			fmt.Sprintf("/bin/sh %s /opt/kubeadm/kube-images > /var/log/import-kube-images.log", filepath.Join(helperScriptPath, "import.sh")),
		},
	})

	preStage = append(preStage, yip.Stage{
		If:   "[ ! -f /opt/sentinel_kubeadmversion ]",
		Name: "Create kubeadm sentinel version file",
		Commands: []string{
			"kubeadm version -o short > /opt/sentinel_kubeadmversion",
		},
	})

	cluster.ClusterToken = transformToken(cluster.ClusterToken)

	stages = append(stages, preStage...)

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
			Name: "Run Kubeadm Init",
			If:   "[ ! -f /opt/kubeadm.init ]",
			Commands: []string{
				fmt.Sprintf("bash %s", filepath.Join(helperScriptPath, "kube-init.sh")),
				"touch /opt/kubeadm.init",
			},
		},
		{
			Name: "Run Post Kubeadm Init",
			If:   "[ ! -f /opt/post-kubeadm.init ]",
			Commands: []string{
				fmt.Sprintf("bash %s", filepath.Join(helperScriptPath, "kube-post-init.sh")),
				"touch /opt/post-kubeadm.init",
			},
		},
		{
			Name: "Run Kubeadm Upgrade",
			Commands: []string{
				fmt.Sprintf("bash %s %s", filepath.Join(helperScriptPath, "kube-upgrade.sh"), cluster.Role),
			},
	}
		},
}

func getJoinYipStages(cluster clusterplugin.Cluster, joinCfg kubeadmapiv3.JoinConfiguration) []yip.Stage {
	kubeadmCfg := getJoinNodeConfiguration(cluster, joinCfg)
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
				fmt.Sprintf("bash %s %s", filepath.Join(helperScriptPath, "kube-join.sh"), cluster.Role),
				"touch /opt/kubeadm.join",
			},
		},
		{
			Name: "Run Kubeadm Upgrade",
			Commands: []string{
				fmt.Sprintf("bash %s %s", filepath.Join(helperScriptPath, "kube-upgrade.sh"), cluster.Role),
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
			TTL: &metav1.Duration{
				Duration: 0,
			},
		},
	}
	initCfg.CertificateKey = certificateKey
	initCfg.LocalAPIEndpoint = kubeadmapiv3.APIEndpoint{
		AdvertiseAddress: "0.0.0.0",
	}
	clusterCfg.APIServer.CertSANs = append(clusterCfg.APIServer.CertSANs, cluster.ControlPlaneHost)
	clusterCfg.ControlPlaneEndpoint = fmt.Sprintf("%s:6443", cluster.ControlPlaneHost)

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

func getCertificateKey(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

func transformToken(clusterToken string) string {
	hash := sha256.New()
	hash.Write([]byte(clusterToken))
	hashString := hex.EncodeToString(hash.Sum(nil))
	return fmt.Sprintf("%s.%s", hashString[len(hashString)-6:], hashString[:16])
}

func kubeletProxyEnv(clusterCfg kubeadmapiv3.ClusterConfiguration, proxyMap map[string]string) string {
	var proxy []string
	var noProxy string
	var isProxyConfigured bool

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTP_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]
	defaultNoProxy := getDefaultNoProxy(clusterCfg)

	if len(httpProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTP_PROXY=%s", httpProxy))
		isProxyConfigured = true
	}

	if len(httpsProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTPS_PROXY=%s", httpsProxy))
		isProxyConfigured = true
	}

	if isProxyConfigured {
		noProxy = defaultNoProxy
	}

	if len(userNoProxy) > 0 {
		noProxy = noProxy + "," + userNoProxy
	}

	if len(noProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("NO_PROXY=%s", noProxy))
	}

	return strings.Join(proxy, "\n")
}

func containerdProxyEnv(clusterCfg kubeadmapiv3.ClusterConfiguration, proxyMap map[string]string) string {
	var proxy []string
	var isProxyConfigured bool
	var noProxy string

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTPS_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]
	defaultNoProxy := getDefaultNoProxy(clusterCfg)

	if len(httpProxy) > 0 || len(httpsProxy) > 0 || len(userNoProxy) > 0 {
		proxy = append(proxy, "[Service]")
		isProxyConfigured = true
	}

	if len(httpProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"HTTP_PROXY=%s"+"\"", httpProxy))
	}

	if len(httpsProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"HTTPS_PROXY=%s"+"\"", httpsProxy))
	}

	if isProxyConfigured {
		noProxy = defaultNoProxy
	}

	if len(userNoProxy) > 0 {
		noProxy = noProxy + "," + userNoProxy
	}

	if len(noProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"NO_PROXY=%s"+"\"", noProxy))
	}

	return strings.Join(proxy, "\n")
}

func getDefaultNoProxy(clusterCfg kubeadmapiv3.ClusterConfiguration) string {
	var noProxy string

	cluster_cidr := clusterCfg.Networking.PodSubnet
	service_cidr := clusterCfg.Networking.ServiceSubnet

	if len(cluster_cidr) > 0 {
		noProxy = noProxy + "," + cluster_cidr
	}
	if len(service_cidr) > 0 {
		noProxy = noProxy + "," + service_cidr
	}

	noProxy = noProxy + "," + getNodeCIDR() + "," + K8sNoProxy
	return noProxy
}

func getNodeCIDR() string {
	addrs, _ := net.InterfaceAddrs()
	var result string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				result = addr.String()
				break
			}
		}
	}
	return result
}
