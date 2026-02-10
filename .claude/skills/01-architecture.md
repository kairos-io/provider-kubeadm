---
skill: Kubeadm Provider Architecture
description: Skill documentation for provider-kubeadm
type: general
repository: provider-kubeadm
team: edge
topics: [kubernetes, provider, edge, cluster]
difficulty: intermediate
last_updated: 2026-02-09
related_skills: []
memory_references: []
---

# Kubeadm Provider Architecture

## Overview

The provider-kubeadm is a Kairos/C3OS cluster provider that orchestrates upstream Kubernetes cluster initialization using kubeadm on Palette Edge hosts. Unlike provider-k3s or provider-rke2 which bundle networking components, provider-kubeadm uses standard kubeadm workflows and requires manual CNI (Container Network Interface) installation.

## Key Concepts

### Kubernetes Distribution: kubeadm (Upstream Kubernetes)

kubeadm is the official tool for bootstrapping production Kubernetes clusters:

- **Upstream Kubernetes**: Uses unmodified Kubernetes from kubernetes.io
- **CNI Agnostic**: Requires manual CNI installation (Flannel, Calico, Cilium, etc.)
- **Production-ready**: Official tool for cluster lifecycle management
- **Version Flexibility**: Supports multiple Kubernetes versions
- **API Versions**: Dual support for v1beta3 (<1.31) and v1beta4 (>=1.31)
- **Community Standard**: Most widely documented Kubernetes deployment method

### kubeadm vs K3s vs RKE2

| Feature | K3s | RKE2 | Kubeadm |
|---------|-----|------|---------|
| **Target Use Case** | Edge, IoT, dev/test | Enterprise edge | General-purpose, upstream |
| **Binary Size** | ~100MB | ~150MB | Varies (standard K8s) |
| **Built-in CNI** | Flannel | Canal (Calico + Flannel) | **NONE - Manual install required** |
| **NetworkPolicy** | Not by default | Yes (via Calico) | Depends on CNI choice |
| **Configuration** | Simple YAML | Simple YAML | kubeadm InitConfiguration/JoinConfiguration |
| **Shell Requirement** | POSIX sh | POSIX sh | **bash (NOT POSIX)** |
| **Complexity** | Lower | Medium | Higher |

### ⚠️ CRITICAL: Manual CNI Installation Required

**Unlike K3s/RKE2, kubeadm does NOT include a CNI**. After cluster initialization, you MUST manually install a CNI or nodes will remain in `NotReady` state with `NetworkPluginNotReady` error.

**Supported CNI Options**:
- **Flannel** (Recommended for simplicity) - VXLAN overlay, port 8472
- **Calico** (Advanced) - BGP routing, NetworkPolicy support
- **Cilium** (Advanced) - eBPF-based networking

See `02-cni-installation.md` for complete CNI installation guide.

### ⚠️ Bash Dependency (NOT POSIX Compliant)

**All 9 shell scripts use `#!/bin/bash`** and bash-specific features:

```bash
#!/bin/bash

exec   > >(tee -ia /var/log/kube-init.log)
exec  2> >(tee -ia /var/log/kube-init.log >& 2)
exec 19>> /var/log/kube-init.log

export BASH_XTRACEFD="19"
set -ex
```

**Bash-specific features used**:
- Process substitution: `>(command)`
- File descriptor assignment: `exec 19>>`
- `BASH_XTRACEFD` variable (bash 4.1+)

**Impact**: Provider-kubeadm requires bash to be installed on edge hosts. Minimal Linux distributions that only provide POSIX sh (dash, ash) will NOT work.

See `10-bash-dependency.md` for full documentation.

### Kairos/C3OS Integration

Provider-kubeadm integrates with Kairos immutable Linux distribution:

- **Cloud-init Configuration**: Declarative cluster setup via cluster section
- **Immutable OS**: A/B partition updates with atomic upgrades
- **Boot Stages**: Yip-based stage execution during boot.before phase
- **Service Management**: systemd service orchestration (kubeadm uses systemd)
- **Image Import**: Local container image preloading for air-gap deployments

