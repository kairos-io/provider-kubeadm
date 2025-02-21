#!/bin/bash

set -x
trap 'echo -n $(date)' DEBUG

if [ -f /etc/spectro/environment ]; then
  . /etc/spectro/environment
fi

export PATH="$PATH:$STYLUS_ROOT/usr/bin"
export PATH="$PATH:$STYLUS_ROOT/usr/local/bin"

if [ -S /run/spectro/containerd/containerd.sock ]; then
    kubeadm reset -f --cri-socket unix:///run/spectro/containerd/containerd.sock --cleanup-tmp-dir
else
    kubeadm reset -f --cleanup-tmp-dir
fi

iptables -F && iptables -t nat -F && iptables -t mangle -F && iptables -X
rm -rf /etc/kubernetes/etcd
rm -rf /etc/kubernetes/manifests
rm -rf /etc/kubernetes/pki
rm -rf /etc/containerd/config.toml
systemctl stop kubelet
if systemctl cat spectro-containerd >/dev/null 2<&1; then
  systemctl stop spectro-containerd
fi

if systemctl cat containerd >/dev/null 2<&1; then
  systemctl stop containerd
fi

umount -l /var/lib/kubelet
rm -rf /var/lib/kubelet && rm -rf ${STYLUS_ROOT}/var/lib/kubelet
umount -l /var/lib/spectro/containerd
rm -rf /var/lib/spectro/containerd && rm -rf ${STYLUS_ROOT}/var/lib/spectro/containerd
umount -l /opt/bin
rm -rf /opt/bin && rm -rf ${STYLUS_ROOT}/opt/bin
umount -l /opt/cni/bin
rm -rf /opt/cni && rm -rf ${STYLUS_ROOT}/opt/cni
umount -l /etc/kubernetes
rm -rf /etc/kubernetes && rm -rf ${STYLUS_ROOT}/etc/kubernetes

rm -rf ${STYLUS_ROOT}/opt/kubeadm
rm -rf ${STYLUS_ROOT}/opt/containerd
rm -rf ${STYLUS_ROOT}/opt/*init
rm -rf ${STYLUS_ROOT}/opt/*join
rm -rf ${STYLUS_ROOT}/opt/kube-images
rm -rf ${STYLUS_ROOT}/opt/sentinel_kubeadmversion

rm -rf ${STYLUS_ROOT}/etc/systemd/system/spectro-kubelet.slice
rm -rf ${STYLUS_ROOT}/etc/systemd/system/spectro-containerd.slice
rm -rf ${STYLUS_ROOT}/etc/systemd/system/kubelet.service
rm -rf ${STYLUS_ROOT}/etc/systemd/system/containerd.service 2> /dev/null
rm -rf ${STYLUS_ROOT}/etc/systemd/system/spectro-containerd.service 2> /dev/null

rm -rf /var/log/kube*.log
rm -rf /var/log/apiserver
rm -rf /var/log/pods

