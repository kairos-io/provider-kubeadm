#!/bin/bash

exec   > >(tee -ia /var/log/kube-init.log)
exec  2> >(tee -ia /var/log/kube-init.log >& 2)
exec 19>> /var/log/kube-init.log

export BASH_XTRACEFD="19"
set -ex

root_path=$1
PROXY_CONFIGURED=$2
proxy_http=$3
proxy_https=$4
proxy_no=$5
KUBE_VIP_LOC="/etc/kubernetes/manifests/kube-vip.yaml"

export PATH="$PATH:$root_path/usr/bin"
export PATH="$PATH:$root_path/usr/local/bin"

restart_containerd() {
  if systemctl cat spectro-containerd >/dev/null 2<&1; then
    systemctl restart spectro-containerd
  fi

  if systemctl cat containerd >/dev/null 2<&1; then
    systemctl restart containerd
  fi
}

do_kubeadm_reset() {
  if [ -S /run/spectro/containerd/containerd.sock ]; then
    kubeadm reset -f --cri-socket unix:///run/spectro/containerd/containerd.sock --cleanup-tmp-dir
  else
    kubeadm reset -f --cleanup-tmp-dir
  fi

  iptables -F && iptables -t nat -F && iptables -t mangle -F && iptables -X && rm -rf /etc/kubernetes/etcd /etc/kubernetes/manifests /etc/kubernetes/pki
  rm -rf "$root_path"/etc/cni/net.d
  if [ -f /run/systemd/system/etc-cni-net.d.mount ]; then
    mkdir -p "$root_path"/etc/cni/net.d
    systemctl restart etc-cni-net.d.mount
  fi
  systemctl daemon-reload
  restart_containerd
}

backup_kube_vip_manifest_if_present() {
  if [ -f "$KUBE_VIP_LOC" ]; then
    cp $KUBE_VIP_LOC "$root_path"/opt/kubeadm/kube-vip.yaml
  fi
}

restore_kube_vip_manifest_after_reset() {
  if [ -f "$root_path/opt/kubeadm/kube-vip.yaml" ]; then
      mkdir -p "$root_path"/etc/kubernetes/manifests
      cp "$root_path"/opt/kubeadm/kube-vip.yaml $KUBE_VIP_LOC
  fi
}

if [ "$PROXY_CONFIGURED" = true ]; then
  until HTTP_PROXY=$proxy_http http_proxy=$proxy_http HTTPS_PROXY=$proxy_https https_proxy=$proxy_https NO_PROXY=$proxy_no no_proxy=$proxy_no kubeadm init --config "$root_path"/opt/kubeadm/kubeadm.yaml --upload-certs --ignore-preflight-errors=NumCPU --ignore-preflight-errors=Mem --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5 > /dev/null
  do
    backup_kube_vip_manifest_if_present
    echo "failed to apply kubeadm init, applying reset";
    do_kubeadm_reset
    echo "retrying in 10s"
    sleep 10;
    restore_kube_vip_manifest_after_reset
  done;
else
  until kubeadm init --config "$root_path"/opt/kubeadm/kubeadm.yaml --upload-certs --ignore-preflight-errors=NumCPU --ignore-preflight-errors=Mem --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5 > /dev/null
  do
    backup_kube_vip_manifest_if_present
    echo "failed to apply kubeadm init, applying reset";
    do_kubeadm_reset
    echo "retrying in 10s"
    sleep 10;
    restore_kube_vip_manifest_after_reset
  done;
fi