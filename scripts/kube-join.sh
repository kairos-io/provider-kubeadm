#!/bin/bash

exec   > >(tee -ia /var/log/kube-join.log)
exec  2> >(tee -ia /var/log/kube-join.log >& 2)
exec 19>> /var/log/kube-join.log

export BASH_XTRACEFD="19"
set -ex

NODE_ROLE=$1

root_path=$2
PROXY_CONFIGURED=$3
proxy_http=$4
proxy_https=$5
proxy_no=$6

export PATH="$PATH:$root_path/usr/bin"
export PATH="$PATH:$root_path/usr/local/bin"

KUBE_VIP_LOC="/etc/kubernetes/manifests/kube-vip.yaml"

# Prevents infinite retry loops (original used until loop)
MAX_RETRIES=3
# Time to wait for etcd/static pods to become ready
ETCD_HEALTH_TIMEOUT=60
RETRY_DELAY=30

restart_containerd() {
  if systemctl cat spectro-containerd >/dev/null 2<&1; then
    systemctl restart spectro-containerd
  fi

  if systemctl cat containerd >/dev/null 2<&1; then
    systemctl restart containerd
  fi
}

# Function to wait for static pods to be created and running
wait_for_static_pods() {
  local timeout=$1
  local start_time=$(date +%s)

  echo "Waiting for static pods to be created and running..."

  # Check if crictl is available
  if ! command -v crictl >/dev/null 2>&1; then
    echo "crictl not available, checking only static pod manifests"
    # Fallback: just check if manifests exist and wait
    while [ $(($(date +%s) - start_time)) -lt $timeout ]; do
      if [ -f /etc/kubernetes/manifests/etcd.yaml ] && [ -f /etc/kubernetes/manifests/kube-apiserver.yaml ] && [ -f /etc/kubernetes/manifests/kube-controller-manager.yaml ] && [ -f /etc/kubernetes/manifests/kube-scheduler.yaml ]; then
        echo "Static pod manifests are ready, waiting additional time for pods to stabilize..."
        sleep 15
        return 0
      fi
      sleep 5
    done
    echo "Timeout waiting for static pod manifests after ${timeout}s"
    return 1
  fi

  while [ $(($(date +%s) - start_time)) -lt $timeout ]; do
    # Check if all required static pod manifests exist
    if [ -f /etc/kubernetes/manifests/etcd.yaml ] && [ -f /etc/kubernetes/manifests/kube-apiserver.yaml ] && [ -f /etc/kubernetes/manifests/kube-controller-manager.yaml ] && [ -f /etc/kubernetes/manifests/kube-scheduler.yaml ]; then
      # Check if etcd container is running
      local etcd_container_id=$(crictl ps --name etcd --quiet 2>/dev/null)
      if [ -n "$etcd_container_id" ] && crictl ps --id "$etcd_container_id" --format table | grep -q "Running"; then
        echo "Static pods are running, waiting additional time for etcd to stabilize..."
        sleep 10
        return 0
      fi
    fi
    sleep 5
  done

  echo "Timeout waiting for static pods to be ready after ${timeout}s"
  return 1
}

# Function to check if etcd is healthy (only if etcdctl and crictl are available)
check_etcd_health_if_available() {
  local timeout=$1
  local start_time=$(date +%s)

  # Check if etcdctl is available
  if ! command -v etcdctl >/dev/null 2>&1; then
    echo "etcdctl not available, skipping etcd health check"
    return 0
  fi

  # Check if crictl is available
  if ! command -v crictl >/dev/null 2>&1; then
    echo "crictl not available, skipping etcd health check"
    return 0
  fi

  echo "Checking etcd health with etcdctl..."

  while [ $(($(date +%s) - start_time)) -lt $timeout ]; do
    # Check if etcd container is running
    local etcd_container_id=$(crictl ps --name etcd --quiet 2>/dev/null)
    if [ -n "$etcd_container_id" ] && crictl ps --id "$etcd_container_id" --format table | grep -q "Running"; then
      # Try to connect to local etcd directly
      if timeout 10s etcdctl endpoint health --endpoints=https://127.0.0.1:2379 --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key >/dev/null 2>&1; then
        echo "etcd is healthy"
        return 0
      fi
    fi
    sleep 5
  done

  echo "etcd health check timed out after ${timeout}s"
  return 1
}

