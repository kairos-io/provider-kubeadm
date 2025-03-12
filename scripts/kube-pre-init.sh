#!/bin/bash

exec   > >(tee -ia /var/log/kube-pre-init.log)
exec  2> >(tee -ia /var/log/kube-pre-init.log >& 2)
exec 19>> /var/log/kube-pre-init.log

export BASH_XTRACEFD="19"
set -x

root_path=$1

export PATH="$PATH:$root_path/usr/bin"
export PATH="$PATH:$root_path/usr/local/bin"

sysctl --system
modprobe overlay
modprobe br_netfilter
systemctl daemon-reload

if [ -f "$root_path"/opt/spectrocloud/kubeadm/bin/kubelet ]; then
  cp "$root_path"/opt/spectrocloud/kubeadm/bin/kubelet "$root_path"/usr/local/bin/kubelet
  systemctl daemon-reload
  systemctl enable kubelet && systemctl restart kubelet
  rm -rf "$root_path"/opt/spectrocloud/kubeadm/bin/kubelet
fi

if [ ! -f "$root_path"/usr/local/bin/kubelet ]; then
  mkdir -p "$root_path"/usr/local/bin
  cp "$root_path"/opt/kubeadm/bin/kubelet "$root_path"/usr/local/bin/kubelet
  systemctl enable kubelet && systemctl start kubelet
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