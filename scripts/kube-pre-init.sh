#!/bin/bash

set -x

export PATH="$PATH:$root_path/usr/bin"

root_path=$1

sysctl --system
modprobe overlay
modprobe br_netfilter
systemctl daemon-reload

systemctl enable kubelet && systemctl start kubelet

if systemctl cat spectro-containerd >/dev/null 2<&1; then
  systemctl enable spectro-containerd && systemctl start spectro-containerd
fi

if systemctl cat containerd >/dev/null 2<&1; then
  systemctl enable containerd && systemctl start containerd
fi

if [ ! -f "$root_path"/opt/sentinel_kubeadmversion ]; then
  kubeadm version -o short > "$root_path"/opt/sentinel_kubeadmversion
fi