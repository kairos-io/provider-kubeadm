package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kyaml "sigs.k8s.io/yaml"

	"k8s.io/klog/v2"

	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"

	"k8s.io/utils/pointer"

	nodeutil "k8s.io/component-helpers/node/util"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

type kubeletFlagsOpts struct {
	nodeRegOpts              *kubeadmapiv3.NodeRegistrationOptions
	pauseImage               string
	registerTaintsUsingFlags bool
}

// WriteKubeletConfigToDisk writes the kubelet config object down to a file
func WriteKubeletConfigToDisk(clusterCfg *kubeadmapiv3.ClusterConfiguration, kubeletCfg *kubeletv1beta1.KubeletConfiguration) {
	mutateDefaults(clusterCfg, kubeletCfg)
	data, _ := kyaml.Marshal(kubeletCfg)
	writeConfigBytesToDisk(data, "/var/lib/kubelet")
}

func mutateDefaults(clusterCfg *kubeadmapiv3.ClusterConfiguration, kubeletCfg *kubeletv1beta1.KubeletConfiguration) {
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

func isServiceActive(name string) (bool, error) {
	initSystem, err := initsystem.GetInitSystem()
	if err != nil {
		return false, err
	}
	return initSystem.ServiceIsActive(name), nil
}

func writeConfigBytesToDisk(b []byte, kubeletDir string) {
	configFile := filepath.Join(kubeletDir, constants.KubeletConfigurationFileName)
	_ = os.WriteFile(configFile, b, 0644)
}

func RegenerateKubeletKubeadmArgsFile(clusterCfg *kubeadmapiv3.ClusterConfiguration, nodeReg *kubeadmapiv3.NodeRegistrationOptions, nodeRole string) string {
	var registerTaintsUsingFlags bool
	if nodeRole == "worker" {
		registerTaintsUsingFlags = true
	}

	flagOpts := kubeletFlagsOpts{
		nodeRegOpts:              nodeReg,
		pauseImage:               fmt.Sprintf("%s/%s:%s", clusterCfg.ImageRepository, "pause", constants.PauseVersion),
		registerTaintsUsingFlags: registerTaintsUsingFlags,
	}
	stringMap := buildKubeletArgMapCommon(flagOpts)
	argList := kubeadmutil.BuildArgumentListFromMap(stringMap, nodeReg.KubeletExtraArgs)
	return fmt.Sprintf("%s=%q\n", constants.KubeletEnvFileVariableName, strings.Join(argList, " "))
}

func buildKubeletArgMapCommon(opts kubeletFlagsOpts) map[string]string {
	kubeletFlags := map[string]string{}
	if opts.nodeRegOpts.CRISocket == "" {
		kubeletFlags["container-runtime-endpoint"] = constants.CRISocketContainerd
	} else {
		kubeletFlags["container-runtime-endpoint"] = opts.nodeRegOpts.CRISocket
	}

	if opts.pauseImage != "" {
		kubeletFlags["pod-infra-container-image"] = opts.pauseImage
	}

	if opts.registerTaintsUsingFlags && opts.nodeRegOpts.Taints != nil && len(opts.nodeRegOpts.Taints) > 0 {
		var taintStrs []string
		for _, taint := range opts.nodeRegOpts.Taints {
			taintStrs = append(taintStrs, taint.ToString())
		}

		kubeletFlags["register-with-taints"] = strings.Join(taintStrs, ",")
	}

	nodeName, hostname := getNodeNameAndHostname(opts.nodeRegOpts)
	if nodeName != hostname {
		klog.V(1).Infof("setting kubelet hostname-override to %q", nodeName)
		kubeletFlags["hostname-override"] = nodeName
	}

	return kubeletFlags
}

func getNodeNameAndHostname(cfg *kubeadmapiv3.NodeRegistrationOptions) (string, string) {
	hostname, _ := nodeutil.GetHostname("")
	nodeName := hostname
	if cfg.Name != "" {
		nodeName = cfg.Name
	}
	if name, ok := cfg.KubeletExtraArgs["hostname-override"]; ok {
		nodeName = name
	}
	return nodeName, hostname
}
