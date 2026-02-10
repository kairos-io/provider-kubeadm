---
skill: kubeadm CNI Installation Guide
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

# kubeadm CNI Installation Guide

## Overview

**CRITICAL**: Unlike K3s/RKE2, kubeadm does NOT include a Container Network Interface (CNI). After running `kubeadm init`, you MUST manually install a CNI plugin or all nodes will remain in `NotReady` state with `NetworkPluginNotReady` error.

This skill provides step-by-step installation guides for the three most common CNI options used with kubeadm on Palette Edge clusters.

## Key Concepts

### What is a CNI?

A Container Network Interface (CNI) plugin provides pod-to-pod networking in Kubernetes:

- **Pod IP Assignment**: Allocates unique IPs to pods
- **Pod Communication**: Enables pods to communicate across nodes
- **Network Policies**: (Optional) Enforces network isolation rules
- **Service Discovery**: Integrates with kube-proxy for service routing

### Why kubeadm Requires Manual CNI

kubeadm follows the upstream Kubernetes philosophy of remaining CNI-agnostic:

- ✅ **Flexibility**: Choose the CNI that fits your requirements
- ✅ **No Lock-in**: Switch CNIs without changing Kubernetes distribution
- ❌ **Extra Step**: Must manually install CNI after cluster init
- ❌ **NotReady State**: Nodes stuck until CNI installed

### CNI Selection Criteria

| Criterion | Flannel (Recommended) | Calico | Cilium |
|-----------|----------------------|--------|---------|
| **Ease of Setup** | ⭐⭐⭐⭐⭐ Simplest | ⭐⭐⭐ Moderate | ⭐⭐ Complex |
| **NetworkPolicy** | ❌ No | ✅ Yes | ✅ Yes (eBPF) |
| **Overlay Type** | VXLAN (port 8472) | BGP routing or VXLAN | VXLAN or native routing |
| **Resource Usage** | Low | Medium | Medium-High |
| **Edge Suitability** | ✅ Best for simple edge | ✅ Good for enterprise | ⚠️ Requires kernel 4.9+ |
| **Debugging** | Easy | Moderate | Complex |

**Recommendation**: Use **Flannel** for most Palette Edge deployments unless you need NetworkPolicy or advanced networking features.

## Implementation Patterns

### Pattern 1: Flannel CNI Installation (Recommended)

**Use Case**: Simple overlay networking for edge clusters without NetworkPolicy requirements.

#### Step 1: Complete kubeadm init

```bash
# After kubeadm init completes, verify control plane is running
kubectl get nodes
# NAME          STATUS     ROLES           AGE   VERSION
# control-1     NotReady   control-plane   1m    v1.28.0
# ⚠️  NotReady - no CNI installed yet
```

#### Step 2: Install Flannel

```bash
# Apply Flannel manifest
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Wait for Flannel pods to start
kubectl get pods -n kube-flannel -w
# NAME                    READY   STATUS    RESTARTS   AGE
# kube-flannel-ds-amd64   1/1     Running   0          30s
```

#### Step 3: Verify nodes become Ready

```bash
# Check node status
kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   3m    v1.28.0
# ✅ Node is Ready after Flannel installation
```

#### Step 4: Test pod networking

```bash
# Create test pods
kubectl run test-1 --image=nginx
kubectl run test-2 --image=nginx

# Verify pods get IPs from podSubnet (10.244.0.0/16)
kubectl get pods -o wide
# NAME     READY   STATUS    IP            NODE
# test-1   1/1     Running   10.244.0.5    control-1
# test-2   1/1     Running   10.244.0.6    control-1

# Test pod-to-pod connectivity
kubectl exec test-1 -- ping -c 3 10.244.0.6
# ✅ Should succeed
```

#### Flannel Configuration

