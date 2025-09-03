# Kairos Kubeadm Cluster Plugin
Kairos provider for Kubeadm

## Overview

The provider-kubeadm enables Kairos to bootstrap Kubernetes clusters using kubeadm instead of the default k3s/k0s providers. This gives you a standard upstream Kubernetes cluster with full control over the configuration.

## Prerequisites

- Kairos image with provider-kubeadm and containerd (see [Building Custom Image](#building-custom-image) section)
- Network connectivity between nodes
- Basic understanding of Kubernetes networking

## Testing
The project includes comprehensive unit and integration tests with full coverage verification. All tests are designed to run without external dependencies for reliable CI/CD execution.

## Configuration Files

### Master/Control Plane Node

Use [`kairos-master-minimal.yaml`](./kairos-master-minimal.yaml) for the first node (control plane):

```yaml
#cloud-config
# Minimal Kairos configuration for master/init node
# IMPORTANT: Update control_plane_host to your actual node IP address
# After deployment, workers should use the same cluster_token string
install:
  device: "auto"
  auto: true
  reboot: true

cluster:
  cluster_token: "your-cluster-token-here"  # Any string - workers must use the same string
  control_plane_host: 192.168.122.71  # ← CHANGE TO YOUR ACTUAL NODE IP
  role: init
  config: |
    clusterConfiguration:
      # IMPORTANT: This version must match KUBEADM_VERSION in the Dockerfile used to build your Kairos image
      kubernetesVersion: v1.34.0
      controlPlaneEndpoint: "192.168.122.71:6443"  # ← SAME IP AS ABOVE
      networking:
        podSubnet: 10.244.0.0/16      # Flannel default
        serviceSubnet: 10.96.0.0/12   # Kubernetes default

stages:
  initramfs:
  - users:
      kairos:
        groups:
          - sudo
        passwd: kairos
  - commands:
    - ln -s /etc/kubernetes/admin.conf /run/kubeconfig
    # Disable firewalld to prevent Kubernetes networking issues
    - systemctl disable --now firewalld
```

### Worker Nodes

Use [`kairos-worker-minimal.yaml`](./kairos-worker-minimal.yaml) for worker nodes:

```yaml
#cloud-config
# Minimal Kairos configuration for worker node
# IMPORTANT: Use the same control_plane_host as your master node
install:
  device: "auto"
  auto: true
  reboot: true

cluster:
  cluster_token: "your-cluster-token-here"  # Use the same string as your master node
  control_plane_host: 192.168.122.71  # ← SAME IP AS YOUR MASTER NODE
  role: worker

stages:
  initramfs:
  - users:
      kairos:
        groups:
          - sudo
        passwd: kairos
  - commands:
    # Disable firewalld to prevent Kubernetes networking issues
    - systemctl disable --now firewalld
```

## Deployment Steps

### 1. Deploy Master Node

1. Customize the master configuration:
   - Set your desired `control_plane_host` IP address
   - Update `kubernetesVersion` if needed
   - Adjust network subnets if required
   - Generate a secure cluster token (or use the auto-generated one)

2. Deploy the master node using your preferred method (ISO, PXE, etc.)

3. Wait for the node to boot and initialize the cluster

### 2. Deploy Worker Nodes

1. Update the worker configuration:
   - Set the same `cluster_token` string as used in your master configuration
   - Set the same `control_plane_host` as the master
   - Deploy as many worker nodes as needed

> **Note**: The `cluster_token` can be any string of your choice. The provider automatically converts it to a valid kubeadm token format. You just need to use the same string on both master and worker nodes.

2. Deploy worker nodes using your preferred method

3. Workers will automatically join the cluster

### 3. Verify Cluster

Once all nodes are deployed, you can verify the cluster by SSH'ing to the master node:

```bash
# SSH to master node
ssh kairos@<master-ip>

# Check cluster status
kubectl get nodes
```

### 4. Install CNI Network Plugin

**⚠️ CRITICAL**: After the cluster initializes, you **must** install a CNI (Container Network Interface) plugin for pod networking to work.

#### Install Flannel (Recommended)

SSH to the master node and install Flannel:

```bash
# Simple one-command installation - works perfectly with kubeadm defaults!
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

**Why Flannel?**
- ✅ **Zero configuration needed** with kubeadm defaults
- ✅ **Simple and reliable** for most use cases
- ✅ **Well-documented** and widely used

#### Alternative CNI Plugins

If you need different features, consider these alternatives:
- **Calico**: Advanced network policies, BGP routing (requires configuration)
- **Weave Net**: Automatic encryption, service mesh features
- **Cilium**: eBPF-based networking, advanced observability

**Important Notes**:
- Wait 2-3 minutes for the CNI pods to start
- The cluster won't be functional until CNI is installed
- Flannel works perfectly with kubeadm's default pod subnet (`10.244.0.0/16`)

### 5. Verify Cluster

SSH to the master node and check cluster status:

```bash
# Check nodes (should show Ready status after CNI installation)
kubectl get nodes

# Check system pods (should include CNI pods)
kubectl get pods -n kube-system

# Verify CNI is working
kubectl get pods -n kube-flannel  # For Flannel
kubectl get pods -n calico-system # For Calico

# Get cluster info
kubectl cluster-info
```

Example of healthy cluster output:
```bash
$ kubectl get nodes
NAME     STATUS   ROLES           AGE   VERSION
fedora   Ready    control-plane   10m   v1.34.0

$ kubectl get pods -n kube-system
NAME                             READY   STATUS    RESTARTS   AGE
coredns-6f6b679f8f-abc12         1/1     Running   0          10m
coredns-6f6b679f8f-def34         1/1     Running   0          10m
etcd-fedora                      1/1     Running   0          10m
kube-apiserver-fedora            1/1     Running   0          10m
kube-controller-manager-fedora   1/1     Running   0          10m
kube-flannel-ds-xyz89            1/1     Running   0          5m
kube-proxy-gh567                 1/1     Running   0          10m
kube-scheduler-fedora            1/1     Running   0          10m
```

## Configuration Details

### Essential Settings

- **`cluster_token`**: Bootstrap token for node authentication (expires after 24 hours)
- **`control_plane_host`**: IP address where the Kubernetes API server will be accessible
- **`role`**: Either `init` (first master), `controlplane` (additional masters), or `worker`
- **`kubernetesVersion`**: Must match the `KUBEADM_VERSION` build argument in your Dockerfile

#### ⚠️ Critical: Version Consistency Requirement

The `kubernetesVersion` specified in your configuration files **must exactly match** the `KUBEADM_VERSION` build argument used when building your Kairos image. Mismatched versions will cause cluster initialization to fail.

**Example consistency check**:
- Dockerfile: `ARG KUBEADM_VERSION=v1.34.0`
- Config file: `kubernetesVersion: v1.34.0` ✅
- Config file: `kubernetesVersion: v1.33.0` ❌ (mismatch!)

### Network Configuration

#### Critical IP Address Setup

**⚠️ IMPORTANT**: The `control_plane_host` must match the **actual IP address** of your master node, not a VIP or placeholder IP.

To find your node's IP address:
```bash
# Check the node's actual IP address
ip addr show | grep "inet.*scope global"
```

Then update your configuration:
```yaml
cluster:
  control_plane_host: 192.168.122.71  # ← Use your ACTUAL node IP here
  config: |
    clusterConfiguration:
      controlPlaneEndpoint: "192.168.122.71:6443"  # ← Same IP here
```

#### Subnet Configuration

**Recommended: Use standard defaults**:
- **podSubnet**: `10.244.0.0/16` (Flannel default)
- **serviceSubnet**: `10.96.0.0/12` (Kubernetes default)

**Why these settings work well**:
- ✅ **Perfect Flannel compatibility** - matches Flannel's default configuration
- ✅ **No CNI configuration required** - Flannel works out of the box  
- ✅ **Standard Kubernetes setup** - matches most documentation
- ✅ **Unlikely to conflict** with host networks

**Important**: The `podSubnet` **must be specified** for CNI plugins to work properly. You cannot omit the networking section entirely.

**For custom networks** (only if defaults conflict):
```yaml
networking:
  podSubnet: 192.168.0.0/16      # Custom pod subnet
  serviceSubnet: 192.169.0.0/16  # Custom service subnet
```

**Subnet compatibility check**:
- Host network: `192.168.122.0/24` (example)
- Recommended: `10.244.0.0/16` (pods), `10.96.0.0/12` (services) ✅ (no overlap)

### Auto-Detection

This minimal configuration relies on kubeadm's auto-detection for:
- Container runtime socket location
- Node IP addresses
- Default security settings

If auto-detection fails, you can uncomment and customize the advanced settings in the configuration files.

## Token Management

### Important Notes

- **Tokens are only needed during initial join** - once a node joins, it stays in the cluster permanently
- **Tokens expire after 24 hours** by default for security
- **Expired tokens don't affect existing nodes** - only prevent new nodes from joining

### Token Commands

```bash
# List existing tokens
kubeadm token list

# Create new token (24h expiry)
kubeadm token create

# Create permanent token (use carefully!)
kubeadm token create --ttl 0

# Get full join command
kubeadm token create --print-join-command
```

## Troubleshooting

### Common Issues

#### 1. Kubelet Won't Start - "NetworkPluginNotReady"

**Symptoms**: 
```
"Container runtime network not ready" networkReady="NetworkPluginNotReady message:Network plugin returns error: cni plugin not initialized"
```

**Cause**: No CNI plugin installed after cluster initialization.

**Solution**: Install a CNI plugin (see [CNI Installation](#4-install-cni-network-plugin) section above).

#### 2. Control Plane Connection Refused

**Symptoms**:
```
kubectl: The connection to the server localhost:8080 was refused
# OR
dial tcp 192.168.122.41:6443: connect: connection refused
```

**Causes & Solutions**:

**A. Wrong IP in configuration**:
- Check actual node IP: `ip addr show`
- Update `control_plane_host` to match the actual IP
- Add `controlPlaneEndpoint` with same IP

**B. Missing kubeconfig**:
```bash
export KUBECONFIG=/etc/kubernetes/admin.conf
# OR copy to default location
mkdir -p ~/.kube
sudo cp /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config
```

#### 3. CNI Subnet Mismatch (Flannel)

**Symptoms**:
```
failed to acquire lease: subnet "10.244.0.0/16" specified in the flannel net config doesn't contain "192.168.0.0/24" PodCIDR
```

**Solution**: Edit Flannel configuration to match your `podSubnet`:
```bash
# Download and modify Flannel manifest
curl -s https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml -o kube-flannel.yml
sed -i 's|10.244.0.0/16|192.168.0.0/16|g' kube-flannel.yml
kubectl apply -f kube-flannel.yml
```

#### 4. Node IP Detection Issues

**Symptoms**: Wrong IP detected by kubeadm during init.

**Solutions**:
- Use `--apiserver-advertise-address` flag
- Set explicit `controlPlaneEndpoint` in configuration
- Check routing table: `ip route show default`

#### 5. General Debugging

1. **Node not joining**: Check network connectivity and firewall rules
2. **Token expired**: Generate a new token on the master node  
3. **IP conflicts**: Adjust podSubnet/serviceSubnet in master config
4. **Auto-detection fails**: Use explicit configuration values

### Log Files

Check these logs for detailed error information:
```bash
# Kubelet logs
journalctl -u kubelet -f

# Container runtime logs  
journalctl -u containerd -f

# Kairos initialization logs
cat /var/log/kube-init.log
cat /var/log/kube-post-init.log

# Pod logs
kubectl logs -n kube-system <pod-name>
```

### Firewall Configuration

**Automatic firewall disabling**: The provided configurations automatically disable `firewalld` on both master and worker nodes to prevent Kubernetes networking issues.

**Why disable firewalld?**
- ✅ **Prevents CNI conflicts** - Avoids interference with pod networking
- ✅ **Simplifies setup** - No need to configure complex firewall rules
- ✅ **Reduces troubleshooting** - Eliminates firewall-related networking issues

**Alternative approach** (if you need firewall enabled):
```bash
# Instead of disabling, configure firewalld for Kubernetes
firewall-cmd --permanent --add-port=6443/tcp     # API server
firewall-cmd --permanent --add-port=2379-2380/tcp # etcd
firewall-cmd --permanent --add-port=10250/tcp    # kubelet
firewall-cmd --permanent --add-port=10251/tcp    # kube-scheduler  
firewall-cmd --permanent --add-port=10252/tcp    # kube-controller-manager
firewall-cmd --permanent --add-port=8472/udp     # Flannel VXLAN
firewall-cmd --reload
```

### Required Ports (Reference)

If you choose to configure firewall instead of disabling it:
- 6443: Kubernetes API server
- 2379-2380: etcd server client API  
- 10250: Kubelet API
- 10251: kube-scheduler
- 10252: kube-controller-manager
- 8472/udp: Flannel VXLAN (CNI specific)

## Building Custom Image

The default Kairos images come with provider-kairos (k3s/k0s). To use provider-kubeadm, you need to build a custom image that includes:
- containerd runtime
- kubeadm, kubelet, and kubectl binaries (specific version)
- provider-kubeadm plugin

**Important**: The `kubernetesVersion` in your configuration must match the `KUBEADM_VERSION` build argument used when building your Kairos image with the Dockerfile.

### Prerequisites

- Docker
- Internet connection for downloading binaries

### Step 1: Build a container image

Use the `Dockerfile` at the root of this repository to build an image:

```bash
# Set your desired Kubernetes version
export KUBERNETES_VERSION=v1.32.0
export IMAGE_NAME=mykairos-kubeadm

# IMPORTANT: Remember to update kubernetesVersion in your config files to match!
# Build the image
docker build \
  --build-arg KUBEADM_VERSION=${KUBERNETES_VERSION} \
  --build-arg TARGETARCH=amd64 \
  -t ${IMAGE_NAME}:latest \
  .
```

### Step 2: Create Bootable ISO

Use Auroraboot to convert your Docker image into a bootable ISO:

```bash
# Create build directory
mkdir -p build

# Build ISO from your custom image
docker run --rm -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/build:/build \
  quay.io/kairos/auroraboot:latest \
    --debug build-iso \
    --output /build \
    docker://${IMAGE_NAME}:latest
```

### Step 3: Verify the Build

Check that your ISO contains the provider-kubeadm:

```bash
# List the generated files
ls -la build/

# The ISO file should be present
# Example: kairos-ubuntu-24.04-generic-amd64-generic-v3.1.1.iso
```

### Customization Options

#### Different Architectures

For ARM64 builds:

```bash
docker build \
  --build-arg KUBEADM_VERSION=v1.32.0 \
  --build-arg VERSION=v4.7.0-rc.4 \
  --build-arg TARGETARCH=arm64 \
  --platform linux/arm64 \
  -t ${IMAGE_NAME}-arm64:latest \
  .
```

> **Note:** The `VERSION` build argument sets the version string that gets embedded into the provider binary for reporting purposes. It does not affect which version of the provider code is built - that is determined by the Git commit/tag you're building from.

#### Different Kubernetes Versions

```bash
# For Kubernetes v1.31.x
docker build \
  --build-arg KUBEADM_VERSION=v1.31.3 \
  --build-arg VERSION=v4.7.0-rc.4 \
  -t mykairos-kubeadm-v1.31:latest \
  .
```

#### Different Base Images

```bash
# Using Fedora base instead of Ubuntu
FROM quay.io/kairos/fedora:39-core-amd64-generic-v3.1.1 AS base-kairos
```

### Troubleshooting

#### Build Issues

1. **Provider download fails**: Check the [latest releases](https://github.com/kairos-io/provider-kubeadm/releases) for correct version
2. **Checksum verification fails**: Version mismatch between binary and checksum file
3. **Architecture mismatch**: Ensure `TARGETARCH` matches your target platform

#### Runtime Issues

1. **Provider not found**: Verify the binary is in `/system/providers/provider-kubeadm`
2. **Version mismatch**: Ensure `KUBEADM_VERSION` in Dockerfile matches `kubernetesVersion` in your config files
3. **Permission issues**: Binary should be executable (`chmod +x`)

### Advanced Configuration

For production builds, consider:

- **Multi-stage optimization**: Reduce final image size
- **Security scanning**: Scan for vulnerabilities
- **Automated builds**: CI/CD pipeline for regular updates
- **Version pinning**: Lock all component versions for reproducibility

## Advanced Configuration

For production deployments, consider:
- High availability control plane setup
- External etcd cluster
- Load balancer for API server
- Network policies and security hardening
- Monitoring and logging setup

## Contributing

Feel free to submit issues and pull requests to improve this configuration and documentation.

## License

This configuration is provided under the same license as the Kairos project.
