#!/bin/bash

exec   > >(tee -ia /var/log/kube-post-init.log)
exec  2> >(tee -ia /var/log/kube-post-init.log >& 2)
exec 19>> /var/log/kube-post-init.log

export BASH_XTRACEFD="19"
set -ex

host=$1

export KUBECONFIG=/etc/kubernetes/admin.conf
kubectl get cm -n kube-system kubeadm-config -o yaml | sed "/^\( *\)kind: ClusterConfiguration.*/a\ \ \ \ controlPlaneEndpoint: ${host}" > /opt/kubeadm/kubeadm-config.yaml
kubectl delete cm -n kube-system kubeadm-config && kubectl apply -f /opt/kubeadm/kubeadm-config.yaml && rm /opt/kubeadm/kubeadm-config.yaml

echo "updated kubeadm-config controlPlaneEndpoint with ${host}"

secret=$(kubectl get secrets kubeadm-certs -n kube-system -o jsonpath="{['metadata']['ownerReferences'][0]['name']}")
kubectl get secrets -n kube-system "${secret}" -o yaml | kubectl apply set-last-applied --create-annotation=true -f -
kubectl get secrets -n kube-system "${secret}" -o yaml | sed '/^\( *\)expiration.*/d' | kubectl apply -f -

echo "updated kubeadm-certs expiration"

kubectl get cm -n kube-public cluster-info -o yaml | sed "s/^\( *\)server.*/\1server: https:\/\/${host}:6443/" > /opt/kubeadm/cluster-info.yaml
kubectl delete cm -n kube-public cluster-info && kubectl apply -f /opt/kubeadm/cluster-info.yaml && rm /opt/kubeadm/cluster-info.yaml

echo "updated cluster-info server with ${host}"