#!/bin/bash

set -x
trap 'echo -n $(date)' DEBUG

if [ -f /etc/spectro/environment ]; then
  . /etc/spectro/environment
fi

export PATH="$PATH:$STYLUS_ROOT/usr/bin"

if [ -f /run/spectro/containerd/containerd.sock ]; then
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
umount -l /opt/spectro/bin
rm -rf /opt/spectro/bin && rm -rf ${STYLUS_ROOT}/opt/spectro/bin
umount -l /opt/spectro/cni/bin
rm -rf /opt/spectro/cni && rm -rf ${STYLUS_ROOT}/opt/spectro/cni
umount -l /etc/kubernetes
rm -rf /etc/kubernetes && rm -rf ${STYLUS_ROOT}/etc/kubernetes

rm -rf ${STYLUS_ROOT}/opt/kubeadm
rm -rf ${STYLUS_ROOT}/opt/*init
rm -rf ${STYLUS_ROOT}/opt/kube-images
rm -rf ${STYLUS_ROOT}/opt/sentinel_kubeadmversion

rm -rf /etc/systemd/system/etc-default-kubelet.mount
rm -rf /etc/systemd/system/etc-cni-net.d.mount
rm -rf /etc/systemd/system/opt-spectro-cni-bin.mount
rm -rf /etc/systemd/system/opt-spectro-bin.mount
rm -rf /etc/systemd/system/var-lib-spectro-containerd.mount
rm -rf /etc/systemd/system/var-lib-kubelet.mount

rm -rf /etc/systemd/system/spectro-kubelet.slice
rm -rf /etc/systemd/system/spectro-containerd.slice
rm -rf /etc/systemd/system/kubelet.service
rm -rf /etc/systemd/system/containerd.service 2> /dev/null
rm -rf /etc/systemd/system/spectro-containerd.service 2> /dev/null

rm -rf /var/log/kube*.log
rm -rf /var/log/apiserver
rm -rf /var/log/pods

