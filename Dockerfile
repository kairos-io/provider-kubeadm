# Multi-stage Dockerfile to build Kairos image with kubeadm provider using kairos-init

# Build arguments
ARG BASE_IMAGE=quay.io/kairos/fedora:40-core-amd64-generic-v3.5.1
# IMPORTANT: This version must match the kubernetesVersion in your configuration files
# (e.g., kairos-master-minimal.yaml, kairos-worker-minimal.yaml)
ARG KUBEADM_VERSION=latest
ARG CRICTL_VERSION=1.25.0
ARG RELEASE_VERSION=0.4.0 # Update to 0.18.0?  https://github.com/kubernetes/release/releases/tag/v0.18.0
ARG FIPS_ENABLED=false
ARG KAIROS_INIT_VERSION=v0.6.0
ARG VERSION=latest

# Stage 1: Get kairos-init binary
FROM quay.io/kairos/kairos-init:${KAIROS_INIT_VERSION} AS kairos-init

# Stage 2: Build the provider binary
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION
ENV GO_LDFLAGS="-X github.com/kairos-io/kairos/provider-kubeadm/version.Version=${VERSION} -w -s"
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "${GO_LDFLAGS}" -o agent-provider-kubeadm main.go

# Stage 3: Download Kubernetes binaries
FROM alpine:latest AS k8s-binaries
ARG KUBEADM_VERSION
ARG CRICTL_VERSION
ARG FIPS_ENABLED=false

RUN apk add --no-cache curl jq

WORKDIR /binaries

