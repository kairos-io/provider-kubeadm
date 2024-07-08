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

do_kubeadm_reset() {
  kubeadm reset -f
  iptables -F && iptables -t nat -F && iptables -t mangle -F && iptables -X && rm -rf /etc/kubernetes/etcd /etc/kubernetes/manifests /etc/kubernetes/pki
  rm -rf "$root_path"/opt/spectro/cni/net.d
  systemctl daemon-reload
  systemctl restart spectro-containerd
}

backup_kube_vip_manifest_if_present() {
  if [ -f "$KUBE_VIP_LOC" ]; then
    cp $KUBE_VIP_LOC "$root_path"/opt/kubeadm/kube-vip.yaml
  fi
}

restore_kube_vip_manifest_after_reset() {
  if [ -f "$root_path/opt/kubeadm/kube-vip.yaml" ]; then
      mkdir -p /etc/kubernetes/manifests
      cp "$root_path"/opt/kubeadm/kube-vip.yaml $KUBE_VIP_LOC
  fi
}

if [ "$PROXY_CONFIGURED" = true ]; then
  until HTTP_PROXY=$proxy_http http_proxy=$proxy_http HTTPS_PROXY=$proxy_https https_proxy=$proxy_https NO_PROXY=$proxy_no no_proxy=$proxy_no kubeadm init --config "$root_path"/opt/kubeadm/kubeadm.yaml --upload-certs --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5 > /dev/null
  do
    backup_kube_vip_manifest_if_present
    echo "failed to apply kubeadm init, applying reset";
    do_kubeadm_reset
    echo "retrying in 10s"
    sleep 10;
    restore_kube_vip_manifest_after_reset
  done;
else
  until kubeadm init --config "$root_path"/opt/kubeadm/kubeadm.yaml --upload-certs --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5 > /dev/null
  do
    backup_kube_vip_manifest_if_present
    echo "failed to apply kubeadm init, applying reset";
    do_kubeadm_reset
    echo "retrying in 10s"
    sleep 10;
    restore_kube_vip_manifest_after_reset
  done;
fi