# Function to clean up cluster state before retry
cleanup_cluster_state() {
  local node_name=$(hostname)
  echo "Cleaning up cluster state for node: $node_name"

  # Try to remove node from cluster if possible
  if [ -f /etc/kubernetes/admin.conf ]; then
    kubectl delete node $node_name --kubeconfig=/etc/kubernetes/admin.conf --ignore-not-found=true 2>/dev/null || true
  fi

  # For control plane nodes, try to remove etcd member
  if [ "$NODE_ROLE" = "controlplane" ]; then
    # This would need to be done from another control plane node
    echo "Note: etcd member cleanup should be done from another control plane node"
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
  if [ -f "$KUBE_VIP_LOC" ] && [ "$NODE_ROLE" != "worker" ]; then
    cp $KUBE_VIP_LOC "$root_path"/opt/kubeadm/kube-vip.yaml
  fi
}

restore_kube_vip_manifest_after_reset() {
  if [ -f "$root_path/opt/kubeadm/kube-vip.yaml" ] && [ "$NODE_ROLE" != "worker" ]; then
    mkdir -p "$root_path"/etc/kubernetes/manifests
    cp "$root_path"/opt/kubeadm/kube-vip.yaml $KUBE_VIP_LOC
  fi
}

# Function to attempt kubeadm join with proper error handling
attempt_kubeadm_join() {
  local attempt=$1
  local max_retries=$2

  echo "Attempting kubeadm join (attempt $attempt/$max_retries)"

  if [ "$PROXY_CONFIGURED" = true ]; then
    HTTP_PROXY=$proxy_http http_proxy=$proxy_http HTTPS_PROXY=$proxy_https https_proxy=$proxy_https NO_PROXY=$proxy_no no_proxy=$proxy_no kubeadm join --config "$root_path"/opt/kubeadm/kubeadm.yaml --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5
  else
    kubeadm join --config "$root_path"/opt/kubeadm/kubeadm.yaml --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests -v=5
  fi
}

# Main join logic with intelligent retry
retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
  retry_count=$((retry_count + 1))

  if attempt_kubeadm_join $retry_count $MAX_RETRIES; then
    echo "kubeadm join completed successfully"
    break
  else
    echo "kubeadm join failed on attempt $retry_count"

    # Check if this is an etcd health check failure
    if grep -q "etcd cluster is not healthy" /var/log/kube-join.log 2>/dev/null; then
      echo "Detected etcd health check failure"

      # For first attempt, try to wait for static pods to be ready
      if [ $retry_count -eq 1 ]; then
        echo "Waiting for static pods to be ready before retry..."
        if wait_for_static_pods $ETCD_HEALTH_TIMEOUT; then
          echo "Static pods are ready, checking etcd health if available..."
          if check_etcd_health_if_available 30; then
            echo "etcd is healthy, retrying join without reset"
            continue
          else
            echo "etcd health check failed or etcdctl not available, retrying join without reset"
            continue
          fi
        else
          echo "Static pods not ready, proceeding with reset"
        fi
      fi
    fi

    # If we've reached max retries, exit with error
    if [ $retry_count -eq $MAX_RETRIES ]; then
      echo "Maximum retry attempts ($MAX_RETRIES) reached. Exiting."
      exit 1
    fi

    # Clean up cluster state before reset
    cleanup_cluster_state

    # Reset and retry
    backup_kube_vip_manifest_if_present
    echo "Resetting kubeadm and retrying in ${RETRY_DELAY}s"
    do_kubeadm_reset
    echo "Retrying in ${RETRY_DELAY}s"
    sleep $RETRY_DELAY
    restore_kube_vip_manifest_after_reset
  fi
done
