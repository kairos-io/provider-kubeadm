VERSION 0.6
FROM alpine

ARG KUBEADM_VERSION=latest
ARG BASE_IMAGE=quay.io/kairos/core-opensuse:latest
ARG IMAGE_REPOSITORY=quay.io/kairos

ARG CRICTL_VERSION=1.25.0
ARG RELEASE_VERSION=0.4.0

ARG LUET_VERSION=0.32.4
ARG GOLINT_VERSION=v1.46.2
ARG GOLANG_VERSION=1.18

ARG KUBEADM_VERSION=latest
ARG BASE_IMAGE_NAME=$(echo $BASE_IMAGE | grep -o [^/]*: | rev | cut -c2- | rev)
ARG BASE_IMAGE_TAG=$(echo $BASE_IMAGE | grep -o :.* | cut -c2-)
ARG KUBEADM_VERSION_TAG=$(echo $KUBEADM_VERSION | sed s/+/-/)

build-cosign:
    FROM gcr.io/projectsigstore/cosign:v1.13.1
    SAVE ARTIFACT /ko-app/cosign cosign

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

VERSION:
    COMMAND
    FROM alpine
    RUN apk add git

    COPY . ./

    RUN echo $(git describe --exact-match --tags || echo "v0.0.0-$(git log --oneline -n 1 | cut -d" " -f1)") > VERSION

    SAVE ARTIFACT VERSION VERSION

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
    DO +VERSION
    ARG VERSION=$(cat VERSION)

    FROM $BASE_IMAGE

    WORKDIR /usr/bin
    RUN curl -L "https://github.com/kubernetes-sigs/cri-tools/releases/download/v${CRICTL_VERSION}/crictl-v${CRICTL_VERSION}-linux-amd64.tar.gz" | sudo tar -C /usr/bin/ -xz
    RUN curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${KUBEADM_VERSION}/bin/linux/amd64/kubeadm
    RUN curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${KUBEADM_VERSION}/bin/linux/amd64/kubelet
    RUN curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${KUBEADM_VERSION}/bin/linux/amd64/kubectl
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

    SAVE IMAGE --push $IMAGE_REPOSITORY/${BASE_IMAGE_NAME}-kubeadm:${KUBEADM_VERSION_TAG}
    SAVE IMAGE --push $IMAGE_REPOSITORY/${BASE_IMAGE_NAME}-kubeadm:${KUBEADM_VERSION_TAG}_${VERSION}

cosign:
    ARG --required ACTIONS_ID_TOKEN_REQUEST_TOKEN
    ARG --required ACTIONS_ID_TOKEN_REQUEST_URL

    ARG --required REGISTRY
    ARG --required REGISTRY_USER
    ARG --required REGISTRY_PASSWORD

    DO +VERSION
    ARG VERSION=$(cat VERSION)

    FROM docker

    ENV ACTIONS_ID_TOKEN_REQUEST_TOKEN=${ACTIONS_ID_TOKEN_REQUEST_TOKEN}
    ENV ACTIONS_ID_TOKEN_REQUEST_URL=${ACTIONS_ID_TOKEN_REQUEST_URL}

    ENV REGISTRY=${REGISTRY}
    ENV REGISTRY_USER=${REGISTRY_USER}
    ENV REGISTRY_PASSWORD=${REGISTRY_PASSWORD}

    ENV COSIGN_EXPERIMENTAL=1
    COPY +build-cosign/cosign /usr/local/bin/

    RUN echo $REGISTRY_PASSWORD | docker login -u $REGISTRY_USER --password-stdin $REGISTRY

    SAVE IMAGE --push $IMAGE_REPOSITORY/${BASE_IMAGE_NAME}-kubeadm:${KUBEADM_VERSION_TAG}
    SAVE IMAGE --push $IMAGE_REPOSITORY/${BASE_IMAGE_NAME}-kubeadm:${KUBEADM_VERSION_TAG}_${VERSION}

docker-all-platforms:
     BUILD --platform=linux/amd64 +docker
     BUILD --platform=linux/arm64 +docker

cosign-all-platforms:
     BUILD --platform=linux/amd64 +cosign
     BUILD --platform=linux/arm64 +cosign