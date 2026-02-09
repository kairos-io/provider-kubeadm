# kubeadm Configuration Patterns

## Overview

Provider-kubeadm uses standard kubeadm configuration with cloud-init integration. This skill covers configuration patterns for master/worker nodes, network setup, proxy configuration, and version compatibility.

## Key Concepts

### kubeadm Configuration Structure

kubeadm uses typed YAML configurations:

**InitConfiguration** (First control plane node):
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.0123456789abcdef
  ttl: 24h0m0s
  usages:
  - signing
  - authentication
nodeRegistration:
  name: control-1
  criSocket: unix:///run/containerd/containerd.sock
```

**ClusterConfiguration** (Cluster-wide settings):
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.28.0
clusterName: edge-cluster
controlPlaneEndpoint: "10.0.1.100:6443"
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
  dnsDomain: cluster.local
```

**JoinConfiguration** (Worker and additional control plane nodes):
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
discovery:
  bootstrapToken:
    token: abcdef.0123456789abcdef
    apiServerEndpoint: "10.0.1.100:6443"
    caCertHashes:
    - "sha256:1234567890abcdef..."
nodeRegistration:
  name: worker-1
  criSocket: unix:///run/containerd/containerd.sock
```

### Cloud-init Integration

Provider-kubeadm processes cloud-init configuration:

```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: init  # or controlplane, or worker
  env:
    HTTP_PROXY: "http://proxy.example.com:8080"
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

## Implementation Patterns

### Pattern 1: Init Node (Bootstrap First Control Plane)

**Use Case**: First control plane node that initializes the cluster and etcd.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"   # This node's IP
  role: init
  env:
    HTTP_PROXY: "http://proxy.example.com:8080"
    HTTPS_PROXY: "https://proxy.example.com:8080"
    NO_PROXY: "localhost,127.0.0.1,.corp.com"
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    kubernetesVersion: v1.28.0
    clusterName: edge-cluster-001
    controlPlaneEndpoint: "10.0.1.100:6443"
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
      dnsDomain: cluster.local
    apiServer:
      certSANs:
      - "10.0.1.100"
      - "edge-control-1"
      extraArgs:
        authorization-mode: "Node,RBAC"
    etcd:
      local:
        dataDir: /var/lib/etcd
```

**Generated kubeadm.yaml** (provider creates):
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.0123456789abcdef
  ttl: 24h0m0s
nodeRegistration:
  criSocket: unix:///run/containerd/containerd.sock
  taints: []
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.28.0
clusterName: edge-cluster-001
controlPlaneEndpoint: "10.0.1.100:6443"
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
  dnsDomain: cluster.local
apiServer:
  certSANs:
  - "10.0.1.100"
  - "edge-control-1"
  extraArgs:
    authorization-mode: "Node,RBAC"
etcd:
  local:
    dataDir: /var/lib/etcd
```

**Execution**:
```bash
# Provider generates kube-init.sh
kubeadm init --config /opt/kubeadm/kubeadm.yaml \
  --upload-certs \
  --ignore-preflight-errors=NumCPU \
  --ignore-preflight-errors=Mem \
  -v=5
```

---

### Pattern 2: ControlPlane Node (Additional Control Plane)

**Use Case**: Additional control plane nodes joining existing cluster for HA.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"   # First control plane IP
  certificate_key: "0123456789abcdef0123456789abcdef..."
  role: controlplane
  env:
    HTTP_PROXY: "http://proxy.example.com:8080"
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated kubeadm-join.yaml**:
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
discovery:
  bootstrapToken:
    token: abcdef.0123456789abcdef
    apiServerEndpoint: "10.0.1.100:6443"
    caCertHashes:
    - "sha256:1234567890abcdef..."  # Provider extracts from init node
  timeout: 5m0s
nodeRegistration:
  name: control-2
  criSocket: unix:///run/containerd/containerd.sock
  taints: []
controlPlane:
  localAPIEndpoint:
    advertiseAddress: 10.0.1.101   # This node's IP
    bindPort: 6443
  certificateKey: "0123456789abcdef..."  # From cloud-init
```

**Execution**:
```bash
# Provider generates kube-join.sh
kubeadm join 10.0.1.100:6443 \
  --config /opt/kubeadm/kubeadm-join.yaml \
  --control-plane \
  --certificate-key 0123456789abcdef... \
  -v=5
```

---

### Pattern 3: Worker Node

**Use Case**: Worker nodes running application pods.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: worker
  env:
    HTTP_PROXY: "http://proxy.example.com:8080"
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: JoinConfiguration
    nodeRegistration:
      name: worker-1
      kubeletExtraArgs:
        node-labels: "node-role.kubernetes.io/worker="
```

**Generated kubeadm-join.yaml**:
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
discovery:
  bootstrapToken:
    token: abcdef.0123456789abcdef
    apiServerEndpoint: "10.0.1.100:6443"
    caCertHashes:
    - "sha256:1234567890abcdef..."
  timeout: 5m0s
nodeRegistration:
  name: worker-1
  criSocket: unix:///run/containerd/containerd.sock
  taints: []
  kubeletExtraArgs:
    node-labels: "node-role.kubernetes.io/worker="
```

**Execution**:
```bash
# Provider generates kube-join.sh
kubeadm join 10.0.1.100:6443 \
  --config /opt/kubeadm/kubeadm-join.yaml \
  -v=5
```

---

### Pattern 4: Custom Network CIDRs

**Use Case**: Non-default pod/service subnets to avoid conflicts with existing networks.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  role: init
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.100.0.0/16        # Custom pod network
      serviceSubnet: 10.200.0.0/16    # Custom service network
      dnsDomain: edge.local           # Custom domain
```

