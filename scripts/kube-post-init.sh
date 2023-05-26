#!/bin/bash

exec   > >(tee -ia /var/log/kube-post-init.log)
exec  2> >(tee -ia /var/log/kube-post-init.log >& 2)
exec 19>> /var/log/kube-post-init.log

export BASH_XTRACEFD="19"
set -ex

export KUBECONFIG=/etc/kubernetes/admin.conf

secret=$(kubectl get secrets kubeadm-certs -n kube-system -o jsonpath="{['metadata']['ownerReferences'][0]['name']}")
kubectl get secrets -n kube-system "${secret}" -o yaml | kubectl apply set-last-applied --create-annotation=true -f -
kubectl get secrets -n kube-system "${secret}" -o yaml | sed '/^\( *\)expiration.*/d' | kubectl apply -f -

echo "updated kubeadm-certs expiration"