VERSION 0.6
FROM alpine

ARG KUBEADM_VERSION
ARG C3OS_BASE_IMAGE=quay.io/c3os/core-opensuse:latest
ARG IMAGE=quay.io/c3os/provider-kubeadm:dev

ARG LUET_VERSION=0.32.4
ARG GOLANG_VERSION=1.18
ARG RELEASE_VERSION=0.4.0

go-deps:
    FROM golang:$GOLANG_VERSION
    WORKDIR /build
    COPY go.mod go.sum ./
    RUN go mod download
    RUN apt-get update && apt-get install -y upx
    SAVE ARTIFACT go.mod AS LOCAL go.mod
    SAVE ARTIFACT go.sum AS LOCAL go.sum

BUILD_GOLANG:
    COMMAND
    WORKDIR /build
    COPY . ./
    ARG BIN
    ARG SRC
    ENV CGO_ENABLED=0
    RUN go build -ldflags "-s -w" -o ${BIN} ./${SRC} && upx ${BIN}
    SAVE ARTIFACT ${BIN} ${BIN} AS LOCAL build/${BIN}

build-provider:
    FROM +go-deps
    DO +BUILD_GOLANG --BIN=agent-provider-kubeadm --SRC=main.go

lint:
    FROM golang:$GOLANG_VERSION
    RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.46.2
    WORKDIR /build
    COPY . .
    RUN golangci-lint run

docker:
    FROM ${C3OS_BASE_IMAGE}

    WORKDIR /usr/bin
    RUN curl -L "https://github.com/kubernetes-sigs/cri-tools/releases/download/v${KUBEADM_VERSION}/crictl-v${KUBEADM_VERSION}-linux-amd64.tar.gz" | sudo tar -C /usr/bin/ -xz
    RUN curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/v${KUBEADM_VERSION}/bin/linux/amd64/kubeadm
    RUN curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/v${KUBEADM_VERSION}/bin/linux/amd64/kubelet
    RUN curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/v${KUBEADM_VERSION}/bin/linux/amd64/kubectl
    RUN chmod +x kubeadm
    RUN chmod +x kubelet
    RUN chmod +x kubectl

    RUN curl -sSL "https://raw.githubusercontent.com/kubernetes/release/v${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubelet/lib/systemd/system/kubelet.service" | sudo tee /etc/systemd/system/kubelet.service
    RUN mkdir -p /etc/systemd/system/kubelet.service.d
    RUN curl -sSL "https://raw.githubusercontent.com/kubernetes/release/v${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubeadm/10-kubeadm.conf" | sudo tee /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

    COPY luet/repositories.yaml /etc/luet/luet.yaml

    RUN luet repo list \
        && luet install -y container-runtime/containerd \
        && luet cleanup \
        && rm /etc/luet/luet.yaml

    WORKDIR /

    RUN mkdir -p /usr/local/lib/systemd/system
    RUN curl -sSL "https://raw.githubusercontent.com/containerd/containerd/main/containerd.service" | sudo tee /usr/local/lib/systemd/system/containerd.service
    COPY containerd/config.toml /etc/containerd/config.toml

    RUN mkdir -p /usr/local/sbin
    RUN curl -SL -o runc.amd64 "https://github.com/opencontainers/runc/releases/download/v1.1.4/runc.amd64"
    RUN install -m 755 runc.amd64 /usr/local/sbin/runc

    RUN echo "overlay" >> /etc/modules-load.d/k8s.conf
    RUN echo "br_netfilter" >> /etc/modules-load.d/k8s.conf

    RUN echo net.bridge.bridge-nf-call-iptables=1 >> /etc/sysctl.d/k8s.conf
    RUN echo net.bridge.bridge-nf-call-ip6tables=1 >> /etc/sysctl.d/k8s.conf
    RUN echo net.ipv4.ip_forward=1 >> /etc/sysctl.d/k8s.conf

    COPY +build-provider/agent-provider-kubeadm /system/providers/agent-provider-kubeadm
    COPY cni/calico.yaml /opt/kubeadm/calico.yaml

    SAVE IMAGE --push $IMAGE