**Important**: CNI configuration MUST match these CIDRs

```bash
# After kubeadm init, install Flannel with matching CIDR
curl -sO https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Edit net-conf.json to match podSubnet
# Change "Network": "10.244.0.0/16" to "Network": "10.100.0.0/16"
sed -i 's|"Network": "10.244.0.0/16"|"Network": "10.100.0.0/16"|' kube-flannel.yml

kubectl apply -f kube-flannel.yml
```

---

### Pattern 5: Multi-interface Nodes

**Use Case**: Nodes with multiple network interfaces (e.g., management + data plane).

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  role: init
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    controlPlaneEndpoint: "10.0.1.100:6443"   # Management interface
    ---
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: InitConfiguration
    localAPIEndpoint:
      advertiseAddress: 10.0.1.100     # Management interface IP
      bindPort: 6443
    nodeRegistration:
      kubeletExtraArgs:
        node-ip: "10.0.1.100"          # Force kubelet to use management IP
```

**For worker nodes with multi-interface**:
```yaml
nodeRegistration:
  kubeletExtraArgs:
    node-ip: "10.0.2.50"               # Worker data plane IP
```

---

### Pattern 6: Air-Gap Deployment

**Use Case**: Clusters without internet access, pre-loaded container images.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  import_images: true                  # Enable image import
  role: init
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    imageRepository: registry.corp.com/kubernetes  # Local registry
    kubernetesVersion: v1.28.0
```

**Pre-import Images**:
```bash
# Provider automatically imports images from:
# /var/opt/spectro-cloud/cluster-images/
# /spectro-cloud/cluster-images/

# Images must be available in OCI layout format
# Provider executes import.sh to load into containerd
```

---

## Common Pitfalls

### ❌ WRONG: Missing podSubnet

```yaml
# kubeadm config without podSubnet
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
networking:
  serviceSubnet: 10.96.0.0/12
  # ❌ Missing podSubnet!
```

```bash
kubeadm init --config kubeadm.yaml
# ✅ Init succeeds, but...

kubectl apply -f kube-flannel.yml
# ❌ Flannel fails - no podSubnet specified in kubeadm config
```

### ✅ CORRECT: Explicit podSubnet

```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
networking:
  podSubnet: 10.244.0.0/16     # ✅ Required for CNI
  serviceSubnet: 10.96.0.0/12
```

---

### ❌ WRONG: Token format invalid

```yaml
cluster:
  cluster_token: "my-secret-token"   # ❌ Invalid format!
```

kubeadm tokens MUST match format: `[a-z0-9]{6}.[a-z0-9]{16}`

```bash
kubeadm join will fail:
# error: invalid bootstrap token
```

### ✅ CORRECT: Valid token format

```yaml
cluster:
  cluster_token: "abcdef.0123456789abcdef"   # ✅ Valid format
```

Generate valid tokens:
```bash
kubeadm token generate
# abcdef.0123456789abcdef
```

---

### ❌ WRONG: Mismatched API versions

```yaml
# Init node uses v1beta3
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration

# Worker joins with v1beta4
apiVersion: kubeadm.k8s.io/v1beta4   # ❌ Mismatch!
kind: JoinConfiguration
```

### ✅ CORRECT: Consistent API versions

```yaml
# All nodes use same API version
apiVersion: kubeadm.k8s.io/v1beta3
```

---

### ❌ WRONG: Control plane endpoint points to self on init

```yaml
# On init node (10.0.1.100)
controlPlaneEndpoint: "localhost:6443"   # ❌ Workers can't reach "localhost"
```

### ✅ CORRECT: Use reachable IP/DNS

```yaml
controlPlaneEndpoint: "10.0.1.100:6443"   # ✅ Workers can reach this
# OR
controlPlaneEndpoint: "edge-control.example.com:6443"   # ✅ DNS name
```

---

## Integration Points

### With Provider

- Provider parses `cluster.config` and generates kubeadm YAML files
- Provider generates role-specific shell scripts (kube-init.sh, kube-join.sh)
- Provider merges user config with generated config

### With Proxy

- Provider extracts `podSubnet` and `serviceSubnet` for NO_PROXY calculation
- Proxy settings applied to kubelet and containerd (see `04-proxy-configuration.md`)

### With CNI

- CNI MUST read podSubnet from kubeadm config or match it in CNI manifest
- Provider does NOT automatically install CNI (manual step required)

### With Stylus

- Appliance mode: config files in `/opt/kubeadm/`
- Agent mode: config files in `${STYLUS_ROOT}/opt/kubeadm/`

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-kubeadm/pkg/provider/provider.go` - Config parsing and generation
- `/Users/rishi/work/src/provider-kubeadm/pkg/domain/types.go` - Configuration data structures
- `/Users/rishi/work/src/provider-kubeadm/stages/files.go` - File generation logic

**Shell Scripts**:
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-init.sh` - Init node execution
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-join.sh` - Join node execution

## Related Skills

- See `provider-kubeadm:01-architecture` for kubeadm overview
- See `provider-kubeadm:02-cni-installation` for CNI setup after config
- See `provider-kubeadm:04-proxy-configuration` for proxy setup with configs
- See `provider-kubeadm:05-cluster-roles` for role-specific behavior details
- See `provider-kubeadm:09-version-compatibility` for v1beta3 vs v1beta4 differences

## Documentation References

**kubeadm Configuration Reference**:
- v1beta3: https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta3/
- v1beta4: https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta4/

**kubeadm Commands**:
- https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/
- https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-join/
