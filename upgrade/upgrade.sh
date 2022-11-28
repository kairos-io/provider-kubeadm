#!/bin/sh

set -ex

NODE_ROLE=$1

run_upgrade() {
    echo "running upgrade process on $NODE_ROLE"

    old_version=$(cat /opt/sentinel_kubeadmversion)
    echo "found last deployed version $old_version"

    current_version=$(kubeadm version -o short)
    echo "found current deployed version $current_version"

    until [ "$current_version" = "$old_version" ]
    do
        upgrade_command="kubeadm upgrade node"
        if [ "$NODE_ROLE" != "worker" ]
        then
            master_api_version=$(kubectl --kubeconfig /etc/kubernetes/admin.conf get cm kubeadm-config -n kube-system -o yaml | grep kubernetesVersion | tr -s " " | cut -d' ' -f 3)
            if [ "$master_api_version" = "" ]; then
              echo "master api version empty, retrying in 60 seconds"
              sleep 60
              continue
            fi
            if [ "$master_api_version" = "$old_version" ]
            then
                upgrade_command="kubeadm upgrade apply -y $current_version"
            fi
        fi
        echo "upgrading node from $old_version to $current_version using command: $upgrade_command"
        if $upgrade_command
        then
            echo "$current_version" > /opt/sentinel_kubeadmversion
            old_version=$current_version
            echo "upgrade success"
        else
            echo "upgrade failed, retrying in 60 seconds"
            sleep 60
        fi
    done
}

run_upgrade