# Resolve "latest" to actual version if needed
RUN if [ "$KUBEADM_VERSION" = "latest" ]; then \
        if [ "$FIPS_ENABLED" = "true" ]; then \
            # For FIPS, use the latest available version in spectro-fips (1.25.2 as of last check)
            RESOLVED_VERSION="1.25.2"; \
        else \
            # Get the latest stable version from Kubernetes API
            RESOLVED_VERSION=$(curl -s https://api.github.com/repos/kubernetes/kubernetes/releases/latest | jq -r '.tag_name'); \
        fi; \
        echo "Resolved KUBEADM_VERSION from 'latest' to: $RESOLVED_VERSION"; \
        echo "$RESOLVED_VERSION" > /tmp/k8s_version; \
    else \
        echo "$KUBEADM_VERSION" > /tmp/k8s_version; \
    fi

# Download crictl
RUN if [ "$FIPS_ENABLED" = "true" ]; then \
        curl -L "https://storage.googleapis.com/spectro-fips/cri-tools/v${CRICTL_VERSION}/cri-tools-${CRICTL_VERSION}-linux-amd64.tar.gz" | tar -xz; \
    else \
        curl -L "https://github.com/kubernetes-sigs/cri-tools/releases/download/v${CRICTL_VERSION}/crictl-v${CRICTL_VERSION}-linux-amd64.tar.gz" | tar -xz; \
    fi

# Download kubeadm, kubelet, kubectl
RUN K8S_VERSION=$(cat /tmp/k8s_version) && \
    if [ "$FIPS_ENABLED" = "true" ]; then \
        curl -L -o kubeadm "https://storage.googleapis.com/spectro-fips/${K8S_VERSION}/kubeadm" && \
        curl -L -o kubelet "https://storage.googleapis.com/spectro-fips/${K8S_VERSION}/kubelet" && \
        curl -L -o kubectl "https://storage.googleapis.com/spectro-fips/${K8S_VERSION}/kubectl"; \
    else \
        curl -L -o kubeadm "https://dl.k8s.io/${K8S_VERSION}/bin/linux/amd64/kubeadm" && \
        curl -L -o kubelet "https://dl.k8s.io/${K8S_VERSION}/bin/linux/amd64/kubelet" && \
        curl -L -o kubectl "https://dl.k8s.io/${K8S_VERSION}/bin/linux/amd64/kubectl"; \
    fi

RUN chmod +x kubeadm kubelet kubectl crictl

# Save the resolved version for later use
RUN cp /tmp/k8s_version /binaries/k8s_version

# Stage 4: Download containerd and CNI plugins
FROM alpine:latest AS containerd-binaries
ARG FIPS_ENABLED=false

RUN apk add --no-cache curl

WORKDIR /containerd

# Download containerd
RUN if [ "$FIPS_ENABLED" = "true" ]; then \
        curl -sSL "https://storage.googleapis.com/spectro-fips/containerd/v1.6.4/containerd-1.6.4-linux-amd64.tar.gz" | tar -xz; \
    else \
        curl -sSL "https://github.com/containerd/containerd/releases/download/v2.1.4/containerd-2.1.4-linux-amd64.tar.gz" | tar -xz; \
    fi

# Download runc
RUN if [ "$FIPS_ENABLED" = "true" ]; then \
        curl -SL -o runc "https://storage.googleapis.com/spectro-fips/runc-1.1.4/runc"; \
    else \
        curl -SL -o runc "https://github.com/opencontainers/runc/releases/download/v1.3.0/runc.amd64"; \
    fi

RUN chmod +x runc

# Download CNI plugins
RUN mkdir -p cni-plugins && \
    if [ "$FIPS_ENABLED" = "true" ]; then \
        curl -sSL "https://storage.googleapis.com/spectro-fips/cni-plugins/v1.1.1/cni-plugins-1.1.1-linux-amd64.tar.gz" | tar -C cni-plugins -xz; \
    else \
        curl -sSL "https://github.com/containernetworking/plugins/releases/download/v1.8.0/cni-plugins-linux-amd64-v1.8.0.tgz" | tar -C cni-plugins -xz; \
    fi

# Stage 5: Main image
FROM ${BASE_IMAGE}
ARG KUBEADM_VERSION
ARG RELEASE_VERSION
ARG VERSION
ARG FIPS_ENABLED=false

# Copy kairos-init (but don't run it yet)
COPY --from=kairos-init /kairos-init /kairos-init

# Copy Kubernetes binaries and version info
COPY --from=k8s-binaries /binaries/kubeadm /usr/bin/kubeadm
COPY --from=k8s-binaries /binaries/kubelet /usr/bin/kubelet
COPY --from=k8s-binaries /binaries/kubectl /usr/bin/kubectl
COPY --from=k8s-binaries /binaries/crictl /usr/bin/crictl
COPY --from=k8s-binaries /binaries/k8s_version /tmp/k8s_version

# Setup containerd
COPY --from=containerd-binaries /containerd/bin/* /opt/bin/
COPY --from=containerd-binaries /containerd/runc /opt/bin/runc
RUN mkdir -p /opt/cni/bin
COPY --from=containerd-binaries /containerd/cni-plugins/* /opt/cni/bin/

# Copy containerd from /opt/bin/ctr to /usr/bin/ctr for compatibility
RUN cp /opt/bin/ctr /usr/bin/ctr

# Download and setup containerd systemd service
RUN curl -sSL "https://raw.githubusercontent.com/containerd/containerd/main/containerd.service" | \
    sed "s?ExecStart=/usr/local/bin/containerd?ExecStart=/opt/bin/containerd?" > /etc/systemd/system/containerd.service

# Setup kubelet systemd service
RUN curl -sSL "https://raw.githubusercontent.com/kubernetes/release/v${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubelet/lib/systemd/system/kubelet.service" > /etc/systemd/system/kubelet.service && \
    mkdir -p /etc/systemd/system/kubelet.service.d && \
    curl -sSL "https://raw.githubusercontent.com/kubernetes/release/v${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubeadm/10-kubeadm.conf" > /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

# Copy containerd configuration
COPY containerd/config.toml /etc/containerd/config.toml

# Copy scripts
RUN mkdir -p /opt/kubeadm/scripts
COPY scripts/* /opt/kubeadm/scripts/

# Copy provider binary
COPY --from=builder /build/agent-provider-kubeadm /system/providers/agent-provider-kubeadm

# Load Kubernetes images (only if not FIPS)
RUN if [ "$FIPS_ENABLED" != "true" ]; then \
        K8S_VERSION=$(cat /tmp/k8s_version) && \
        bash /opt/kubeadm/scripts/kube-images-load.sh ${K8S_VERSION}; \
    fi

# Setup kernel modules
RUN echo "overlay" >> /etc/modules-load.d/k8s.conf && \
    echo "br_netfilter" >> /etc/modules-load.d/k8s.conf

# Setup networking parameters
RUN echo "net.bridge.bridge-nf-call-iptables=1" >> /etc/sysctl.d/k8s.conf && \
    echo "net.bridge.bridge-nf-call-ip6tables=1" >> /etc/sysctl.d/k8s.conf && \
    echo "net.ipv4.ip_forward=1" >> /etc/sysctl.d/k8s.conf

# Set OS identification environment variables
ARG BASE_IMAGE
ARG IMAGE_REPOSITORY=quay.io/kairos
RUN BASE_IMAGE_NAME=$(echo ${BASE_IMAGE} | grep -o '[^/]*:' | rev | cut -c2- | rev) && \
    BASE_IMAGE_TAG=$(echo ${BASE_IMAGE} | grep -o ':.*' | cut -c2-) && \
    KUBEADM_VERSION_TAG=$(echo ${KUBEADM_VERSION} | sed 's/+/-/') && \
    echo "OS_ID=${BASE_IMAGE_NAME}-kubeadm" >> /etc/os-release && \
    echo "OS_NAME=${BASE_IMAGE_NAME}-kubeadm:${BASE_IMAGE_TAG}" >> /etc/os-release && \
    echo "OS_REPO=${IMAGE_REPOSITORY}" >> /etc/os-release && \
    echo "OS_VERSION=${KUBEADM_VERSION_TAG}_${VERSION}" >> /etc/os-release && \
    echo "OS_LABEL=${BASE_IMAGE_TAG}_${KUBEADM_VERSION_TAG}_${VERSION}" >> /etc/os-release



# Configure dracut to omit iSCSI modules that cause boot failures
# This prevents the "iscsiroot requested but kernel/initrd does not support iscsi" error
# Following the pattern from official Kairos examples for Ubuntu builds
RUN mkdir -p /etc/dracut.conf.d && \
    echo 'omit_dracutmodules+=" iscsi iscsiroot "' > /etc/dracut.conf.d/no-iscsi.conf

# Now run kairos-init at the very end after all setup is complete
# This ensures all binaries and configurations are available for initramfs creation
# Use the resolved K8s version or generate a proper semver if VERSION is "latest"
RUN KAIROS_VERSION="${VERSION}" && \
    if [ "${VERSION}" = "latest" ]; then \
        K8S_VERSION=$(cat /tmp/k8s_version) && \
        KAIROS_VERSION="v1.0.0-${K8S_VERSION}"; \
    fi && \
    echo "Running kairos-init with version: ${KAIROS_VERSION}" && \
    /kairos-init -l info -m generic --version "${KAIROS_VERSION}"

RUN /kairos-init validate;

# Clean up kairos-init binary
RUN rm /kairos-init
