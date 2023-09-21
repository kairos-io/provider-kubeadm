package utils

import (
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"

	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/utils/pointer"
)

func MutateClusterConfigDefaults(clusterCfg *kubeadmapiv3.ClusterConfiguration) {
	if clusterCfg.ImageRepository == "" {
		clusterCfg.ImageRepository = kubeadmapiv3.DefaultImageRepository
	}
}

func MutateKubeletDefaults(clusterCfg *kubeadmapiv3.ClusterConfiguration, kubeletCfg *kubeletv1beta1.KubeletConfiguration) {
	kubeletCfg.APIVersion = "kubelet.config.k8s.io/v1beta1"
	kubeletCfg.Kind = "KubeletConfiguration"

	if kubeletCfg.FeatureGates == nil {
		kubeletCfg.FeatureGates = map[string]bool{}
	}

	if kubeletCfg.StaticPodPath == "" {
		kubeletCfg.StaticPodPath = kubeadmapiv3.DefaultManifestsDir
	}

	var clusterDNS string
	dnsIP, err := constants.GetDNSIP(clusterCfg.Networking.ServiceSubnet)
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
		kubeletCfg.Authentication.Anonymous.Enabled = pointer.Bool(false)
	}

	// On every client request to the kubelet API, execute a webhook (SubjectAccessReview request) to the API server
	// and ask it whether the client is authorized to access the kubelet API
	if kubeletCfg.Authorization.Mode == "" {
		kubeletCfg.Authorization.Mode = kubeletv1beta1.KubeletAuthorizationModeWebhook
	}

	// Let clients using other authentication methods like ServiceAccount tokens also access the kubelet API
	if kubeletCfg.Authentication.Webhook.Enabled == nil {
		kubeletCfg.Authentication.Webhook.Enabled = pointer.Bool(true)
	}

	// Serve a /healthz webserver on localhost:10248 that kubeadm can talk to
	if kubeletCfg.HealthzBindAddress == "" {
		kubeletCfg.HealthzBindAddress = "127.0.0.1"
	}

	if kubeletCfg.HealthzPort == nil {
		kubeletCfg.HealthzPort = pointer.Int32(constants.KubeletHealthzPort)
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

	// We cannot show a warning for RotateCertificates==false and we must hardcode it to true.
	// There is no way to determine if the user has set this or not, given the field is a non-pointer.
	kubeletCfg.RotateCertificates = true

	if len(kubeletCfg.CgroupDriver) == 0 {
		kubeletCfg.CgroupDriver = constants.CgroupDriverSystemd
	}

	ok, _ := isServiceActive("systemd-resolved")
	if ok && kubeletCfg.ResolverConfig == nil {
		kubeletCfg.ResolverConfig = pointer.String("/run/systemd/resolve/resolv.conf")
	}
}
