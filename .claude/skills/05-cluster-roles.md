# kubeadm Cluster Roles

## Overview

Provider-kubeadm supports three distinct cluster roles for edge cluster configuration: **init**, **controlplane**, and **worker**. Each role has different responsibilities and generates different kubeadm configurations.

## Key Concepts

### Cluster Roles

**init** - Bootstrap first control plane node:
- Initializes embedded etcd cluster
- Creates cluster certificates and keys
- Generates bootstrap tokens for node joining
- Runs `kubeadm init`

**controlplane** - Additional control plane nodes (HA):
- Joins existing etcd cluster
- Replicates control plane components
- Uses certificate key from init node
- Runs `kubeadm join --control-plane`

**worker** - Worker nodes (compute):
- Runs kubelet and container runtime only
- No control plane components or etcd
- Joins cluster using bootstrap token
- Runs `kubeadm join`

### Role Configuration Flow

```
Init Node (Role: init)
  ↓
  kubeadm init --config kubeadm.yaml
  ↓
  Cluster Created, Tokens Generated
  ↓
  ┌─────────────────┬──────────────────┐
  ↓                 ↓                  ↓
ControlPlane     ControlPlane       Worker
(Role: controlplane)  (Role: controlplane)  (Role: worker)
  ↓                 ↓                  ↓
  kubeadm join      kubeadm join       kubeadm join
  --control-plane   --control-plane    (worker)
```

## Implementation Patterns

### Pattern 1: Init Node (Bootstrap First Control Plane)

**Use Case**: First control plane node that creates the cluster.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"   # This node's IP
  role: init                         # ← Init role
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    kubernetesVersion: v1.28.0
    controlPlaneEndpoint: "10.0.1.100:6443"
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated `/opt/kubeadm/kubeadm.yaml`**:
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.0123456789abcdef
  ttl: 24h0m0s                       # Token expires in 24 hours
  usages:
  - signing
  - authentication
nodeRegistration:
  name: control-1
  criSocket: unix:///run/containerd/containerd.sock
  taints: []                         # Remove NoSchedule taint (allow pods on control plane)
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.28.0
clusterName: edge-cluster
controlPlaneEndpoint: "10.0.1.100:6443"
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
etcd:
  local:
    dataDir: /var/lib/etcd            # Embedded etcd data directory
```

**Execution**:
```bash
# Provider generates kube-init.sh
kubeadm init --config /opt/kubeadm/kubeadm.yaml \
  --upload-certs \                   # Upload certs for control plane join
  --ignore-preflight-errors=NumCPU \
  --ignore-preflight-errors=Mem \
  -v=5

# After init, save join command outputs:
# 1. Worker join command with token
# 2. Control plane join command with certificate-key
```

**Post-Init Actions**:
```bash
# Provider executes kube-post-init.sh

# 1. Setup kubeconfig
mkdir -p $HOME/.kube
cp -i /etc/kubernetes/admin.conf $HOME/.kube/config

# 2. Apply CNI (MANUAL STEP - see 02-cni-installation.md)
kubectl apply -f kube-flannel.yml

