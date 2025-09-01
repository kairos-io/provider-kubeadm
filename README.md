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
# After deployment, get the worker join token with:
#   kubeadm token create --print-join-command
install:
  device: "auto"
  auto: true
  reboot: true

cluster:
  cluster_token: "your-cluster-token-here"
  control_plane_host: 10.10.131.183  # VIP IP for the cluster
  role: init
  config: |
    clusterConfiguration:
      kubernetesVersion: v1.32.4
      networking:
        podSubnet: 192.168.0.0/16
        serviceSubnet: 192.169.0.0/16

stages:
  initramfs:
  - users:
      kairos:
        groups:
          - sudo
        passwd: kairos
  - commands:
    - ln -s /etc/kubernetes/admin.conf /run/kubeconfig
```

### Worker Nodes

Use [`kairos-worker-minimal.yaml`](./kairos-worker-minimal.yaml) for worker nodes:

```yaml
#cloud-config
# Minimal Kairos configuration for worker node
install:
  device: "auto"
  auto: true
  reboot: true

cluster:
  cluster_token: "your-cluster-token-here"  # Get from: kubeadm token create
  control_plane_host: 10.10.131.183  # VIP IP for the cluster
  role: worker

stages:
  initramfs:
  - users:
      kairos:
        groups:
          - sudo
        passwd: kairos
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

### 2. Get Join Token

Once the master node is running, SSH to it and get the join token:

```bash
# SSH to master node
ssh kairos@<master-ip>

# Generate join token and command
sudo kubeadm token create --print-join-command
```

This will output something like:
```bash
kubeadm join 10.10.131.183:6443 --token abc123.defghijk456789 --discovery-token-ca-cert-hash sha256:1234567890abcdef...
```

Extract the token part (e.g., `abc123.defghijk456789`) for your worker configuration.

### 3. Deploy Worker Nodes

1. Update the worker configuration:
   - Set the `cluster_token` from step 2
   - Set the same `control_plane_host` as the master
   - Deploy as many worker nodes as needed

2. Deploy worker nodes using your preferred method

3. Workers will automatically join the cluster

### 4. Verify Cluster

SSH to the master node and check cluster status:

```bash
# Check nodes
kubectl get nodes

# Check system pods
kubectl get pods -n kube-system

# Get cluster info
kubectl cluster-info
```

## Configuration Details

### Essential Settings

- **`cluster_token`**: Bootstrap token for node authentication (expires after 24 hours)
- **`control_plane_host`**: IP address where the Kubernetes API server will be accessible
- **`role`**: Either `init` (first master), `controlplane` (additional masters), or `worker`
- **`kubernetesVersion`**: Must match the kubeadm/kubelet binaries in your Kairos image

### Network Configuration

- **`podSubnet`**: IP range for pod networking (default: 192.168.0.0/16)
- **`serviceSubnet`**: IP range for services (default: 192.169.0.0/16)

Adjust these subnets if they conflict with your existing network infrastructure.

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

1. **Node not joining**: Check network connectivity and firewall rules
2. **Token expired**: Generate a new token on the master node
3. **IP conflicts**: Adjust podSubnet/serviceSubnet in master config
4. **Auto-detection fails**: Uncomment and set explicit paths in config

### Required Ports

Ensure these ports are open between nodes:
- 6443: Kubernetes API server
- 2379-2380: etcd server client API
- 10250: Kubelet API
- 10251: kube-scheduler
- 10252: kube-controller-manager

## Building Custom Image

The default Kairos images come with provider-kairos (k3s/k0s). To use provider-kubeadm, you need to build a custom image that includes:
- containerd runtime
- kubeadm, kubelet, and kubectl binaries (specific version)
- provider-kubeadm plugin

**Important**: The `kubernetesVersion` in your configuration must match the kubeadm/kubelet version in your image.

### Prerequisites

- Docker
- Internet connection for downloading binaries

### Step 1: Create Dockerfile

Create a `Dockerfile` that extends the base Kairos image with provider-kubeadm:

