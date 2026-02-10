---
skill: kubeadm Networking
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

# kubeadm Networking

## Overview

Provider-kubeadm networking architecture differs significantly from K3s/RKE2: **NO built-in CNI**. This skill covers network configuration, CNI selection, firewall requirements, and the interaction between Stylus overlay networking and CNI pod networking.

## Key Concepts

### No Built-in CNI

**Critical Difference from K3s/RKE2**:
- **K3s**: Includes Flannel CNI by default
- **RKE2**: Includes Canal CNI (Calico + Flannel) by default
- **kubeadm**: **NO CNI included** - manual installation required

**Implications**:
1. After `kubeadm init`, nodes stuck in `NotReady`
2. CoreDNS pods stuck in `Pending`
3. CNI must be installed manually (see `02-cni-installation.md`)

### Network Layers

**Provider-kubeadm networking has two separate layers**:

1. **Node-to-Node (Host Networking)**:
   - Physical network interfaces (eth0, eth1)
   - Optional: Stylus overlay (VxLAN for multi-site)
   - Control plane communication (API server port 6443)

2. **Pod-to-Pod (CNI Networking)**:
   - CNI plugin (Flannel/Calico/Cilium)
   - Pod IP assignment from podSubnet
   - Service IP assignment from serviceSubnet
   - Overlay networking (CNI VXLAN, separate from Stylus overlay)

### Stylus Overlay vs CNI Overlay

**Important**: These are TWO SEPARATE overlay networks:

**Stylus Overlay** (Host-to-Host):
- **Purpose**: Connect edge hosts across different sites/networks
- **Interface**: `scbr0` (bridge), `scvxlan-*` (tunnels)
- **Port**: UDP 4789 (VXLAN)
- **Scope**: Host networking layer
- **Example**: Edge host in Site A can reach edge host in Site B

**CNI Overlay** (Pod-to-Pod):
- **Purpose**: Connect pods running on different nodes
- **Interface**: CNI-specific (e.g., `flannel.1`, `vxlan.calico`)
- **Port**: CNI-specific (Flannel: UDP 8472, Calico: UDP 4789)
- **Scope**: Kubernetes pod networking
- **Example**: Pod on node-1 can reach pod on node-2

**They Coexist** - Stylus overlay for hosts, CNI overlay for pods.

## Implementation Patterns

### Pattern 1: Network Configuration (podSubnet and serviceSubnet)

**Use Case**: Define IP ranges for pods and services.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16        # Pod IP range
      serviceSubnet: 10.96.0.0/12     # Service IP range
      dnsDomain: cluster.local        # DNS suffix
```

**Network Allocation**:
- **podSubnet**: Each node gets a /24 subnet (e.g., node-1: 10.244.0.0/24, node-2: 10.244.1.0/24)
- **serviceSubnet**: Service IPs allocated from this range
- **Node IPs**: Outside these ranges (e.g., 10.0.1.0/24)

**Example**:
```
Node Network: 10.0.1.0/24
  - node-1: 10.0.1.100
  - node-2: 10.0.1.101

Pod Network (podSubnet: 10.244.0.0/16):
  - node-1 pods: 10.244.0.0/24 (10.244.0.1-254)
  - node-2 pods: 10.244.1.0/24 (10.244.1.1-254)

Service Network (serviceSubnet: 10.96.0.0/12):
  - kubernetes.default.svc: 10.96.0.1
  - coredns.kube-system.svc: 10.96.0.10
```

---

### Pattern 2: Flannel CNI (VXLAN Overlay)

**Use Case**: Simple pod networking with VXLAN encapsulation.

**After kubeadm init, install Flannel**:
```bash
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

**Flannel Network Architecture**:
```
┌─────────────────────┐         ┌─────────────────────┐
│ Node 1              │         │ Node 2              │
│                     │         │                     │
│  ┌───────────────┐  │         │  ┌───────────────┐  │
│  │ Pod A         │  │         │  │ Pod B         │  │
│  │ 10.244.0.5    │  │         │  │ 10.244.1.5    │  │
│  └───────┬───────┘  │         │  └───────┬───────┘  │
│          │ veth     │         │          │ veth     │
│  ┌───────▼───────┐  │         │  ┌───────▼───────┐  │
│  │ cni0 bridge   │  │         │  │ cni0 bridge   │  │
│  │ 10.244.0.1    │  │         │  │ 10.244.1.1    │  │
│  └───────┬───────┘  │         │  └───────┬───────┘  │
│          │          │         │          │          │
│  ┌───────▼───────┐  │  VXLAN  │  ┌───────▼───────┐  │
│  │ flannel.1     │◄─┼─────────┼─►│ flannel.1     │  │
│  │ VXLAN tunnel  │  │ UDP 8472│  │ VXLAN tunnel  │  │
│  └───────┬───────┘  │         │  └───────┬───────┘  │
│          │          │         │          │          │
│  ┌───────▼───────┐  │         │  ┌───────▼───────┐  │
│  │ eth0          │◄─┼─────────┼─►│ eth0          │  │
│  │ 10.0.1.100    │  │         │  │ 10.0.1.101    │  │
│  └───────────────┘  │         │  └───────────────┘  │
└─────────────────────┘         └─────────────────────┘
```

