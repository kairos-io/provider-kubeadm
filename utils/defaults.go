package utils

import (
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/utils/ptr"

	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"

	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func MutateClusterConfigBeta3Defaults(clusterCtx *domain.ClusterContext, clusterCfg *kubeadmapiv3.ClusterConfiguration) {
	clusterCfg.APIServer.CertSANs = appendIfNotPresent(clusterCfg.APIServer.CertSANs, clusterCtx.ControlPlaneHost)
	clusterCfg.ControlPlaneEndpoint = fmt.Sprintf("%s:6443", clusterCtx.ControlPlaneHost)

	if clusterCfg.ImageRepository == "" {
		clusterCfg.ImageRepository = kubeadmapiv3.DefaultImageRepository
	}
}

func MutateClusterConfigBeta4Defaults(clusterCtx *domain.ClusterContext, clusterCfg *kubeadmapiv4.ClusterConfiguration) {
	clusterCfg.APIServer.CertSANs = appendIfNotPresent(clusterCfg.APIServer.CertSANs, clusterCtx.ControlPlaneHost)
	clusterCfg.ControlPlaneEndpoint = fmt.Sprintf("%s:6443", clusterCtx.ControlPlaneHost)

	if clusterCfg.ImageRepository == "" {
		clusterCfg.ImageRepository = kubeadmapiv4.DefaultImageRepository
	}
}

func MutateKubeletDefaults(clusterCtx *domain.ClusterContext, kubeletCfg *kubeletv1beta1.KubeletConfiguration) {
	kubeletCfg.APIVersion = "kubelet.config.k8s.io/v1beta1"
	kubeletCfg.Kind = "KubeletConfiguration"

	if kubeletCfg.FeatureGates == nil {
		kubeletCfg.FeatureGates = map[string]bool{}
	}

	if kubeletCfg.StaticPodPath == "" {
		kubeletCfg.StaticPodPath = kubeadmapiv3.DefaultManifestsDir
	}

	var clusterDNS string
	dnsIP, err := constants.GetDNSIP(clusterCtx.ServiceCidr)
	if err != nil {
		clusterDNS = kubeadmapiv3.DefaultClusterDNSIP
	} else {
		clusterDNS = dnsIP.String()
	}

	if kubeletCfg.ClusterDNS == nil {
		kubeletCfg.ClusterDNS = []string{clusterDNS}
	}

	if kubeletCfg.ClusterDomain == "" {
		kubeletCfg.ClusterDomain = kubeadmapiv3.DefaultServiceDNSDomain
	}

	// Require all clients to the kubelet API to have client certs signed by the cluster CA
	if kubeletCfg.Authentication.X509.ClientCAFile == "" {
		kubeletCfg.Authentication.X509.ClientCAFile = filepath.Join(constants.KubernetesDir, constants.DefaultCertificateDir, constants.CACertName)
	}

	if kubeletCfg.Authentication.Anonymous.Enabled == nil {
		kubeletCfg.Authentication.Anonymous.Enabled = ptr.To(false)
	}

	if kubeletCfg.Authorization.Mode == "" {
		kubeletCfg.Authorization.Mode = kubeletv1beta1.KubeletAuthorizationModeWebhook
	}

	// Let clients using other authentication methods like ServiceAccount tokens also access the kubelet API
	if kubeletCfg.Authentication.Webhook.Enabled == nil {
		kubeletCfg.Authentication.Webhook.Enabled = ptr.To(true)
	}

	// Serve a /healthz webserver on localhost:10248 that kubeadm can talk to
	if kubeletCfg.HealthzBindAddress == "" {
		kubeletCfg.HealthzBindAddress = "127.0.0.1"
	}

	if kubeletCfg.HealthzPort == nil {
		kubeletCfg.HealthzPort = ptr.To(int32(constants.KubeletHealthzPort))
	}

	if kubeletCfg.ShutdownGracePeriod.Duration == 0 {
		kubeletCfg.ShutdownGracePeriod = metav1.Duration{
			Duration: 120 * time.Second,
		}
	}

	if kubeletCfg.ShutdownGracePeriodCriticalPods.Duration == 0 {
		kubeletCfg.ShutdownGracePeriodCriticalPods = metav1.Duration{
			Duration: 60 * time.Second,
		}
	}

	kubeletCfg.RotateCertificates = true

	if len(kubeletCfg.CgroupDriver) == 0 {
		kubeletCfg.CgroupDriver = constants.CgroupDriverSystemd
	}

	ok, _ := isServiceActive("systemd-resolved")
	if ok && kubeletCfg.ResolverConfig == nil {
		kubeletCfg.ResolverConfig = ptr.To("/run/systemd/resolve/resolv.conf")
	}
}

func appendIfNotPresent(slice []string, element string) []string {
	for _, e := range slice {
		if e == element {
			return slice
		}
	}
	return append(slice, element)
}
