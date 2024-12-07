#!/bin/bash

set -x

root_path=$1

export PATH="$PATH:$root_path/usr/bin"

sysctl --system
modprobe overlay
modprobe br_netfilter
systemctl daemon-reload

systemctl enable kubelet && systemctl start kubelet

if systemctl cat spectro-containerd >/dev/null 2<&1; then
  systemctl enable spectro-containerd && systemctl restart spectro-containerd
fi

if systemctl cat containerd >/dev/null 2<&1; then
  systemctl enable containerd && systemctl restart containerd
fi

if [ ! -f "$root_path"/opt/sentinel_kubeadmversion ]; then
  kubeadm version -o short > "$root_path"/opt/sentinel_kubeadmversion
fi