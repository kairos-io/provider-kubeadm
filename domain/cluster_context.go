package domain

type ClusterContext struct {
	RootPath                    string `json:"rootPath" yaml:"rootPath"`
	NodeRole                    string `json:"nodeRole" yaml:"nodeRole"`
	ClusterCidr                 string `json:"clusterCidr" yaml:"clusterCidr"`
	ServiceCidr                 string `json:"serviceCidr" yaml:"serviceCidr"`
	KubeletArgs                 string `json:"kubeletArgs" yaml:"kubeletArgs"`
	CertSansRevision            string `json:"certSans" yaml:"certSans"`
	ControlPlaneHost            string `json:"controlPlaneHost" yaml:"controlPlaneHost"`
	ClusterToken                string `json:"clusterToken" yaml:"clusterToken"`
	UserOptions                 string `json:"userOptions" yaml:"userOptions"`
	LocalImagesPath             string `json:"localImagesPath" yaml:"localImagesPath"`
	CustomNodeIp                string `json:"customNodeIp" yaml:"customNodeIp"`
	ContainerdServiceFolderName string `json:"containerdServiceFolderName" yaml:"containerdServiceFolderName"`

	EnvConfig map[string]string `json:"envConfig" yaml:"envConfig"`
}
