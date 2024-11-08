package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/kairos-io/kairos/provider-kubeadm/log"

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
	log.InitLogger("/var/log/provider-kubeadm.log")
	logrus.Info("starting provider-kubeadm")
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
	logrus.Infof("completed provider-kubeadm")
}

func handleClusterReset(event *pluggable.Event) pluggable.EventResponse {
	logrus.Info("handling cluster reset event")

	var payload bus.EventPayload
	var config clusterplugin.Config
	var response pluggable.EventResponse

	// parse the boot payload
	if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
		logrus.Error(fmt.Sprintf("failed to parse boot event: %s", err.Error()))
		response.Error = fmt.Sprintf("failed to parse boot event: %s", err.Error())
		return response
	}

	// parse config from boot payload
	if err := yaml.Unmarshal([]byte(payload.Config), &config); err != nil {
		logrus.Error(fmt.Sprintf("failed to parse config from boot event: %s", err.Error()))
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
		logrus.Error(fmt.Sprintf("failed to reset cluster: %s", string(output)))
		response.Error = fmt.Sprintf("failed to reset cluster: %s", string(output))
	}

	return response
}

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	var finalStages []yip.Stage

	clusterCtx := CreateClusterContext(cluster)

	preStage := []yip.Stage{
		stages.GetPreKubeadmProxyStage(clusterCtx, cluster),
		stages.GetPreKubeadmCommandStages(clusterCtx.RootPath),
		stages.GetPreKubeadmSwapOffDisableStage(),
		stages.GetPreKubeadmImportCoreK8sImageStage(clusterCtx.RootPath),
	}

	if cluster.ImportLocalImages {
		preStage = append(preStage, stages.GetPreKubeadmImportLocalImageStage(cluster))
	}

	finalStages = append(finalStages, preStage...)

	cmpResult, err := utils.IsKubeadmVersionGreaterThan131()
	if err != nil {
		logrus.Fatalf("failed to check if kubeadm version is greater than 131: %v", err)
	} else if cmpResult < 0 {
		logrus.Info("kubeadm version is less than 1.31")
		finalStages = append(finalStages, getV1Beta3FinalStage(cluster, clusterCtx)...)
	} else {
		logrus.Info("kubeadm version is greater than or equal to 1.31")
		finalStages = append(finalStages, getV1Beta4FinalStage(cluster, clusterCtx)...)
	}

	cfg := yip.YipConfig{
		Name: "Kubeadm Kairos Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": finalStages,
		},
	}

	return cfg
}

func CreateClusterContext(cluster clusterplugin.Cluster) *domain.ClusterContext {
	return &domain.ClusterContext{
		RootPath:         utils.GetClusterRootPath(cluster),
		NodeRole:         string(cluster.Role),
		EnvConfig:        cluster.Env,
		ControlPlaneHost: cluster.ControlPlaneHost,
		ClusterToken:     utils.TransformToken(cluster.ClusterToken),
	}
}

func getV1Beta3FinalStage(cluster clusterplugin.Cluster, clusterCtx *domain.ClusterContext) []yip.Stage {
	var finalStages []yip.Stage
	var kubeadmConfig domain.KubeadmConfigBeta3

	if cluster.Options != "" {
		userOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
		_ = json.Unmarshal(userOptions, &kubeadmConfig)
	}

	if cluster.Role == clusterplugin.RoleInit {
		finalStages = append(finalStages, stages.GetInitYipStagesV1Beta3(clusterCtx, kubeadmConfig)...)
	} else if (cluster.Role == clusterplugin.RoleControlPlane) || (cluster.Role == clusterplugin.RoleWorker) {
		finalStages = append(finalStages, stages.GetJoinYipStagesV1Beta3(clusterCtx, kubeadmConfig)...)
	}

	return finalStages
}

func getV1Beta4FinalStage(cluster clusterplugin.Cluster, clusterCtx *domain.ClusterContext) []yip.Stage {
	var finalStages []yip.Stage
	var kubeadmConfig domain.KubeadmConfigBeta4

	if cluster.Options != "" {
		userOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
		_ = json.Unmarshal(userOptions, &kubeadmConfig)
	}

	if cluster.Role == clusterplugin.RoleInit {
		finalStages = append(finalStages, stages.GetInitYipStagesV1Beta4(clusterCtx, kubeadmConfig)...)
	} else if (cluster.Role == clusterplugin.RoleControlPlane) || (cluster.Role == clusterplugin.RoleWorker) {
		finalStages = append(finalStages, stages.GetJoinYipStagesV1Beta4(clusterCtx, kubeadmConfig)...)
	}

	return finalStages
}
