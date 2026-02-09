# kubeadm Version Compatibility

## Overview

Provider-kubeadm supports dual kubeadm API versions for compatibility across Kubernetes versions: **v1beta3** (Kubernetes < 1.31) and **v1beta4** (Kubernetes >= 1.31). This skill covers API version selection, migration patterns, and version-specific configurations.

## Key Concepts

### kubeadm API Versions

**v1beta3** (Current, < K8s 1.31):
- Introduced in Kubernetes 1.22
- Default for Kubernetes 1.22-1.30
- Stable and widely used
- Will be deprecated in future K8s versions

**v1beta4** (Future, >= K8s 1.31):
- Introduced in Kubernetes 1.31
- Required for Kubernetes 1.31+
- New fields and validation
- Migration path from v1beta3

### Version Detection

Provider-kubeadm auto-detects API version from user configuration:

```go
func determineAPIVersion(config string) string {
    if strings.Contains(config, "kubeadm.k8s.io/v1beta4") {
        return "v1beta4"
    }
    return "v1beta3"  // Default
}
```

### KUBEADM_VERSION Build Argument

The kubeadm binary version is controlled via Docker build argument:

```dockerfile
# Dockerfile
ARG KUBEADM_VERSION=v1.28.0
RUN curl -L https://dl.k8s.io/release/${KUBEADM_VERSION}/bin/linux/amd64/kubeadm -o /usr/local/bin/kubeadm
```

**Provider container image tags**:
- `provider-kubeadm:v1.28` - kubeadm 1.28.x
- `provider-kubeadm:v1.29` - kubeadm 1.29.x
- `provider-kubeadm:v1.30` - kubeadm 1.30.x
- `provider-kubeadm:v1.31` - kubeadm 1.31.x (requires v1beta4)

## Implementation Patterns

### Pattern 1: v1beta3 Configuration (K8s < 1.31)

**Use Case**: Kubernetes 1.22-1.30 clusters.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: init
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3    # ← v1beta3
    kind: ClusterConfiguration
    kubernetesVersion: v1.28.0
    controlPlaneEndpoint: "10.0.1.100:6443"
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
      dnsDomain: cluster.local
    apiServer:
      certSANs:
      - "10.0.1.100"
      extraArgs:
        authorization-mode: "Node,RBAC"
```

**Generated kubeadm.yaml**:
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.0123456789abcdef
  ttl: 24h0m0s
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.28.0
# ... rest of config
```

---

### Pattern 2: v1beta4 Configuration (K8s >= 1.31)

**Use Case**: Kubernetes 1.31+ clusters.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: init
  config: |
    apiVersion: kubeadm.k8s.io/v1beta4    # ← v1beta4
    kind: ClusterConfiguration
    kubernetesVersion: v1.31.0
    controlPlaneEndpoint: "10.0.1.100:6443"
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
      dnsDomain: cluster.local
    apiServer:
      certSANs:
      - "10.0.1.100"
      extraArgs:
        authorization-mode: "Node,RBAC"
```

**Generated kubeadm.yaml**:
```yaml
apiVersion: kubeadm.k8s.io/v1beta4    # ← v1beta4
kind: InitConfiguration
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.0123456789abcdef
  ttl: 24h0m0s
---
apiVersion: kubeadm.k8s.io/v1beta4    # ← v1beta4
kind: ClusterConfiguration
kubernetesVersion: v1.31.0
# ... rest of config
```

---

### Pattern 3: Version Mismatch Handling

**Use Case**: Detect and prevent version mismatches.

**Mismatch Scenario 1**: v1beta4 with K8s < 1.31
```yaml
cluster:
  config: |
    apiVersion: kubeadm.k8s.io/v1beta4    # ❌ Requires K8s >= 1.31
    kind: ClusterConfiguration
    kubernetesVersion: v1.28.0            # ❌ Too old for v1beta4
```

**Result**: kubeadm init fails
```bash
kubeadm init --config kubeadm.yaml
# error: unknown API version "kubeadm.k8s.io/v1beta4"
# ❌ kubeadm 1.28 doesn't support v1beta4
```

**Solution**: Use v1beta3 for K8s < 1.31

---

**Mismatch Scenario 2**: v1beta3 with K8s >= 1.32
```yaml
cluster:
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3    # ⚠️ Deprecated in K8s 1.32+
    kind: ClusterConfiguration
    kubernetesVersion: v1.32.0
```

**Result**: kubeadm init succeeds with warning
```bash
kubeadm init --config kubeadm.yaml
# [WARNING]: kubeadm.k8s.io/v1beta3 is deprecated, use v1beta4
# ⚠️ Works but with deprecation warning
```

**Solution**: Migrate to v1beta4 for K8s >= 1.31

---

### Pattern 4: Mixed API Versions (Init vs Join)

**Use Case**: Existing cluster using v1beta3, new nodes joining.

**Init Node** (already deployed):
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
```

**Join Configuration** (must match):
```yaml
#cloud-config
cluster:
  role: worker
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3    # ✅ Must match init node
    kind: JoinConfiguration
```

**Important**: Join configuration API version must match cluster API version.

---

### Pattern 5: Kubernetes Version Constraints

**Use Case**: Ensure kubeadm version matches Kubernetes version.

**Version Matching Rules**:
- kubeadm version should match (or be 1 minor version newer than) Kubernetes version
- **Supported**: kubeadm 1.28.x to init K8s 1.28.x cluster
- **Supported**: kubeadm 1.29.x to init K8s 1.28.x cluster (newer kubeadm)
- **Unsupported**: kubeadm 1.28.x to init K8s 1.30.x cluster (too old kubeadm)