### Component Architecture

```
┌─────────────────────────────────────────────────────┐
│              Cloud-Init (User Configuration)         │
│  cluster:                                           │
│    cluster_token: token123                         │
│    control_plane_host: 10.0.1.100                  │
│    role: init|controlplane|worker                  │
│    config: |                                       │
│      apiVersion: kubeadm.k8s.io/v1beta3           │
│      kind: ClusterConfiguration                    │
│      networking:                                    │
│        podSubnet: 10.244.0.0/16                    │
│        serviceSubnet: 10.96.0.0/12                 │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│       Provider-kubeadm (Cluster Plugin)             │
│  • Parse cluster configuration                      │
│  • Generate kubeadm InitConfiguration/JoinConfig    │
│  • Configure proxy settings                         │
│  • Handle role-specific setup                       │
│  • Generate shell scripts for cluster lifecycle     │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│               Yip Stage Execution                   │
│  boot.before:                                       │
│    1. Disable swap                                  │
│    2. Install kubeadm config files                  │
│    3. Import local images (optional)                │
│    4. Execute kube-init.sh or kube-join.sh          │
│    5. Wait for cluster ready                        │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│           kubeadm Cluster Initialization            │
│  ┌──────────────┐  ┌────────────────┐              │
│  │ Control      │  │  Worker Node   │              │
│  │ Plane Node   │  │                │              │
│  │              │  │                │              │
│  │ • API Server │  │ • Kubelet      │              │
│  │ • Scheduler  │  │ • Kube-proxy   │              │
│  │ • Controller │  │ • Container    │              │
│  │ • etcd       │  │   Runtime      │              │
│  └──────────────┘  └────────────────┘              │
│                                                      │
│  ⚠️  NO CNI - Must install manually                 │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│           Manual CNI Installation                   │
│  (User must run: kubectl apply -f flannel.yaml)     │
│                                                      │
│  After CNI installed → Nodes become Ready           │
└─────────────────────────────────────────────────────┘
```

### Configuration Flow

1. **Cluster Definition**: User defines cluster configuration in cloud-init with kubeadm YAML
2. **Provider Execution**: Provider-kubeadm processes configuration via ClusterProvider()
3. **Config Generation**: Creates kubeadm.yaml in /opt/kubeadm/
4. **Script Generation**: Generates bash scripts (kube-init.sh, kube-join.sh, etc.)
5. **kubeadm Execution**: Scripts run `kubeadm init` or `kubeadm join`
6. **CNI Installation**: **User must manually install CNI**
7. **Cluster Ready**: After CNI installed, nodes become Ready

### Dual kubeadm API Support

Provider-kubeadm supports two kubeadm API versions:

**v1beta3** (Kubernetes < 1.31):
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
```

**v1beta4** (Kubernetes >= 1.31):
```yaml
apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
```

The provider automatically detects the API version from user configuration and generates appropriate kubeadm commands.

See `09-version-compatibility.md` for complete details.

### File Structure

```
/opt/kubeadm/
├── kubeadm.yaml                    # Generated kubeadm configuration
├── kubeadm-join.yaml               # Generated join configuration (workers)
├── kube-init.sh                    # Cluster init script (bash)
├── kube-join.sh                    # Node join script (bash)
├── kube-pre-init.sh                # Pre-initialization script
├── kube-post-init.sh               # Post-initialization script
├── kube-reconfigure.sh             # Reconfiguration script
├── kube-upgrade.sh                 # Cluster upgrade script
├── kube-reset.sh                   # Cluster reset script
└── import.sh                       # Image import script

/etc/default/
└── kubelet                         # Kubelet proxy environment variables

/run/systemd/system/
├── containerd.service.d/
│   └── http-proxy.conf             # Containerd proxy config (standard)
└── spectro-containerd.service.d/
    └── http-proxy.conf             # Spectro containerd proxy config (agent mode)

