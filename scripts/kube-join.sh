#!/bin/bash

exec   > >(tee -ia /var/log/kube-join.log)
exec  2> >(tee -ia /var/log/kube-join.log >& 2)
exec 19>> /var/log/kube-join.log

export BASH_XTRACEFD="19"
set -ex

until kubeadm join --config /opt/kubeadm/kubeadm.yaml --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5 > /dev/null
do
  echo "failed to apply kubeadm join, will retry in 10s";
  sleep 10;
done;