```dockerfile
# Build stage for downloading provider-kubeadm
FROM alpine:latest AS provider-kubeadm-downloader

ARG PROVIDER_KUBEADM_VERSION=v4.7.0-rc.4
ARG TARGETARCH=amd64

# Install curl and checksum tools
RUN apk add --no-cache curl sha256sum

# Download provider-kubeadm binary and checksum
WORKDIR /tmp
RUN curl -fsSL -L "https://github.com/kairos-io/provider-kubeadm/releases/download/${PROVIDER_KUBEADM_VERSION}/agent-provider-kubeadm-${PROVIDER_KUBEADM_VERSION}-linux-${TARGETARCH}.tar.gz" \
    -o provider-kubeadm.tar.gz
RUN curl -fsSL -L "https://github.com/kairos-io/provider-kubeadm/releases/download/${PROVIDER_KUBEADM_VERSION}/agent-provider-kubeadm-${PROVIDER_KUBEADM_VERSION}-checksums.txt" \
    -o checksums.txt

# Verify checksum - extract expected hash and compare with actual
RUN EXPECTED_HASH=$(grep "agent-provider-kubeadm-${PROVIDER_KUBEADM_VERSION}-linux-${TARGETARCH}.tar.gz" checksums.txt | cut -d' ' -f1) && \
    ACTUAL_HASH=$(sha256sum provider-kubeadm.tar.gz | cut -d' ' -f1) && \
    echo "Expected: $EXPECTED_HASH" && \
    echo "Actual:   $ACTUAL_HASH" && \
    [ "$EXPECTED_HASH" = "$ACTUAL_HASH" ] || (echo "Checksum mismatch!" && exit 1)

# Extract binary
RUN tar -xzf provider-kubeadm.tar.gz
RUN chmod +x agent-provider-kubeadm

# Declare global ARGs
ARG KUBERNETES_VERSION=v1.32.0
ARG KUBERNETES_DISTRO=kubeadm
ARG MODEL=generic
ARG TRUSTED_BOOT=false
ARG VERSION=v3.1.1
ARG KAIROS_INIT=v0.5.18

# Get kairos-init stage
FROM quay.io/kairos/kairos-init:${KAIROS_INIT} AS kairos-init

# Main build stage
FROM quay.io/kairos/ubuntu:24.04-core-amd64-generic-v3.1.1

# Re-declare ARGs for this stage
ARG KUBERNETES_VERSION=v1.32.0
ARG KUBERNETES_DISTRO=kubeadm
ARG MODEL=generic
ARG TRUSTED_BOOT=false
ARG VERSION=v3.1.1

# Copy provider-kubeadm binary to the right location
COPY --from=provider-kubeadm-downloader /tmp/agent-provider-kubeadm /system/providers/provider-kubeadm

# Initialize Kairos with kubeadm (with retry for repository issues)
RUN --mount=type=bind,from=kairos-init,src=/kairos-init,dst=/kairos-init \
    for i in 1 2 3; do \
        echo "Attempt $i: Running kairos-init..."; \
        /kairos-init -l debug -m "${MODEL}" -t "${TRUSTED_BOOT}" -k "${KUBERNETES_DISTRO}" --k8sversion "${KUBERNETES_VERSION}" --version "${VERSION}" && \
        /kairos-init validate -t "${TRUSTED_BOOT}" && break; \
        echo "Attempt $i failed, retrying..."; \
        sleep 5; \
    done
```

### Step 2: Build the Docker Image

Build your custom Kairos image:

```bash
# Set your desired Kubernetes version
export KUBERNETES_VERSION=v1.32.0
export IMAGE_NAME=mykairos-kubeadm

# Build the image
docker build \
  --build-arg KUBERNETES_VERSION=${KUBERNETES_VERSION} \
  --build-arg TARGETARCH=amd64 \
  -t ${IMAGE_NAME}:latest \
  .
```

### Step 3: Create Bootable ISO

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

### Step 4: Verify the Build

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
  --build-arg KUBERNETES_VERSION=v1.32.0 \
  --build-arg PROVIDER_KUBEADM_VERSION=v4.7.0-rc.4 \
  --build-arg TARGETARCH=arm64 \
  --platform linux/arm64 \
  -t ${IMAGE_NAME}-arm64:latest \
  .
```

#### Different Kubernetes Versions

```bash
# For Kubernetes v1.31.x
docker build \
  --build-arg KUBERNETES_VERSION=v1.31.3 \
  --build-arg PROVIDER_KUBEADM_VERSION=v4.7.0-rc.4 \
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
2. **Version mismatch**: Ensure `KUBERNETES_VERSION` matches your configuration
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
