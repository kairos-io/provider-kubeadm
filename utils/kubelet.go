package utils

import (
	"fmt"
	"sort"
	"strings"

	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"

	"github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"

	nodeutil "k8s.io/component-helpers/node/util"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

type kubeletFlagsOpts struct {
	name                     string
	pauseImage               string
	criSocket                string
	taints                   []v1.Taint
	kubeletExtraArgs         map[string]string
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

func RegenerateKubeletKubeadmArgsUsingBeta3Config(clusterCfg *kubeadmapiv3.ClusterConfiguration, nodeReg *kubeadmapiv3.NodeRegistrationOptions, nodeRole string) string {
	var registerTaintsUsingFlags bool
	if nodeRole == "worker" {
		registerTaintsUsingFlags = true
	}

	flagOpts := kubeletFlagsOpts{
		name:                     nodeReg.Name,
		pauseImage:               fmt.Sprintf("%s/%s:%s", clusterCfg.ImageRepository, "pause", k8sVersionToPauseImage[clusterCfg.KubernetesVersion]),
		criSocket:                nodeReg.CRISocket,
		taints:                   nodeReg.Taints,
		kubeletExtraArgs:         nodeReg.KubeletExtraArgs,
		registerTaintsUsingFlags: registerTaintsUsingFlags,
	}
	stringMap := buildKubeletArgMapCommon(flagOpts)
	argList := buildArgumentListFromMap(stringMap, nodeReg.KubeletExtraArgs)
	return fmt.Sprintf("%s=%q", constants.KubeletEnvFileVariableName, strings.Join(argList, " "))
}

func RegenerateKubeletKubeadmArgsUsingBeta4Config(clusterCfg *kubeadmapiv4.ClusterConfiguration, nodeReg *kubeadmapiv4.NodeRegistrationOptions, nodeRole string) string {
	var registerTaintsUsingFlags bool
	if nodeRole == "worker" {
		registerTaintsUsingFlags = true
	}

	flagOpts := kubeletFlagsOpts{
		name:                     nodeReg.Name,
		pauseImage:               fmt.Sprintf("%s/%s:%s", clusterCfg.ImageRepository, "pause", k8sVersionToPauseImage[clusterCfg.KubernetesVersion]),
		criSocket:                nodeReg.CRISocket,
		taints:                   nodeReg.Taints,
		kubeletExtraArgs:         convertFromArgs(nodeReg.KubeletExtraArgs),
		registerTaintsUsingFlags: registerTaintsUsingFlags,
	}
	stringMap := buildKubeletArgMapCommon(flagOpts)
	argList := buildArgumentListFromMap(stringMap, convertFromArgs(nodeReg.KubeletExtraArgs))
	return fmt.Sprintf("%s=%q", constants.KubeletEnvFileVariableName, strings.Join(argList, " "))
}

func buildKubeletArgMapCommon(opts kubeletFlagsOpts) map[string]string {
	kubeletFlags := map[string]string{}
	if opts.criSocket == "" {
		kubeletFlags["container-runtime-endpoint"] = constants.CRISocketContainerd
	} else {
		kubeletFlags["container-runtime-endpoint"] = opts.criSocket
	}

	if opts.pauseImage != "" {
		kubeletFlags["pod-infra-container-image"] = opts.pauseImage
	}

	if opts.registerTaintsUsingFlags && opts.taints != nil && len(opts.taints) > 0 {
		var taintStrs []string
		for _, taint := range opts.taints {
			taintStrs = append(taintStrs, taint.ToString())
		}

		kubeletFlags["register-with-taints"] = strings.Join(taintStrs, ",")
	}

	nodeName, hostname := getNodeNameAndHostname(opts.name, opts.kubeletExtraArgs)
	if nodeName != hostname {
		logrus.Infof("setting kubelet hostname-override to %q", nodeName)
		kubeletFlags["hostname-override"] = nodeName
	}

	return kubeletFlags
}

func getNodeNameAndHostname(name string, kubeletExtraArgs map[string]string) (string, string) {
	hostname, _ := nodeutil.GetHostname("")
	nodeName := hostname
	if name != "" {
		nodeName = name
	}
	if name, ok := kubeletExtraArgs["hostname-override"]; ok {
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

func convertFromArgs(in []kubeadmapiv4.Arg) map[string]string {
	if in == nil {
		return nil
	}
	args := make(map[string]string, len(in))
	for _, arg := range in {
		args[arg.Name] = arg.Value
	}
	return args
}

func buildArgumentListFromMap(baseArguments map[string]string, overrideArguments map[string]string) []string {
	var command []string
	var keys []string

	argsMap := make(map[string]string)

	for k, v := range baseArguments {
		argsMap[k] = v
	}

	for k, v := range overrideArguments {
		argsMap[k] = v
	}

	for k := range argsMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		command = append(command, fmt.Sprintf("--%s=%s", k, argsMap[k]))
	}

	return command
}
