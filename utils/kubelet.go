package utils

import (
	"fmt"
	"strings"

	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"

	"k8s.io/klog/v2"

	nodeutil "k8s.io/component-helpers/node/util"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

type kubeletFlagsOpts struct {
	nodeRegOpts              *kubeadmapiv3.NodeRegistrationOptions
	pauseImage               string
	registerTaintsUsingFlags bool
}

var k8sVersionToPauseImage = map[string]string{
	"v1.22.15": "3.5",
	"v1.23.12": "3.6",
	"v1.24.6":  "3.7",
	"v1.25.2":  "3.8",
	"v1.25.9":  "3.8",
	"v1.25.13": "3.8",
	"v1.26.4":  "3.8",
	"v1.26.8":  "3.8",
	"v1.27.2":  "3.9",
	"v1.27.5":  "3.9",
}

func RegenerateKubeletKubeadmArgsFile(clusterCfg *kubeadmapiv3.ClusterConfiguration, nodeReg *kubeadmapiv3.NodeRegistrationOptions, nodeRole string) string {
	var registerTaintsUsingFlags bool
	if nodeRole == "worker" {
		registerTaintsUsingFlags = true
	}

	flagOpts := kubeletFlagsOpts{
		nodeRegOpts:              nodeReg,
		pauseImage:               fmt.Sprintf("%s/%s:%s", clusterCfg.ImageRepository, "pause", k8sVersionToPauseImage[clusterCfg.KubernetesVersion]),
		registerTaintsUsingFlags: registerTaintsUsingFlags,
	}
	stringMap := buildKubeletArgMapCommon(flagOpts)
	argList := kubeadmutil.BuildArgumentListFromMap(stringMap, nodeReg.KubeletExtraArgs)
	return fmt.Sprintf("%s=%q", constants.KubeletEnvFileVariableName, strings.Join(argList, " "))
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

func isServiceActive(name string) (bool, error) {
	initSystem, err := initsystem.GetInitSystem()
	if err != nil {
		return false, err
	}
	return initSystem.ServiceIsActive(name), nil
}