**Default Flannel Settings**:
```yaml
# kube-flannel.yml excerpt
net-conf.json: |
  {
    "Network": "10.244.0.0/16",   # Must match kubeadm podSubnet
    "Backend": {
      "Type": "vxlan",             # VXLAN overlay
      "Port": 8472                 # VXLAN port (UDP)
    }
  }
```

**Custom Flannel Configuration** (if using non-default podSubnet):

```bash
# Download manifest
curl -sO https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Edit to match your podSubnet
# Change "Network": "10.244.0.0/16" to your podSubnet value
vi kube-flannel.yml

# Apply modified manifest
kubectl apply -f kube-flannel.yml
```

**Firewall Requirements**:
- **VXLAN**: UDP port 8472 (must be open between nodes)

---

### Pattern 2: Calico CNI Installation (Advanced)

**Use Case**: Enterprise edge clusters requiring NetworkPolicy enforcement or BGP routing.

#### Step 1: Install Calico Operator

```bash
# Install Tigera Calico operator
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/tigera-operator.yaml

# Verify operator is running
kubectl get pods -n tigera-operator
# NAME                               READY   STATUS    RESTARTS   AGE
# tigera-operator-5fb55776df-abc123  1/1     Running   0          30s
```

#### Step 2: Create Calico Installation CR

```yaml
# calico-installation.yaml
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  # Must match kubeadm podSubnet
  calicoNetwork:
    ipPools:
    - blockSize: 26
      cidr: 10.244.0.0/16        # Match kubeadm podSubnet
      encapsulation: VXLANCrossSubnet
      natOutgoing: Enabled
      nodeSelector: all()
```

```bash
# Apply Calico configuration
kubectl apply -f calico-installation.yaml

# Wait for Calico pods
kubectl get pods -n calico-system -w
# NAME                               READY   STATUS    RESTARTS   AGE
# calico-kube-controllers-...        1/1     Running   0          1m
# calico-node-...                    1/1     Running   0          1m
# calico-typha-...                   1/1     Running   0          1m
```

#### Step 3: Verify installation

```bash
# Check nodes become Ready
kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# control-1     Ready    control-plane   5m    v1.28.0

# Verify Calico is active
kubectl get installation default -o yaml
# status:
#   calicoVersion: v3.26.1
#   variant: Calico
```

#### NetworkPolicy Example

**With Calico, you can enforce network policies**:

```yaml
# deny-all-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
```

```bash
kubectl apply -f deny-all-ingress.yaml

# Now pods cannot receive traffic (unless explicitly allowed)
# Flannel does NOT support NetworkPolicy - policies would be ignored
```

**Firewall Requirements**:
- **BGP**: TCP port 179 (if using BGP mode)
- **VXLAN**: UDP port 4789 (if using VXLAN mode)
- **Typha**: TCP port 5473 (for Calico components)

---

### Pattern 3: Cilium CNI Installation (eBPF-based)

**Use Case**: Advanced networking with eBPF observability and security features.

#### Step 1: Install Cilium CLI

```bash
# Download Cilium CLI
curl -L --remote-name-all https://github.com/cilium/cilium-cli/releases/latest/download/cilium-linux-amd64.tar.gz
tar xzvf cilium-linux-amd64.tar.gz
sudo mv cilium /usr/local/bin/

# Verify installation
cilium version --client
```

#### Step 2: Install Cilium CNI

```bash
# Install Cilium with Helm
cilium install --version 1.14.0 \
  --set ipam.operator.clusterPoolIPv4PodCIDRList="{10.244.0.0/16}" \
  --set kubeProxyReplacement=false

# Wait for Cilium pods
kubectl get pods -n kube-system -l k8s-app=cilium -w
# NAME           READY   STATUS    RESTARTS   AGE
# cilium-abc123  1/1     Running   0          1m
```

#### Step 3: Verify connectivity

```bash
# Run Cilium connectivity test
cilium connectivity test

# Check nodes become Ready
kubectl get nodes
# NAME          STATUS   READY           AGE   VERSION
# control-1     Ready    control-plane   7m    v1.28.0
```

