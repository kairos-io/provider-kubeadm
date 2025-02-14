#!/bin/bash

set -x

root_path=$1

export PATH="$PATH:$root_path/usr/bin"
export PATH="$PATH:$root_path/usr/local/bin"

sysctl --system
modprobe overlay
modprobe br_netfilter
systemctl daemon-reload

# for new 1.30.x clusters
if [ -f "$root_path"/opt/kubeadm/bin/kubelet ]; then
  cp "$root_path"/opt/kubeadm/bin/kubelet "$root_path"/usr/local/bin/kubelet
  systemctl enable kubelet && systemctl start kubelet
fi

# for existing 1.30.x clusters
if [ -f "$root_path"/usr/bin/kubelet ]; then
  cp "$root_path"/usr/bin/kubelet "$root_path"/usr/local/bin/kubelet
  systemctl daemon-reload && systemctl restart kubelet
fi

if systemctl cat spectro-containerd >/dev/null 2<&1; then
  systemctl enable spectro-containerd && systemctl restart spectro-containerd
fi

if systemctl cat containerd >/dev/null 2<&1; then
  systemctl enable containerd && systemctl restart containerd
fi

if [ ! -f "$root_path"/opt/sentinel_kubeadmversion ]; then
  kubeadm version -o short > "$root_path"/opt/sentinel_kubeadmversion
fi