**Build with Specific Version**:
```dockerfile
# Build provider-kubeadm image for K8s 1.28
ARG KUBEADM_VERSION=v1.28.5
RUN curl -L https://dl.k8s.io/release/${KUBEADM_VERSION}/bin/linux/amd64/kubeadm -o /usr/local/bin/kubeadm
```

---

## API Version Differences

### v1beta3 vs v1beta4 Changes

| Feature | v1beta3 | v1beta4 | Notes |
|---------|---------|---------|-------|
| **API Version** | `kubeadm.k8s.io/v1beta3` | `kubeadm.k8s.io/v1beta4` | Required field |
| **K8s Support** | 1.22-1.30 | 1.31+ | Backward compatible |
| **Deprecation** | Stable | Future standard | v1beta3 deprecated in K8s 1.32+ |
| **New Fields** | Standard fields | Enhanced validation | v1beta4 adds stricter validation |
| **CRI Socket** | `criSocket` in nodeRegistration | Same | No change |
| **Networking** | `podSubnet`, `serviceSubnet` | Same | No change |

**Most Configurations Are Compatible**: Provider-kubeadm mostly generates same YAML for both versions.

### Migration Example

**v1beta3 Configuration**:
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.28.0
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
```

**v1beta4 Migration** (minimal changes):
```yaml
apiVersion: kubeadm.k8s.io/v1beta4    # ← Change API version
kind: ClusterConfiguration
kubernetesVersion: v1.31.0            # ← Update K8s version
networking:
  podSubnet: 10.244.0.0/16            # ← Same
  serviceSubnet: 10.96.0.0/12         # ← Same
```

---

## Common Pitfalls

### ❌ WRONG: Using v1beta4 with old kubeadm

```yaml
# kubeadm 1.28 installed
apiVersion: kubeadm.k8s.io/v1beta4
```

**Result**: kubeadm init fails
```bash
kubeadm init
# error: unknown API version "kubeadm.k8s.io/v1beta4"
```

### ✅ CORRECT: Match API version to kubeadm version

```yaml
# kubeadm 1.28
apiVersion: kubeadm.k8s.io/v1beta3    # ✅ Supported

# kubeadm 1.31+
apiVersion: kubeadm.k8s.io/v1beta4    # ✅ Supported
```

---

### ❌ WRONG: Mixing API versions (init vs join)

```yaml
# Init node
apiVersion: kubeadm.k8s.io/v1beta3

# Join node
apiVersion: kubeadm.k8s.io/v1beta4    # ❌ Mismatch!
```

**Result**: Join may fail or behave unpredictably

### ✅ CORRECT: Consistent API versions

```yaml
# All nodes use same API version
apiVersion: kubeadm.k8s.io/v1beta3
```

---

### ❌ WRONG: Old kubeadm for new K8s

```bash
# kubeadm 1.28 installed
kubernetesVersion: v1.30.0    # ❌ Too new for kubeadm 1.28
```

**Result**: May succeed but untested, unsupported

### ✅ CORRECT: Match kubeadm and K8s versions

```bash
# kubeadm 1.28
kubernetesVersion: v1.28.5    # ✅ Matches

# OR use newer kubeadm
# kubeadm 1.30
kubernetesVersion: v1.28.5    # ✅ Backward compatible
```

---

## Version Selection Guide

### For New Clusters

**Kubernetes 1.22-1.30**: Use v1beta3
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kubernetesVersion: v1.28.0
```

**Kubernetes 1.31+**: Use v1beta4
```yaml
apiVersion: kubeadm.k8s.io/v1beta4
kubernetesVersion: v1.31.0
```

### For Existing Clusters

**Upgrading from K8s 1.30 to 1.31**:
1. Keep v1beta3 during upgrade (works with warning)
2. Migrate to v1beta4 after upgrade complete

**Adding Nodes to Existing Cluster**:
- Match API version of existing cluster
- Check API version with: `kubectl get cm -n kube-system kubeadm-config -o yaml`

---

## Integration Points

### With Provider

- Provider auto-detects API version from user config
- Generates appropriate kubeadm YAML
- No provider code changes needed between versions

### With kubeadm Binary

- Provider container image includes specific kubeadm version
- Use `KUBEADM_VERSION` build arg to control binary version
- Match kubeadm binary to target Kubernetes version

### With Kubernetes Releases

- Stay within kubeadm version skew policy (±1 minor version)
- Test new API versions before production use
- Monitor kubeadm deprecation announcements

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-kubeadm/pkg/provider/provider.go` - API version detection
- `/Users/rishi/work/src/provider-kubeadm/Dockerfile` - KUBEADM_VERSION build arg

**Kubernetes Version Matrix**:
| K8s Version | Recommended API | kubeadm Version | Status |
|-------------|----------------|-----------------|--------|
| 1.22-1.27 | v1beta3 | Match K8s version | Supported |
| 1.28-1.30 | v1beta3 | Match K8s version | Supported |
| 1.31+ | v1beta4 | Match K8s version | Required |
| 1.32+ | v1beta4 | Match K8s version | v1beta3 deprecated |

## Related Skills

- See `provider-kubeadm:01-architecture` for kubeadm overview
- See `provider-kubeadm:03-configuration-patterns` for configuration examples
- See `provider-kubeadm:08-troubleshooting` for version mismatch errors

## Documentation References

**kubeadm API Documentation**:
- v1beta3: https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta3/
- v1beta4: https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta4/

**Version Skew Policy**:
- https://kubernetes.io/releases/version-skew-policy/#kubeadm

**kubeadm Releases**:
- https://kubernetes.io/releases/
