package config

const (
	InitConfigurationTemplate = `---
{{.ClusterConfiguration}}
---
{{.InitConfiguration}}
---
{{.KubeletConfiguration}}
`
)

const (
	JoinConfigurationTemplate = `---
{{.JoinConfiguration}}
---
{{.KubeletConfiguration}}
`
)

type Init struct {
	ClusterConfiguration string
	InitConfiguration    string
	KubeletConfiguration string
}

type Join struct {
	JoinConfiguration    string
	KubeletConfiguration string
}

type ClusterConfiguration struct {
	TypeMeta             TypeMeta   `yaml:",inline"`
	ControlPlaneEndpoint string     `yaml:"controlPlaneEndpoint"`
	Networking           Networking `yaml:"networking"`
}

type Networking struct {
	PodSubnet string `yaml:"podSubnet"`
}

type InitConfiguration struct {
	TypeMeta        TypeMeta         `yaml:",inline"`
	BootstrapTokens []BootstrapToken `yaml:"bootstrapTokens"`
	CertificateKey  string           `yaml:"certificateKey"`
}

type BootstrapToken struct {
	Token string `yaml:"token"`
}

type JoinConfiguration struct {
	TypeMeta         TypeMeta          `yaml:",inline"`
	Discovery        Discovery         `yaml:"discovery"`
	JoinControlPlane *JoinControlPlane `yaml:"controlPlane,omitempty"`
}

type JoinControlPlane struct {
	CertificateKey string `yaml:"certificateKey,omitempty"`
}

type Discovery struct {
	BootstrapToken BootstrapTokenDiscovery `yaml:"bootstrapToken,omitempty"`
}

type BootstrapTokenDiscovery struct {
	Token                    string `yaml:"token"`
	APIServerEndpoint        string `yaml:"apiServerEndpoint,omitempty"`
	UnsafeSkipCAVerification bool   `yaml:"unsafeSkipCAVerification,omitempty"`
}

type KubeletConfiguration struct {
	TypeMeta     TypeMeta `yaml:",inline"`
	CgroupDriver string   `yaml:"cgroupDriver"`
}

type TypeMeta struct {
	Kind       string `yaml:"kind"`
	APIVersion string `yaml:"apiVersion"`
}
