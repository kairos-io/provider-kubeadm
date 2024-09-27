package stages

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/kairos-io/kairos/provider-kubeadm/pkg/domain"
	yip "github.com/mudler/yip/pkg/schema"
)

//go:embed mount.tmpl
var mountTemplate string

func GetRootPathMountStage(rootPath string) yip.Stage {
	mps := []domain.MountPoint{
		{
			Name:   "opt-bin",
			Before: domain.KubernetesServices,
			Source: filepath.Join(rootPath, "opt/bin"),
			Target: "/opt/bin",
		},
		{
			Name:   "opt-cni-bin",
			Before: domain.KubernetesServices,
			Source: filepath.Join(rootPath, "opt/cni/bin"),
			Target: "/opt/cni/bin",
		},
		{
			Name:   "etc-cni-netd",
			Before: domain.KubernetesServices,
			Source: filepath.Join(rootPath, "etc/cni/net.d"),
			Target: "/etc/cni/net.d",
		},
		{
			Name:   "var-lib-spectro-containerd",
			Before: domain.KubernetesServices,
			Source: filepath.Join(rootPath, "var/lib/spectro/containerd"),
			Target: "/var/lib/spectro/containerd",
		},
		{
			Name:   "etc-default-kubelet",
			Before: domain.KubernetesServices,
			Source: filepath.Join(rootPath, "etc/default/kubelet"),
			Target: "/etc/default/kubelet",
		},
		{
			Name:   "etc-kubernetes",
			Before: domain.KubernetesServices,
			Source: filepath.Join(rootPath, "etc/kubernetes"),
			Target: "/etc/kubernetes",
		},
		{
			Name:   "var-lib-kubelet",
			Before: domain.KubernetesServices,
			Source: filepath.Join(rootPath, "var/lib/kubelet"),
			Target: "/var/lib/kubelet",
		},
	}

	stage := yip.Stage{
		Name: "Mount Kubernetes directories",
	}
	for _, mp := range mps {
		stage.Files = append(stage.Files, yip.File{
			Path:        filepath.Join(domain.RunSystemdSystemDir, fmt.Sprintf("%s.mount", mp.Name)),
			Permissions: 0644,
			Content:     parseMountUnitFile(mp),
		})

		stage.Commands = append(stage.Commands,
			fmt.Sprintf("mkdir -p %s", mp.Source),
			fmt.Sprintf("mkdir -p %s", mp.Target),
			fmt.Sprintf("systemctl enable --now %s.mount", mp.Name),
		)
	}

	return stage
}

func parseMountUnitFile(mp domain.MountPoint) string {
	mount, _ := template.New("mount").Parse(mountTemplate)
	var buf bytes.Buffer
	_ = mount.Execute(&buf, mp)
	return buf.String()
}