# 3. Verify control plane ready
kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   2m    v1.28.0
```

**Generated Outputs**:
- `/etc/kubernetes/admin.conf` - Admin kubeconfig
- `/etc/kubernetes/pki/*` - Cluster certificates
- `/var/lib/etcd/` - etcd database
- `/opt/kubeadm/certificate-key.txt` - Certificate key for control plane join

---

### Pattern 2: ControlPlane Node (Additional Control Plane for HA)

**Use Case**: Additional control plane nodes for high availability.

**Prerequisites**:
- Init node must have completed successfully
- Certificate key from init node required

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"        # Init node IP
  certificate_key: "abc123def456..."      # From init node
  role: controlplane                      # ← ControlPlane role
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated `/opt/kubeadm/kubeadm-join.yaml`**:
```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
discovery:
  bootstrapToken:
    token: abcdef.0123456789abcdef
    apiServerEndpoint: "10.0.1.100:6443"  # Init node endpoint
    caCertHashes:
    - "sha256:1234567890abcdef..."         # CA cert hash from init
  timeout: 5m0s
nodeRegistration:
  name: control-2
  criSocket: unix:///run/containerd/containerd.sock
  taints: []
controlPlane:                              # ← Control plane specific config
  localAPIEndpoint:
    advertiseAddress: 10.0.1.101          # This node's IP
    bindPort: 6443
  certificateKey: "abc123def456..."        # From init node
```

**Execution**:
```bash
# Provider generates kube-join.sh
kubeadm join 10.0.1.100:6443 \
  --config /opt/kubeadm/kubeadm-join.yaml \
  --control-plane \                      # ← Control plane flag
  --certificate-key abc123def456... \
  -v=5
```

**Result**:
- Node joins as control plane
- Etcd member added to cluster
- Control plane components replicated
- API server reachable on this node's IP

**Verify HA Setup**:
```bash
kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   10m   v1.28.0
# control-2     Ready    control-plane   2m    v1.28.0

kubectl get pods -n kube-system -l component=etcd
# NAME                READY   STATUS    RESTARTS   AGE
# etcd-control-1      1/1     Running   0          10m
# etcd-control-2      1/1     Running   0          2m

# Test API server failover
curl -k https://10.0.1.101:6443/healthz
# ok  ← API server responding on control-2
```

---

### Pattern 3: Worker Node

**Use Case**: Compute nodes running application pods.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: worker                            # ← Worker role
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: JoinConfiguration
    nodeRegistration:
      name: worker-1
      kubeletExtraArgs:
        node-labels: "node-role.kubernetes.io/worker="
```

**Generated `/opt/kubeadm/kubeadm-join.yaml`**:
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
# No controlPlane section - worker only
```

**Execution**:
```bash
# Provider generates kube-join.sh
kubeadm join 10.0.1.100:6443 \
  --config /opt/kubeadm/kubeadm-join.yaml \
  -v=5
# No --control-plane flag - worker node
```

**Result**:
- Node joins as worker
- Only kubelet and container runtime running
- No control plane components
- No etcd

**Verify Worker Join**:
```bash
kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   15m   v1.28.0
# worker-1      Ready    worker          1m    v1.28.0

kubectl describe node worker-1
# Roles:              worker
# Taints:             <none>
# Unschedulable:      false
# ← Can schedule pods
```

---

## Role Comparison

| Aspect | Init | ControlPlane | Worker |
|--------|------|--------------|--------|
| **Command** | `kubeadm init` | `kubeadm join --control-plane` | `kubeadm join` |
| **etcd** | Creates embedded etcd | Joins etcd cluster | No etcd |
| **API Server** | Yes | Yes | No |
| **Scheduler** | Yes | Yes | No |
| **Controller Manager** | Yes | Yes | No |
| **Kubelet** | Yes | Yes | Yes |
| **Kube-proxy** | Yes | Yes | Yes |
| **Can Schedule Pods** | Yes (if untainted) | Yes (if untainted) | Yes |
| **Tokens Required** | Generates tokens | Bootstrap token + certificate key | Bootstrap token only |
| **Configuration** | InitConfiguration + ClusterConfiguration | JoinConfiguration + controlPlane section | JoinConfiguration only |

---

## Token Management

### Bootstrap Token

**Generated by init node**:
```yaml
bootstrapTokens:
- token: abcdef.0123456789abcdef
  ttl: 24h0m0s                        # Default: 24 hours
  usages:
  - signing
  - authentication
```

**Token Expiry**: Tokens expire after 24 hours by default.

**Create New Token** (if expired):
```bash
# On init/control plane node
kubeadm token create
# abcdef.0123456789abcdef

# Get CA cert hash
openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | \
  openssl rsa -pubin -outform der 2>/dev/null | \
  openssl dgst -sha256 -hex | sed 's/^.* //'
# sha256:1234567890abcdef...
```

### Certificate Key

**Generated by init node** (for control plane join):
```bash
kubeadm init phase upload-certs --upload-certs
# [upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
# [upload-certs] Using certificate key:
# abc123def456...
```

**Certificate Key Expiry**: 2 hours by default.

**Renew Certificate Key** (if expired):
```bash
kubeadm init phase upload-certs --upload-certs
# New certificate key generated
```

---

## Common Pitfalls

### ❌ WRONG: Using worker role for first node

```yaml
cluster:
  role: worker    # ❌ WRONG - first node must be init
```

**Result**: kubeadm join fails - no cluster exists yet
```bash
kubeadm join
# error: unable to connect to apiServerEndpoint
# ❌ No API server running - no cluster initialized
```

### ✅ CORRECT: Use init role for first node

```yaml
cluster:
  role: init      # ✅ CORRECT - first node creates cluster
```

---

### ❌ WRONG: Missing certificate-key for control plane join

```yaml
cluster:
  role: controlplane
  # ❌ Missing certificate_key
```

**Result**: Control plane join fails
```bash
kubeadm join --control-plane
# error: unable to fetch the kubeadm-certs Secret
# ❌ Certificate key required for control plane join
```

### ✅ CORRECT: Provide certificate-key

```yaml
cluster:
  role: controlplane
  certificate_key: "abc123def456..."   # ✅ From init node
```

---

### ❌ WRONG: Expired bootstrap token

```yaml
# Init node created 25 hours ago
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  role: worker
```

**Result**: Join fails with token error
```bash
kubeadm join
# error: unable to authenticate: token id "abcdef" expired
# ❌ Token expired after 24 hours
```

### ✅ CORRECT: Generate new token

```bash
# On control plane node
kubeadm token create --ttl 0  # Never expires
# ghijkl.0123456789ghijkl
```

```yaml
cluster:
  cluster_token: "ghijkl.0123456789ghijkl"   # ✅ Fresh token
  role: worker
```

---

## Integration Points

### With Provider

- Provider detects role from `cluster.role` in cloud-init
- Generates role-specific kubeadm YAML files
- Creates role-specific shell scripts (kube-init.sh vs kube-join.sh)

### With Stylus

- **Appliance mode**: All roles supported (init, controlplane, worker)
- **Agent mode**: All roles supported with STYLUS_ROOT

### With Proxy

- Proxy configuration applies to all roles
- NO_PROXY calculation same for all roles

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-kubeadm/pkg/provider/provider.go` - Role detection and handling
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-init.sh` - Init role execution
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-join.sh` - ControlPlane/Worker role execution

## Related Skills

- See `provider-kubeadm:01-architecture` for role overview
- See `provider-kubeadm:03-configuration-patterns` for role-specific configurations
- See `provider-kubeadm:08-troubleshooting` for role-specific issues

**Related Provider Skills**:
- See `provider-k3s:03-cluster-roles` for k3s roles (similar concepts)
- See `provider-rke2:03-cluster-roles` for rke2 roles (similar concepts)

## Documentation References

**kubeadm Commands**:
- Init: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/
- Join: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-join/
- Tokens: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-token/

**HA Setup**:
- https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/
