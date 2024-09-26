package provider

import (
	"encoding/json"

	"github.com/kairos-io/kairos/provider-kubeadm/pkg/domain"
	"github.com/kairos-io/kairos/provider-kubeadm/pkg/stages"
	"github.com/kairos-io/kairos/provider-kubeadm/pkg/utils"
	kyaml "sigs.k8s.io/yaml"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
)

func ClusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	var finalStages []yip.Stage
	var kubeadmConfig domain.KubeadmConfig

	if cluster.Options != "" {
		userOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
		_ = json.Unmarshal(userOptions, &kubeadmConfig)
	}

	clusterRootPath := utils.GetClusterRootPath(cluster)

	if len(clusterRootPath) > 0 && clusterRootPath != "/" {
		finalStages = append(finalStages, stages.GetRootPathMountStage(clusterRootPath))
	}

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
