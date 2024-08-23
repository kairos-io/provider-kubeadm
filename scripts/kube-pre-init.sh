#!/bin/bash

set -x

sysctl --system
modprobe overlay
modprobe br_netfilter
systemctl daemon-reload

systemctl enable kubelet && systemctl start kubelet

if systemctl cat spectro-containerd >/dev/null 2<&1; then
  systemctl enable spectro-containerd && systemctl start spectro-containerd
fi

if systemctl cat containerd >/dev/null 2<&1; then
  systemctl enable containerd && systemctl start containerd
fi