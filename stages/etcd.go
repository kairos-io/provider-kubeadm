package stages

import (
	"fmt"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	yip "github.com/mudler/yip/pkg/schema"
)

func GetPreKubeadmEtcdUserStages(clusterCtx *domain.ClusterContext) []yip.Stage {
	etcdDataDir := clusterCtx.EtcdDataDir

	return []yip.Stage{
		{
			Name: "Create etcd user and group (CNTR-K8-003120)",
			Users: map[string]yip.User{
				"etcd": {
					System:       true,
					PrimaryGroup: "etcd",
					NoCreateHome: true,
					Shell:        "/sbin/nologin",
				},
			},
		},
		{
			Name: "Set etcd data directory ownership and permissions (CNTR-K8-003260)",
			Directories: []yip.Directory{
				{
					Path:        etcdDataDir,
					Permissions: 0700,
				},
			},
			Commands: []string{
				fmt.Sprintf("chown -R etcd:etcd %s", etcdDataDir),
				fmt.Sprintf("chmod 700 %s", etcdDataDir),
				fmt.Sprintf("find %s -type f -exec chmod 600 {} \\; 2>/dev/null || true", etcdDataDir),
			},
		},
	}
}
