#!/bin/sh

set -ex

KUBE_VERSION=$1

systemctl daemon-reload && systemctl restart containerd
kubeadm config images pull --kubernetes-version "$KUBE_VERSION" --v=5 2>&1 | tee /opt/kubeadm/logs/images-pull.log