/var/log/
├── kube-init.log                   # Init logs
├── kube-join.log                   # Join logs
├── kube-import-images.log          # Image import logs
└── ...
```

## Implementation Patterns

### Role-Based Configuration

The provider handles three distinct roles with different configurations:

**Init Role (Bootstrap First Control Plane)**:
```go
// pkg/provider/provider.go
case clusterplugin.RoleInit:
    // Generate InitConfiguration
    clusterConfig := domain.ClusterConfig{
        APIVersion: determineAPIVersion(cluster.Config),
        Kind:       "InitConfiguration",
        // ... init-specific config
    }
    // Creates kube-init.sh with `kubeadm init`
```

**ControlPlane Role (Additional Control Plane)**:
```go
case clusterplugin.RoleControlPlane:
    // Generate JoinConfiguration with control-plane flag
    joinConfig := domain.JoinConfig{
        ControlPlane: true,
        CertificateKey: cluster.CertificateKey,
        // ... control plane join config
    }
    // Creates kube-join.sh with `kubeadm join --control-plane`
```

**Worker Role (Worker Node)**:
```go
case clusterplugin.RoleWorker:
    // Generate JoinConfiguration (worker only)
    joinConfig := domain.JoinConfig{
        ControlPlane: false,
        // ... worker join config
    }
    // Creates kube-join.sh with `kubeadm join`
```

### Configuration File Generation

```go
// pkg/provider/provider.go:parseFiles()
files := []yip.File{
    {
        Path:        "/opt/kubeadm/kubeadm.yaml",
        Permissions: 0600,
        Content:     string(kubeadmConfig),  // Generated kubeadm YAML
    },
    {
        Path:        "/opt/kubeadm/kube-init.sh",
        Permissions: 0755,
        Content:     string(initScript),      // Bash script
    },
    // ... other scripts
}
```

### Proxy Configuration

Provider-kubeadm configures proxy settings in two locations:

1. **kubelet environment variables**: `/etc/default/kubelet`
   ```bash
   HTTP_PROXY=http://proxy.example.com:8080
   HTTPS_PROXY=https://proxy.example.com:8080
   NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local
   ```

2. **containerd systemd override**: `/run/systemd/system/containerd.service.d/http-proxy.conf`
   ```ini
   [Service]
   Environment="HTTP_PROXY=http://proxy.example.com:8080"
   Environment="HTTPS_PROXY=https://proxy.example.com:8080"
   Environment="NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local"
   ```

See `04-proxy-configuration.md` for complete proxy setup.

### Deployment Modes

Provider-kubeadm supports both Stylus deployment modes via `STYLUS_ROOT` environment variable:

**Appliance Mode** (Cluster runs locally on device):
```bash
STYLUS_ROOT=/
```
- Config files in `/etc/default/kubelet`
- Containerd in `/run/systemd/system/containerd.service.d/`

**Agent Mode** (Cluster managed by Palette):
```bash
STYLUS_ROOT=/persistent/spectro
```
- Config files in `/persistent/spectro/etc/default/kubelet`
- Spectro containerd in `/run/systemd/system/spectro-containerd.service.d/`

See `06-deployment-modes.md` for complete details.

## Common Pitfalls

### ❌ WRONG: Expecting CNI to be pre-installed

```bash
# After kubeadm init, assuming nodes are Ready
kubectl get nodes
# NAME     STATUS     ROLES           AGE   VERSION
# node-1   NotReady   control-plane   1m    v1.28.0
# ❌ Node stuck in NotReady - no CNI installed!
```

### ✅ CORRECT: Install CNI after kubeadm init

```bash
# After kubeadm init, install Flannel CNI
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Wait for CNI pods to run
kubectl get pods -n kube-flannel
# NAME                    READY   STATUS    RESTARTS   AGE
# kube-flannel-ds-amd64   1/1     Running   0          30s

