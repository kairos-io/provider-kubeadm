---
skill: kubeadm Deployment Modes
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

# kubeadm Deployment Modes

## Overview

Provider-kubeadm supports both **appliance mode** and **agent mode** via Stylus integration using the `STYLUS_ROOT` environment variable. Additionally, it supports dual containerd socket configuration for agent mode.

## Key Concepts

### Deployment Modes

**Appliance Mode** (Local Cluster Management):
- Cluster runs locally on the edge device
- Stylus manages cluster locally without Palette control plane
- `STYLUS_ROOT=/`
- Configuration files in standard locations (`/etc/`, `/opt/`)
- Uses standard `containerd` runtime

**Agent Mode** (Palette-Managed Cluster):
- Cluster managed remotely by Palette control plane
- Stylus acts as agent reporting to Palette
- `STYLUS_ROOT=/persistent/spectro`
- Configuration files in persistent storage
- Uses `spectro-containerd` runtime

### STYLUS_ROOT Environment Variable

The `STYLUS_ROOT` variable determines file paths and runtime socket:

| Mode | STYLUS_ROOT | kubelet config | containerd socket |
|------|-------------|----------------|-------------------|
| Appliance | `/` | `/etc/default/kubelet` | `unix:///run/containerd/containerd.sock` |
| Agent | `/persistent/spectro` | `/persistent/spectro/etc/default/kubelet` | `unix:///run/spectro-containerd/spectro-containerd.sock` |

## Implementation Patterns

### Pattern 1: Appliance Mode Deployment

**Use Case**: Standalone edge clusters managed locally by Stylus.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: init
  # No STYLUS_ROOT - defaults to appliance mode
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated File Paths**:
```bash
# kubeadm configuration
/opt/kubeadm/kubeadm.yaml
/opt/kubeadm/kube-init.sh
/opt/kubeadm/kube-join.sh

# Proxy configuration
/etc/default/kubelet

# Containerd proxy
/run/systemd/system/containerd.service.d/http-proxy.conf

# kubeadm uses standard containerd
nodeRegistration:
  criSocket: unix:///run/containerd/containerd.sock
```

**Execution Flow**:
```bash
# Provider sets STYLUS_ROOT=/
export STYLUS_ROOT=/

# Provider generates files
cat > /opt/kubeadm/kubeadm.yaml <<EOF
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
nodeRegistration:
  criSocket: unix:///run/containerd/containerd.sock
EOF

# kubeadm init
/opt/kubeadm/kube-init.sh
```

---

### Pattern 2: Agent Mode Deployment

**Use Case**: Palette-managed edge clusters with central control plane.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: init
  env:
    STYLUS_ROOT: "/persistent/spectro"   # ← Agent mode
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated File Paths**:
```bash
# kubeadm configuration
/opt/kubeadm/kubeadm.yaml          # Still in /opt (not STYLUS_ROOT)
/opt/kubeadm/kube-init.sh
/opt/kubeadm/kube-join.sh

# Proxy configuration
${STYLUS_ROOT}/etc/default/kubelet = /persistent/spectro/etc/default/kubelet

# Spectro containerd proxy
/run/systemd/system/spectro-containerd.service.d/http-proxy.conf

# kubeadm uses spectro-containerd
nodeRegistration:
  criSocket: unix:///run/spectro-containerd/spectro-containerd.sock
```

**Execution Flow**:
```bash
# Provider sets STYLUS_ROOT
export STYLUS_ROOT=/persistent/spectro

# Provider generates files with STYLUS_ROOT prefix
mkdir -p ${STYLUS_ROOT}/etc/default
cat > ${STYLUS_ROOT}/etc/default/kubelet <<EOF
HTTP_PROXY=http://proxy.corp.com:8080
EOF

# kubeadm uses spectro-containerd socket
cat > /opt/kubeadm/kubeadm.yaml <<EOF
nodeRegistration:
  criSocket: unix:///run/spectro-containerd/spectro-containerd.sock
EOF

# kubeadm init
/opt/kubeadm/kube-init.sh
```

---

### Pattern 3: Dual Containerd Socket (Agent Mode)

**Use Case**: Agent mode with separate containerd runtime for cluster vs system.

**Architecture**:
```
┌──────────────────────────────────────────┐
│ Edge Host (Agent Mode)                   │
│                                           │
│  ┌────────────────┐  ┌─────────────────┐ │
│  │  containerd    │  │ spectro-        │ │
│  │  (standard)    │  │ containerd      │ │
│  │                │  │ (cluster)       │ │
│  │ Port: /run/    │  │ Port: /run/     │ │
│  │ containerd/    │  │ spectro-        │ │
│  │ containerd.sock│  │ containerd/...  │ │
│  │                │  │                 │ │
│  │ System pods    │  │ Cluster pods    │ │
│  │ Stylus         │  │ App workloads   │ │
│  └────────────────┘  └─────────────────┘ │
└──────────────────────────────────────────┘
```

**Containerd Socket Configuration**:

**Appliance Mode**:
```yaml
nodeRegistration:
  criSocket: unix:///run/containerd/containerd.sock
```

**Agent Mode**:
```yaml
nodeRegistration:
  criSocket: unix:///run/spectro-containerd/spectro-containerd.sock
```

**Provider Logic**:
```go
// pkg/provider/provider.go
func getContainerdSocket(stylusRoot string) string {
    if stylusRoot == "/persistent/spectro" {
        return "unix:///run/spectro-containerd/spectro-containerd.sock"
    }
    return "unix:///run/containerd/containerd.sock"
}
```

---

### Pattern 4: Mode Detection in Scripts

**Use Case**: Shell scripts need to detect deployment mode.

