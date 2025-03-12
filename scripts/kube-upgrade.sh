#!/bin/bash

exec   > >(tee -ia /var/log/kube-upgrade.log)
exec  2> >(tee -ia /var/log/kube-upgrade.log >& 2)
exec 19>> /var/log/kube-upgrade.log

set -x

NODE_ROLE=$1

root_path=$2
PROXY_CONFIGURED=$3
proxy_http=$4
proxy_https=$5
proxy_no=$6

export PATH="$PATH:$root_path/usr/bin"
export PATH="$PATH:$root_path/usr/local/bin"

if [ -n "$proxy_no" ]; then
  export NO_PROXY=$proxy_no
  export no_proxy=$proxy_no
fi

if [ -n "$proxy_http" ]; then
  export HTTP_PROXY=$proxy_http
  export http_proxy=$proxy_http
fi

if [ -n "$proxy_https" ]; then
  export https_proxy=$proxy_https
  export HTTPS_PROXY=$proxy_https
fi

CURRENT_NODE_NAME=$(cat /etc/hostname)

export KUBECONFIG=/etc/kubernetes/admin.conf

get_current_upgrading_node_name() {
  kubectl get configmap upgrade-lock -n kube-system --kubeconfig /etc/kubernetes/admin.conf -o jsonpath="{['data']['node']}"
}

delete_lock_config_map() {
  # Delete the configmap lock once the upgrade completes
  if [ "$NODE_ROLE" != "worker" ]
  then
    kubectl --kubeconfig /etc/kubernetes/admin.conf delete configmap upgrade-lock -n kube-system
  fi
}

restart_containerd() {
  if systemctl cat spectro-containerd >/dev/null 2<&1; then
    systemctl restart spectro-containerd
  fi

  if systemctl cat containerd >/dev/null 2<&1; then
    systemctl restart containerd
  fi
}

upgrade_kubelet() {
  echo "upgrading kubelet"
  systemctl stop kubelet
  cp /opt/kubeadm/bin/kubelet "$root_path"/usr/local/bin/kubelet
  systemctl daemon-reload && systemctl restart kubelet
  restart_containerd
  echo "kubelet upgraded"
}

apply_new_kubeadm_config() {
  kubectl get cm kubeadm-config -n kube-system -o jsonpath="{['data']['ClusterConfiguration']}" --kubeconfig /etc/kubernetes/admin.conf > "$root_path"/opt/kubeadm/existing-cluster-config.yaml
  kubeadm init phase upload-config kubeadm --config "$root_path"/opt/kubeadm/cluster-config.yaml
}

revert_kubeadm_config() {
  kubeadm init phase upload-config kubeadm --config "$root_path"/opt/kubeadm/existing-cluster-config.yaml
}

run_upgrade() {
    echo "running upgrade process on $NODE_ROLE"

    old_version=$(cat "$root_path"/opt/sentinel_kubeadmversion)
    echo "found last deployed version $old_version"

    current_version=$(kubeadm version -o short)
    echo "found current deployed version $current_version"

    # Check if the current kubeadm version is equal to the stored kubeadm version
    # If yes, do nothing
    if [ "$current_version" = "$old_version" ]
    then
      echo "node is on latest version"
      exit 0
    fi

    # Proceed to do upgrade operation

    # Try to create an empty configmap in default namespace which will act as a lock, until it succeeds.
    # Once a node creates a configmap, other nodes will remain at this step until the first node deletes the configmap when upgrade completes.
    if [ "$NODE_ROLE" != "worker" ]
    then
      until kubectl --kubeconfig /etc/kubernetes/admin.conf create configmap upgrade-lock -n kube-system --from-literal=node="${CURRENT_NODE_NAME}" > /dev/null
      do
        upgrade_node=$(get_current_upgrading_node_name)
        if [ "$upgrade_node" = "$CURRENT_NODE_NAME" ]; then
          echo "resuming upgrade"
          break
        fi
        echo "failed to create configmap for upgrade lock, upgrading is going on the node ${upgrade_node}, retrying in 60 sec"
        sleep 60
      done
    fi

    # Upgrade loop, runs until both stored and current is same
    until [ "$current_version" = "$old_version" ]
    do
        # worker node will always run 'upgrade node'
        # control plane will also run `upgrade node' except one node will run 'upgrade apply' based on who acquires lock
        upgrade_command="kubeadm upgrade node"
        if [ "$PROXY_CONFIGURED" = true ]; then
          up=("kubeadm upgrade node")
          upgrade_command="${up[*]}"
        fi

        if [ "$NODE_ROLE" != "worker" ]
        then
            # The current api version is stored in kubeadm-config configmap
            # This is being used to check whether the current cp node will run 'upgrade apply' or not
            master_api_version=$(kubectl --kubeconfig /etc/kubernetes/admin.conf get cm kubeadm-config -n kube-system -o yaml | grep -m 1 kubernetesVersion | tr -s " " | cut -d' ' -f 3)
            if [ "$master_api_version" = "" ]; then
              echo "master api version empty, retrying in 60 seconds"
              sleep 60
              continue
            fi

            if [ "$master_api_version" = "$old_version" ]
            then
                apply_new_kubeadm_config
                upgrade_command="kubeadm upgrade apply -y $current_version"
                if [ "$PROXY_CONFIGURED" = true ]; then
                  up=("kubeadm upgrade apply -y ${current_version}")
                  upgrade_command="${up[*]}"
                fi
            fi
        fi
        echo "upgrading node from $old_version to $current_version using command: $upgrade_command"

        if sudo -E bash -c "$upgrade_command"
        then
            # Update current client version in the version file
            echo "$current_version" > "$root_path"/opt/sentinel_kubeadmversion
            old_version=$current_version

            delete_lock_config_map
            echo "upgrade success"
        else
            echo "upgrade failed"
            if echo "$upgrade_command" | grep -q "apply"; then
              echo "reverting kubeadm config"
              revert_kubeadm_config
            fi
            echo "retrying in 60 seconds"
            sleep 60
        fi
    done
    upgrade_kubelet
}

run_upgrade