package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/c3os-io/c3os/provider-kubeadm/pkg/config"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"text/template"

	"github.com/c3os-io/c3os/sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
)

var configurationPath = "/opt/kubeadm"

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {

	// TODO(kpiyush17) add cluster.Options handling, expecting the options to match KubeadmConfig spec

	var stages []yip.Stage

	preStage := yip.Stage{
		Systemctl: yip.Systemctl{
			Enable: []string{"kubelet"},
			Start:  []string{"containerd"},
		},
		Commands: []string{"sudo sysctl --system"},
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

func main() {
	plugin := clusterplugin.ClusterPlugin{
		Provider: clusterProvider,
	}

	if err := plugin.Run(); err != nil {
		logrus.Fatal(err)
	}
}

func getInitYipStages(cluster clusterplugin.Cluster) []yip.Stage {
	kubeadmCfg := getInitNodeConfiguration(cluster)
	return []yip.Stage{
		{
			Name: "Install Kubeadm Configuration Files",
			Files: []yip.File{
				{
					Path:        filepath.Join(configurationPath, "kubeadm.yaml"),
					Permissions: 0640,
					Content:     kubeadmCfg,
				},
			},
		},
		{
			Name: "Kubeadm init",
			Commands: []string{
				fmt.Sprintf("kubeadm init --config %s --upload-certs", filepath.Join(configurationPath, "kubeadm.yaml")),
			},
		},
		{
			Name: "Kubeadm export kubeconfig",
			Commands: []string{
				fmt.Sprintf("export KUBECONFIG=%s", "/etc/kubernetes/admin.conf"),
			},
		},
		{
			Name: "Post-Kubeadm apply Calico CNI",
			Commands: []string{
				fmt.Sprintf("kubectl apply -f %s -n kube-system", filepath.Join(configurationPath, "calico.yaml")),
			},
		},
	}
}

func getJoinYipStages(cluster clusterplugin.Cluster) []yip.Stage {
	kubeadmCfg := getJoinNodeConfiguration(cluster)

	joinCmd := fmt.Sprintf("kubeadm join --config %s", filepath.Join(configurationPath, "kubeadm.yaml"))
	if cluster.Role == clusterplugin.RoleControlPlane {
		joinCmd = joinCmd + " --control-plane"
	}

	return []yip.Stage{
		{
			Name: "Install Kubeadm Configuration Files",
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

	var initCfg bytes.Buffer
	_ = yaml.NewEncoder(&initCfg).Encode(config.InitConfiguration{
		TypeMeta: config.TypeMeta{
			Kind:       "InitConfiguration",
			APIVersion: "kubeadm.k8s.io/v1beta2",
		},
		BootstrapTokens: []config.BootstrapToken{
			{
				Token: cluster.ClusterToken,
			},
		},
		CertificateKey: certificateKey,
	})

	var clusterCfg bytes.Buffer
	_ = yaml.NewEncoder(&clusterCfg).Encode(config.ClusterConfiguration{
		TypeMeta: config.TypeMeta{
			Kind:       "ClusterConfiguration",
			APIVersion: "kubeadm.k8s.io/v1beta2",
		},
		ControlPlaneEndpoint: cluster.ControlPlaneHost,
		Networking: config.Networking{
			PodSubnet: "192.168.0.0/16", // default pod subnet, will be overridden with the value of user option
		},
	})

	var kubeletCfg bytes.Buffer
	_ = yaml.NewEncoder(&kubeletCfg).Encode(config.KubeletConfiguration{
		TypeMeta: config.TypeMeta{
			Kind:       "KubeletConfiguration",
			APIVersion: "kubelet.config.k8s.io/v1beta1",
		},
		CgroupDriver: "systemd",
	})

	initNodeCfg := config.Init{
		ClusterConfiguration: clusterCfg.String(),
		InitConfiguration:    initCfg.String(),
		KubeletConfiguration: kubeletCfg.String(),
	}

	return parseConfigTemplate(string(cluster.Role), config.InitConfigurationTemplate, initNodeCfg)
}

func getJoinNodeConfiguration(cluster clusterplugin.Cluster) string {
	certificateKey := getCertificateKey(cluster.ClusterToken)

	joinConfiguration := config.JoinConfiguration{
		TypeMeta: config.TypeMeta{
			Kind:       "JoinConfiguration",
			APIVersion: "kubeadm.k8s.io/v1beta2",
		},
		Discovery: config.Discovery{
			BootstrapToken: config.BootstrapTokenDiscovery{
				Token:                    cluster.ClusterToken,
				APIServerEndpoint:        cluster.ControlPlaneHost,
				UnsafeSkipCAVerification: true,
			},
		},
	}

	if cluster.Role == clusterplugin.RoleControlPlane {
		joinConfiguration.JoinControlPlane = &config.JoinControlPlane{
			CertificateKey: certificateKey,
		}
	}

	var joinCfg bytes.Buffer
	_ = yaml.NewEncoder(&joinCfg).Encode(joinConfiguration)

	var kubeletCfg bytes.Buffer
	_ = yaml.NewEncoder(&kubeletCfg).Encode(config.KubeletConfiguration{
		TypeMeta: config.TypeMeta{
			Kind:       "KubeletConfiguration",
			APIVersion: "kubeadm.k8s.io/v1beta2",
		},
		CgroupDriver: "systemd",
	})

	joinCpNodeCfg := config.Join{
		JoinConfiguration:    joinCfg.String(),
		KubeletConfiguration: kubeletCfg.String(),
	}

	return parseConfigTemplate(string(cluster.Role), config.JoinConfigurationTemplate, joinCpNodeCfg)
}

func parseConfigTemplate(role, tpl string, input interface{}) string {
	tm, _ := template.New(role).Parse(tpl)

	var out bytes.Buffer
	if err := tm.Execute(&out, input); err != nil {
		logrus.Fatal(err)
	}

	return out.String()
}

func getCertificateKey(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