**kube-init.sh Mode Detection**:
```bash
#!/bin/bash

# Detect STYLUS_ROOT
root_path="${STYLUS_ROOT:-/}"

# Appliance mode
if [ "$root_path" = "/" ]; then
  echo "Running in appliance mode"
  kubelet_config="/etc/default/kubelet"
  containerd_socket="unix:///run/containerd/containerd.sock"

# Agent mode
elif [ "$root_path" = "/persistent/spectro" ]; then
  echo "Running in agent mode"
  kubelet_config="${root_path}/etc/default/kubelet"
  containerd_socket="unix:///run/spectro-containerd/spectro-containerd.sock"
fi

# Use detected paths
source "$kubelet_config"
kubeadm init --config /opt/kubeadm/kubeadm.yaml ...
```

---

### Pattern 5: Provider Events (Agent Mode Only)

**Use Case**: Palette needs cluster status updates in agent mode.

**Provider Events**:
```go
// Only in agent mode (STYLUS_ROOT=/persistent/spectro)
if stylusRoot == "/persistent/spectro" {
    // Send events to Palette via Stylus
    sendProviderEvent("ClusterInitStarted", clusterUID)
    // ... kubeadm init ...
    sendProviderEvent("ClusterInitCompleted", clusterUID)
}
```

**Event Types**:
- `ClusterInitStarted`
- `ClusterInitCompleted`
- `ClusterInitFailed`
- `NodeJoinStarted`
- `NodeJoinCompleted`
- `NodeJoinFailed`

**No Events in Appliance Mode**: Stylus manages locally, no Palette integration.

---

## Mode Comparison

| Aspect | Appliance Mode | Agent Mode |
|--------|----------------|------------|
| **STYLUS_ROOT** | `/` | `/persistent/spectro` |
| **kubelet config** | `/etc/default/kubelet` | `/persistent/spectro/etc/default/kubelet` |
| **containerd socket** | `/run/containerd/containerd.sock` | `/run/spectro-containerd/spectro-containerd.sock` |
| **Palette Integration** | No | Yes (via Stylus) |
| **Provider Events** | No | Yes |
| **Management** | Local (Stylus) | Remote (Palette) |
| **Use Case** | Standalone edge | Managed edge fleet |

---

## Common Pitfalls

### ❌ WRONG: Hardcoded paths without STYLUS_ROOT

```go
// Provider code
kubeletConfigPath := "/etc/default/kubelet"
// ❌ Ignores STYLUS_ROOT - breaks agent mode
```

### ✅ CORRECT: Use STYLUS_ROOT prefix

```go
stylusRoot := os.Getenv("STYLUS_ROOT")
if stylusRoot == "" {
    stylusRoot = "/"  // Default to appliance mode
}
kubeletConfigPath := filepath.Join(stylusRoot, "etc/default/kubelet")
// ✅ Works in both modes
```

---

### ❌ WRONG: Using standard containerd in agent mode

```yaml
# Agent mode (STYLUS_ROOT=/persistent/spectro)
nodeRegistration:
  criSocket: unix:///run/containerd/containerd.sock
  # ❌ Should use spectro-containerd
```

**Result**: kubeadm fails to connect to runtime
```bash
kubeadm init
# [ERROR CRI]: container runtime is not running
# ❌ Wrong socket - spectro-containerd not reached
```

### ✅ CORRECT: Use spectro-containerd in agent mode

```yaml
nodeRegistration:
  criSocket: unix:///run/spectro-containerd/spectro-containerd.sock
  # ✅ Correct socket for agent mode
```

---

### ❌ WRONG: Assuming appliance mode always

```bash
# Shell script
kubelet_config="/etc/default/kubelet"
# ❌ Hardcoded - breaks in agent mode
```

### ✅ CORRECT: Detect mode dynamically

```bash
root_path="${STYLUS_ROOT:-/}"
kubelet_config="${root_path}/etc/default/kubelet"
# ✅ Works in both modes
```

---

## Integration Points

### With Stylus

**Appliance Mode**:
- Stylus calls provider-kubeadm with `STYLUS_ROOT=/`
- Stylus manages cluster lifecycle locally
- No Palette communication

**Agent Mode**:
- Stylus calls provider-kubeadm with `STYLUS_ROOT=/persistent/spectro`
- Stylus forwards provider events to Palette
- Palette provides cluster configuration

### With Proxy Configuration

- Proxy files respect STYLUS_ROOT in both modes
- NO_PROXY calculation same for both modes
- containerd proxy config uses mode-specific service

### With CNI

- CNI installation same for both modes
- CNI pods use mode-specific containerd socket
- No CNI configuration changes between modes

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-kubeadm/pkg/provider/provider.go` - Mode detection logic
- `/Users/rishi/work/src/provider-kubeadm/stages/proxy.go` - STYLUS_ROOT handling in proxy config
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-init.sh` - Mode detection in scripts

**Stylus Integration**:
- See `stylus:ai-knowledge-base/contexts-kb/context-appliance-mode.md` for appliance mode details
- See `stylus:ai-knowledge-base/contexts-kb/context-agent-mode.md` for agent mode details

## Related Skills

- See `provider-kubeadm:01-architecture` for deployment mode overview
- See `provider-kubeadm:04-proxy-configuration` for mode-specific proxy setup
- See `provider-kubeadm:08-troubleshooting` for mode-specific issues

**Related Provider Skills**:
- See `provider-k3s:07-deployment-modes` for k3s deployment modes (same STYLUS_ROOT concept)
- See `provider-rke2:07-deployment-modes` for rke2 deployment modes (same STYLUS_ROOT concept)

## Documentation References

**Stylus Documentation**:
- See Stylus ai-knowledge-base for appliance vs agent mode architecture

**Container Runtime**:
- containerd: https://github.com/containerd/containerd
- spectro-containerd: Spectro Cloud fork with enhanced features