# Now nodes become Ready
kubectl get nodes
# NAME     STATUS   ROLES           AGE   VERSION
# node-1   Ready    control-plane   2m    v1.28.0
# ✅ Node is Ready after CNI installation
```

### ❌ WRONG: Using POSIX shell

```bash
#!/bin/sh
# provider-kubeadm scripts use bash-specific features
exec 19>> /var/log/kube-init.log
# ❌ POSIX sh does not support file descriptor assignment
```

### ✅ CORRECT: Using bash

```bash
#!/bin/bash
# provider-kubeadm requires bash
exec 19>> /var/log/kube-init.log
export BASH_XTRACEFD="19"
# ✅ bash supports these features
```

### ❌ WRONG: Forgetting proxy NO_PROXY for CNI

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.example.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
    # ❌ Missing pod and service CIDRs - CNI communication will fail
```

### ✅ CORRECT: Including pod/service subnets in NO_PROXY

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.example.com:8080"
    NO_PROXY: "localhost,127.0.0.1,10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local"
    # ✅ Provider auto-appends CIDRs from networking config
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16      # Auto-added to NO_PROXY
      serviceSubnet: 10.96.0.0/12   # Auto-added to NO_PROXY
```

## Integration Points

### Dependencies

- **gomi**: Common Go libraries (logging, k8s utilities)
- **hapi**: API schema definitions (for Palette integration)
- **Kairos/C3OS**: Immutable Linux distribution framework
- **kubeadm**: Kubernetes cluster bootstrapping tool (installed on hosts)
- **containerd**: Container runtime (installed on hosts)
- **bash**: Shell interpreter (required on hosts)

### Consumers

- **Stylus**: Edge orchestration agent (appliance and agent modes)
  - Appliance mode: Local cluster management
  - Agent mode: Palette-managed cluster with provider events
- **Palette**: Cluster orchestration platform
  - Provides cluster configuration via Stylus
  - Receives cluster status and events
- **Teams & Teams-Edge-Native**: Automated testing frameworks
  - Validates kubeadm cluster functionality
  - Tests CNI installation scenarios
  - Verifies proxy and deployment mode configurations

## Reference Examples

**Cluster Configuration Files**:
- `/Users/rishi/work/src/provider-kubeadm/pkg/provider/provider.go` - Main provider implementation
- `/Users/rishi/work/src/provider-kubeadm/pkg/domain/types.go` - Configuration data structures
- `/Users/rishi/work/src/provider-kubeadm/stages/proxy.go` - Proxy configuration logic
- `/Users/rishi/work/src/provider-kubeadm/utils/proxy.go` - Proxy utility functions

**Shell Scripts** (bash-based):
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-init.sh` - Cluster initialization
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-join.sh` - Node join
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-reset.sh` - Cluster reset
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-upgrade.sh` - Cluster upgrade

## Related Skills

- See `provider-kubeadm:02-cni-installation` for **CRITICAL** CNI installation guide
- See `provider-kubeadm:03-configuration-patterns` for cluster configuration examples
- See `provider-kubeadm:04-proxy-configuration` for proxy setup details
- See `provider-kubeadm:05-cluster-roles` for role-specific behavior
- See `provider-kubeadm:06-deployment-modes` for appliance vs agent mode
- See `provider-kubeadm:08-troubleshooting` for common issues and fixes
- See `provider-kubeadm:10-bash-dependency` for POSIX vs bash details

**Related Provider Skills**:
- See `provider-k3s:01-architecture` for K3s comparison
- See `provider-rke2:01-architecture` for RKE2 comparison
- See `stylus:ai-knowledge-base` for edge orchestration integration

## Documentation References

**Official kubeadm Documentation**:
- https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/
- https://kubernetes.io/docs/reference/setup-tools/kubeadm/

**CNI Documentation**:
- Flannel: https://github.com/flannel-io/flannel
- Calico: https://docs.tigera.io/calico/latest/about/
- Cilium: https://docs.cilium.io/

**Kairos Documentation**:
- https://kairos.io/docs/

**Provider Repository**:
- https://github.com/kairos-io/provider-kubeadm