**Kernel Requirements**:
- **Minimum**: Linux kernel 4.9+
- **Recommended**: Linux kernel 5.4+ (for full eBPF features)

**Firewall Requirements**:
- **VXLAN**: UDP port 8472 (if using VXLAN mode)
- **Health checks**: TCP port 4240

---

## Common Pitfalls

### ❌ WRONG: Forgetting to install CNI

```bash
kubeadm init --config kubeadm.yaml

# Wait for cluster to be ready
sleep 60

kubectl get nodes
# NAME     STATUS     ROLES           AGE   VERSION
# node-1   NotReady   control-plane   1m    v1.28.0
# ❌ Node stuck in NotReady forever - NO CNI INSTALLED!

kubectl get pods -A
# kube-system   coredns-...   0/1   Pending   0   1m
# ❌ CoreDNS stuck in Pending - waiting for network
```

### ✅ CORRECT: Install CNI immediately after kubeadm init

```bash
kubeadm init --config kubeadm.yaml

# Install CNI immediately (Flannel example)
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Wait for CNI pods
kubectl wait --for=condition=Ready pods -l app=flannel -n kube-flannel --timeout=120s

# Now nodes become Ready
kubectl get nodes
# NAME     STATUS   ROLES           AGE   VERSION
# node-1   Ready    control-plane   2m    v1.28.0
# ✅ Node is Ready
```

### ❌ WRONG: CNI subnet mismatch

```yaml
# kubeadm.yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
networking:
  podSubnet: 10.244.0.0/16     # kubeadm configuration

# Flannel manifest has different subnet
# kube-flannel.yml
net-conf.json: |
  {
    "Network": "10.100.0.0/16"  # ❌ Mismatch!
  }
```

```bash
kubectl apply -f kube-flannel.yml

# Pods get IPs from wrong subnet
kubectl get pods -o wide
# NAME     IP            NODE
# test-1   10.100.0.5    node-1   # ❌ Wrong subnet!

# Connectivity fails
kubectl exec test-1 -- ping 10.96.0.1  # Service IP
# ❌ Fails - routing broken due to subnet mismatch
```

### ✅ CORRECT: Match CNI subnet to kubeadm podSubnet

```yaml
# kubeadm.yaml
networking:
  podSubnet: 10.244.0.0/16     # kubeadm configuration

# Flannel manifest matches
net-conf.json: |
  {
    "Network": "10.244.0.0/16"  # ✅ Match!
  }
```

### ❌ WRONG: Installing multiple CNIs

```bash
# Install Flannel
kubectl apply -f kube-flannel.yml

# Also install Calico (trying to "add" NetworkPolicy support)
kubectl create -f tigera-operator.yaml
kubectl apply -f calico-installation.yaml

# ❌ Conflict! Pods get multiple IPs, routing breaks
kubectl get pods -o wide
# NAME     IP                    NODE
# test-1   10.244.0.5,10.245.0.3 node-1   # ❌ TWO IPs!
```

### ✅ CORRECT: Install only ONE CNI

```bash
# Choose Calico if you need NetworkPolicy
kubectl create -f tigera-operator.yaml
kubectl apply -f calico-installation.yaml

# ✅ Single CNI, single IP per pod
kubectl get pods -o wide
# NAME     IP            NODE
# test-1   10.244.0.5    node-1   # ✅ One IP
```

## Troubleshooting

### Issue: Nodes stuck in NotReady

**Symptom**:
```bash
kubectl get nodes
# NAME     STATUS     ROLES           AGE   VERSION
# node-1   NotReady   control-plane   5m    v1.28.0
```

**Diagnosis**:
```bash
kubectl describe node node-1
# ...
# Ready            False   NetworkPluginNotReady   container runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: cni plugin not initialized
```

**Solution**: Install a CNI (see patterns above)

