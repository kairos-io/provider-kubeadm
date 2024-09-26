package domain

import (
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

type KubeadmConfig struct {
	ClusterConfiguration kubeadmapiv3.ClusterConfiguration   `json:"clusterConfiguration,omitempty" yaml:"clusterConfiguration,omitempty"`
	InitConfiguration    kubeadmapiv3.InitConfiguration      `json:"initConfiguration,omitempty" yaml:"initConfiguration,omitempty"`
	JoinConfiguration    kubeadmapiv3.JoinConfiguration      `json:"joinConfiguration,omitempty" yaml:"joinConfiguration,omitempty"`
	KubeletConfiguration kubeletv1beta1.KubeletConfiguration `json:"kubeletConfiguration,omitempty" yaml:"kubeletConfiguration,omitempty"`
}