**Flannel Configuration**:
```json
{
  "Network": "10.244.0.0/16",   // Must match podSubnet
  "Backend": {
    "Type": "vxlan",
    "Port": 8472                // UDP port for VXLAN
  }
}
```

**Firewall Requirements**:
- **UDP 8472**: VXLAN tunnel (node-to-node)

---

### Pattern 3: Calico CNI (BGP or VXLAN)

**Use Case**: Advanced networking with NetworkPolicy support.

**After kubeadm init, install Calico**:
```bash
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/tigera-operator.yaml

cat <<EOF | kubectl apply -f -
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    ipPools:
    - blockSize: 26
      cidr: 10.244.0.0/16       # Must match podSubnet
      encapsulation: VXLANCrossSubnet
      natOutgoing: Enabled
      nodeSelector: all()
EOF
```

**Calico Network Modes**:

**VXLAN Mode** (Overlay):
- Similar to Flannel
- UDP port 4789 (standard VXLAN port)
- Encapsulates pod traffic

**BGP Mode** (Native Routing):
- No encapsulation
- Uses BGP to advertise pod routes
- More efficient but requires BGP support in network
- TCP port 179 for BGP

**Firewall Requirements**:
- **VXLAN mode**: UDP 4789
- **BGP mode**: TCP 179
- **Typha** (Calico component): TCP 5473

---

### Pattern 4: Multi-Interface Nodes

**Use Case**: Nodes with multiple network interfaces (management + data).

**Cloud-init Configuration**:
```yaml
cluster:
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: InitConfiguration
    localAPIEndpoint:
      advertiseAddress: 10.0.1.100   # Management interface
      bindPort: 6443
    nodeRegistration:
      kubeletExtraArgs:
        node-ip: "10.0.1.100"        # Force kubelet to use management IP
    ---
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    controlPlaneEndpoint: "10.0.1.100:6443"
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Flannel Multi-Interface**:
```yaml
# Flannel manifest - specify interface
containers:
- name: kube-flannel
  args:
  - --ip-masq
  - --kube-subnet-mgr
  - --iface=eth1          # Data plane interface
```

**Calico Multi-Interface**:
```yaml
# Calico IPPool with specific interface
apiVersion: crd.projectcalico.org/v1
kind: IPPool
metadata:
  name: default-ipv4-ippool
spec:
  cidr: 10.244.0.0/16
  ipipMode: Never
  natOutgoing: true
  nodeSelector: all()
  vxlanMode: Always
  # Calico auto-detects interface, can override with:
  # felix: interface-regex=eth1
```

---

### Pattern 5: Stylus Overlay + CNI Coexistence

**Use Case**: Multi-site edge clusters using Stylus overlay for host connectivity + CNI for pod networking.

**Architecture**:
```
Site A                              Site B
┌────────────────────┐             ┌────────────────────┐
│ Node A             │             │ Node B             │
│                    │             │                    │
│ Stylus Overlay:    │  VXLAN      │ Stylus Overlay:    │
│ scbr0, scvxlan-*   │◄───────────►│ scbr0, scvxlan-*   │
│ UDP 4789           │             │ UDP 4789           │
│                    │             │                    │
│ CNI Overlay:       │  VXLAN      │ CNI Overlay:       │
│ flannel.1          │◄───────────►│ flannel.1          │
│ UDP 8472           │             │ UDP 8472           │
│                    │             │                    │
│ Pods:              │             │ Pods:              │
│ 10.244.0.x         │             │ 10.244.1.x         │
└────────────────────┘             └────────────────────┘
```

**Both overlays run simultaneously**:
1. **Stylus overlay (UDP 4789)**: Hosts communicate across sites
2. **Flannel overlay (UDP 8472)**: Pods communicate across nodes

**No Conflict** - Different ports and interfaces.

---

## Network Troubleshooting

### Issue: Pods cannot communicate

**Symptom**:
```bash
kubectl exec pod-1 -- ping 10.244.1.5
# No response
```

**Diagnosis**:
```bash
# Check CNI pods running
kubectl get pods -n kube-flannel
# NAME                    READY   STATUS    RESTARTS   AGE
# kube-flannel-ds-amd64   1/1     Running   0          2m

