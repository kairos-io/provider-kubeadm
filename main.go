package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	"github.com/kairos-io/kairos/provider-kubeadm/stages"
	"github.com/kairos-io/kairos/provider-kubeadm/utils"
	"gopkg.in/yaml.v3"
	kyaml "sigs.k8s.io/yaml"

	"github.com/kairos-io/kairos-sdk/bus"
	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/mudler/go-pluggable"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
)

func main() {
	plugin := clusterplugin.ClusterPlugin{
		Provider: clusterProvider,
	}

	if err := plugin.Run(
		pluggable.FactoryPlugin{
			EventType:     clusterplugin.EventClusterReset,
			PluginHandler: handleClusterReset,
		},
	); err != nil {
		logrus.Fatal(err)
	}
}

func handleClusterReset(event *pluggable.Event) pluggable.EventResponse {
	var payload bus.EventPayload
	var config clusterplugin.Config
	var response pluggable.EventResponse

	// parse the boot payload
	if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
		response.Error = fmt.Sprintf("failed to parse boot event: %s", err.Error())
		return response
	}

	// parse config from boot payload
	if err := yaml.Unmarshal([]byte(payload.Config), &config); err != nil {
		response.Error = fmt.Sprintf("failed to parse config from boot event: %s", err.Error())
		return response
	}

	if config.Cluster == nil {
		return response
	}

	clusterRootPath := utils.GetClusterRootPath(*config.Cluster)
	cmd := exec.Command("/bin/sh", "-c", filepath.Join(clusterRootPath, "/opt/kubeadm/scripts", "kube-reset.sh"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		response.Error = fmt.Sprintf("failed to reset cluster: %s", string(output))
	}

	return response
}

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	var finalStages []yip.Stage
	var kubeadmConfig domain.KubeadmConfig

	if cluster.Options != "" {
		userOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
		_ = json.Unmarshal(userOptions, &kubeadmConfig)
	}

	clusterRootPath := utils.GetClusterRootPath(cluster)

	preStage := []yip.Stage{
		stages.GetPreKubeadmProxyStage(kubeadmConfig, cluster),
		stages.GetPreKubeadmCommandStages(clusterRootPath),
		stages.GetPreKubeadmSwapOffDisableStage(),
		stages.GetPreKubeadmImportCoreK8sImageStage(clusterRootPath),
	}

	if cluster.ImportLocalImages {
		preStage = append(preStage, stages.GetPreKubeadmImportLocalImageStage(cluster))
	}

	cluster.ClusterToken = utils.TransformToken(cluster.ClusterToken)

	finalStages = append(finalStages, preStage...)

	if cluster.Role == clusterplugin.RoleInit {
		finalStages = append(finalStages, stages.GetInitYipStages(cluster, kubeadmConfig.InitConfiguration, kubeadmConfig.ClusterConfiguration, kubeadmConfig.KubeletConfiguration)...)
	} else if (cluster.Role == clusterplugin.RoleControlPlane) || (cluster.Role == clusterplugin.RoleWorker) {
		finalStages = append(finalStages, stages.GetJoinYipStages(cluster, kubeadmConfig.ClusterConfiguration, kubeadmConfig.InitConfiguration, kubeadmConfig.JoinConfiguration, kubeadmConfig.KubeletConfiguration)...)
	}

	cfg := yip.YipConfig{
		Name: "Kubeadm Kairos Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": finalStages,
		},
	}

	return cfg
}
