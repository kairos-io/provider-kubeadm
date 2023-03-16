#!/bin/bash

exec   > >(tee -ia /var/log/kube-init.log)
exec  2> >(tee -ia /var/log/kube-init.log >& 2)
exec 19>> /var/log/kube-init.log

export BASH_XTRACEFD="19"
set -ex

until kubeadm init --config /opt/kubeadm/kubeadm.yaml --upload-certs --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5 > /dev/null
do
  echo "failed to apply kubeadm init, will retry in 10s";
  sleep 10;
done;