# Check flannel interface exists
ip link show flannel.1
# flannel.1@NONE: <BROADCAST,MULTICAST,UP,LOWER_UP>

# Check routes
ip route
# 10.244.1.0/24 via 10.244.1.0 dev flannel.1 onlink
```

**Solution**: If routes missing, check Flannel logs:
```bash
kubectl logs -n kube-flannel kube-flannel-ds-amd64
```

---

### Issue: Service DNS not working

**Symptom**:
```bash
kubectl exec pod-1 -- nslookup kubernetes.default.svc
# Server:    10.96.0.10
# ** server can't find kubernetes.default.svc: NXDOMAIN
```

**Diagnosis**:
```bash
# Check CoreDNS running
kubectl get pods -n kube-system -l k8s-app=kube-dns
# NAME          READY   STATUS    RESTARTS   AGE
# coredns-...   1/1     Running   0          5m

# Check CoreDNS can reach API server
kubectl logs -n kube-system coredns-abc123
```

**Solution**: Ensure CNI is installed and service network is in NO_PROXY.

---

## Firewall Configuration

### Required Ports

**Control Plane**:
- **6443**: Kubernetes API server (TCP)
- **2379-2380**: etcd client/peer (TCP)
- **10250**: Kubelet API (TCP)
- **10259**: kube-scheduler (TCP)
- **10257**: kube-controller-manager (TCP)

**Worker Nodes**:
- **10250**: Kubelet API (TCP)
- **30000-32767**: NodePort services (TCP/UDP)

**CNI-Specific**:
- **Flannel**: UDP 8472 (VXLAN)
- **Calico VXLAN**: UDP 4789
- **Calico BGP**: TCP 179
- **Cilium**: UDP 8472 (VXLAN)

**Stylus Overlay** (if enabled):
- **UDP 4789**: Stylus VXLAN tunnels

**Example firewalld Rules**:
```bash
# Control plane
firewall-cmd --permanent --add-port=6443/tcp
firewall-cmd --permanent --add-port=2379-2380/tcp
firewall-cmd --permanent --add-port=10250/tcp

# Worker
firewall-cmd --permanent --add-port=10250/tcp
firewall-cmd --permanent --add-port=30000-32767/tcp
firewall-cmd --permanent --add-port=30000-32767/udp

# Flannel CNI
firewall-cmd --permanent --add-port=8472/udp

# Stylus overlay (if used)
firewall-cmd --permanent --add-port=4789/udp

firewall-cmd --reload
```

---

## Integration Points

### With Stylus Overlay

- Stylus overlay provides host-to-host connectivity across sites
- CNI overlay provides pod-to-pod connectivity within cluster
- **No conflict** - separate ports and interfaces
- Both can run simultaneously

### With Proxy

- Proxy settings apply to CNI pods (image pulls)
- pod/service subnets must be in NO_PROXY
- CNI pod-to-pod traffic bypasses proxy

### With Deployment Modes

- Networking same for appliance and agent modes
- containerd socket differs (standard vs spectro-containerd)
- CNI configuration identical

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-kubeadm/pkg/provider/provider.go` - Network configuration parsing
- `/Users/rishi/work/src/provider-kubeadm/utils/proxy.go` - NO_PROXY calculation from subnets

## Related Skills

- See `provider-kubeadm:02-cni-installation` for **CRITICAL** CNI setup details
- See `provider-kubeadm:03-configuration-patterns` for podSubnet/serviceSubnet config
- See `provider-kubeadm:04-proxy-configuration` for proxy NO_PROXY with networks
- See `provider-kubeadm:08-troubleshooting` for network troubleshooting

**Related Provider Skills**:
- See `provider-k3s:05-networking` for k3s networking (includes Flannel)
- See `provider-rke2:05-networking` for rke2 networking (includes Canal)

**Stylus Overlay**:
- See `stylus:ai-knowledge-base/features-kb/feature-overlay.md` for Stylus VxLAN overlay details

## Documentation References

**CNI Specifications**:
- CNI Spec: https://github.com/containernetworking/cni/blob/main/SPEC.md
- Flannel: https://github.com/flannel-io/flannel/blob/master/Documentation/
- Calico: https://docs.tigera.io/calico/latest/networking/
- Cilium: https://docs.cilium.io/en/stable/network/

**Kubernetes Networking**:
- https://kubernetes.io/docs/concepts/cluster-administration/networking/
- https://kubernetes.io/docs/concepts/services-networking/network-policies/
