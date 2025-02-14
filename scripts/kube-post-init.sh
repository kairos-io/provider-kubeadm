#!/bin/bash

exec   > >(tee -ia /var/log/kube-post-init.log)
exec  2> >(tee -ia /var/log/kube-post-init.log >& 2)
exec 19>> /var/log/kube-post-init.log

export BASH_XTRACEFD="19"
set -x

root_path=$1

export KUBECONFIG=/etc/kubernetes/admin.conf
export PATH="$PATH:$root_path/usr/bin"
export PATH="$PATH:$root_path/usr/local/bin"

while true;
do
  secret=$(kubectl get secrets kubeadm-certs -n kube-system -o jsonpath="{['metadata']['ownerReferences'][0]['name']}")
  if [ "$secret" != "" ];
  then
    kubectl get secrets -n kube-system "${secret}" -o yaml | kubectl apply set-last-applied --create-annotation=true -f -
    kubectl get secrets -n kube-system "${secret}" -o yaml | sed '/^\( *\)expiration.*/d' | kubectl apply -f -
    echo "updated kubeadm-certs expiration"
    break
  else
    echo "failed to get kubeadm-certs ownerReferences, trying in 30 sec"
    sleep 30
  fi
done