---

### Issue: CNI pods CrashLoopBackOff

**Symptom**:
```bash
kubectl get pods -n kube-flannel
# NAME                    READY   STATUS             RESTARTS   AGE
# kube-flannel-ds-amd64   0/1     CrashLoopBackOff   5          3m
```

**Diagnosis**:
```bash
kubectl logs -n kube-flannel kube-flannel-ds-amd64
# Error: subnet "10.244.0.0/16" does not contain PodCIDR "10.100.0.0/16"
# ❌ Subnet mismatch between kubeadm config and Flannel config
```

**Solution**: Edit Flannel manifest to match kubeadm podSubnet

---

### Issue: Pods cannot reach services

**Symptom**:
```bash
kubectl run test --image=nginx
kubectl exec test -- curl kubernetes.default.svc.cluster.local
# curl: (6) Could not resolve host: kubernetes.default.svc.cluster.local
```

**Diagnosis**:
```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns
# NAME          READY   STATUS    RESTARTS   AGE
# coredns-...   0/1     Pending   0          10m
# ❌ CoreDNS stuck in Pending - waiting for CNI
```

**Solution**: Ensure CNI is installed and running before deploying workloads

---

### Issue: Firewall blocking VXLAN

**Symptom**:
```bash
# From node-1
ping 10.244.1.5  # Pod on node-2
# ❌ No response
```

**Diagnosis**:
```bash
# Check VXLAN interface
ip -d link show flannel.1
# flannel.1@NONE: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450
#     vxlan id 1 local 192.168.1.10 dev eth0 srcport 0 0 dstport 8472

# Test VXLAN connectivity
nc -u -v -z 192.168.1.20 8472
# ❌ Connection refused - firewall blocking UDP 8472
```

**Solution**: Open UDP port 8472 between nodes
```bash
# Firewalld
firewall-cmd --permanent --add-port=8472/udp
firewall-cmd --reload

# iptables
iptables -A INPUT -p udp --dport 8472 -j ACCEPT
```

## Integration Points

### With kubeadm

- CNI installed AFTER `kubeadm init` completes
- CNI subnet MUST match kubeadm `networking.podSubnet`
- kubeadm does NOT manage CNI lifecycle (manual upgrades)

### With Stylus

- CNI manifest can be pre-staged in cloud-init
- Appliance mode: Apply CNI via kubectl after init
- Agent mode: Palette can deploy CNI via cluster profile packs

### With Proxy

- CNI pods must respect HTTP_PROXY settings
- Ensure pod/service subnets in NO_PROXY
- CNI image pull may require proxy configuration

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-post-init.sh` - Post-init hook (CNI can be installed here)
- `/Users/rishi/work/src/provider-kubeadm/scripts/import.sh` - Image import (pre-load CNI images for air-gap)

**CNI Manifests** (download and customize):
- Flannel: https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
- Calico: https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/tigera-operator.yaml
- Cilium: Use `cilium install` CLI

## Related Skills

- See `provider-kubeadm:01-architecture` for kubeadm overview and CNI requirement
- See `provider-kubeadm:03-configuration-patterns` for podSubnet configuration
- See `provider-kubeadm:07-networking` for detailed CNI comparison and networking architecture
- See `provider-kubeadm:08-troubleshooting` for NetworkPluginNotReady and CNI troubleshooting

**Related Provider Skills**:
- See `provider-k3s:01-architecture` - K3s includes Flannel by default
- See `provider-rke2:05-networking` - RKE2 includes Canal CNI by default

## Documentation References

**CNI Plugin Documentation**:
- Flannel: https://github.com/flannel-io/flannel#flannel
- Calico: https://docs.tigera.io/calico/latest/getting-started/kubernetes/
- Cilium: https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/

**Kubernetes CNI Documentation**:
- https://kubernetes.io/docs/concepts/cluster-administration/addons/#networking-and-network-policy
- https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/
