# kubeadm Troubleshooting

## Overview

This skill provides comprehensive troubleshooting guidance for common provider-kubeadm issues including CNI problems, control plane failures, proxy issues, token expiration, and deployment mode complications.

## Common Issues

### Issue 1: NetworkPluginNotReady (Most Common)

**Symptom**:
```bash
kubectl get nodes
# NAME     STATUS     ROLES           AGE   VERSION
# node-1   NotReady   control-plane   5m    v1.28.0
```

**Root Cause**: No CNI installed after kubeadm init.

**Diagnosis**:
```bash
kubectl describe node node-1
# Conditions:
#   Ready            False   NetworkPluginNotReady   container runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: cni plugin not initialized

kubectl get pods -A
# NAMESPACE     NAME                          READY   STATUS    RESTARTS   AGE
# kube-system   coredns-5d78c9869d-abc123     0/1     Pending   0          5m
# kube-system   coredns-5d78c9869d-def456     0/1     Pending   0          5m
```

**Solution**:
```bash
# Install Flannel CNI
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Wait for CNI pods
kubectl get pods -n kube-flannel -w

# Verify nodes become Ready
kubectl get nodes
# NAME     STATUS   ROLES           AGE   VERSION
# node-1   Ready    control-plane   7m    v1.28.0
```

**Prevention**: Always install CNI immediately after kubeadm init (see `02-cni-installation.md`).

---

### Issue 2: Control Plane Connection Refused

**Symptom**:
```bash
kubeadm join 10.0.1.100:6443 --token ...
# error: unable to connect to apiServerEndpoint: dial tcp 10.0.1.100:6443: connect: connection refused
```

**Root Causes**:
1. API server not running on init node
2. Firewall blocking port 6443
3. Wrong control_plane_host IP

**Diagnosis**:
```bash
# On init node - check API server running
kubectl get pods -n kube-system -l component=kube-apiserver
# NAME                          READY   STATUS    RESTARTS   AGE
# kube-apiserver-control-1      1/1     Running   0          10m

# Check API server listening
netstat -tuln | grep 6443
# tcp        0      0 0.0.0.0:6443            0.0.0.0:*               LISTEN

# From joining node - test connectivity
curl -k https://10.0.1.100:6443/healthz
# ok

# If connection refused - check firewall
firewall-cmd --list-ports
# 6443/tcp should be present
```

**Solution**:
```bash
# On control plane node - open port 6443
firewall-cmd --permanent --add-port=6443/tcp
firewall-cmd --reload

# Verify API server logs if not running
journalctl -u kubelet -f
```

---

### Issue 3: CNI Subnet Mismatch

**Symptom**:
```bash
kubectl get pods -n kube-flannel
# NAME                    READY   STATUS             RESTARTS   AGE
# kube-flannel-ds-amd64   0/1     CrashLoopBackOff   5          3m
```

**Root Cause**: Flannel network config doesn't match kubeadm podSubnet.

**Diagnosis**:
```bash
# Check kubeadm config
kubectl get cm -n kube-system kubeadm-config -o yaml
# networking:
#   podSubnet: 10.244.0.0/16

# Check Flannel config
kubectl get cm -n kube-flannel kube-flannel-cfg -o yaml
# net-conf.json: |
#   {
#     "Network": "10.100.0.0/16"    # ❌ Mismatch!
#   }

# Check Flannel logs
kubectl logs -n kube-flannel kube-flannel-ds-amd64
# Error: subnet "10.244.0.0/16" does not contain PodCIDR "10.100.0.0/16"
```

**Solution**:
```bash
# Delete Flannel
kubectl delete -f kube-flannel.yml

# Download and edit manifest
curl -sO https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Edit to match podSubnet
sed -i 's|"Network": "10.100.0.0/16"|"Network": "10.244.0.0/16"|' kube-flannel.yml

# Reapply
kubectl apply -f kube-flannel.yml
```

---

### Issue 4: Token Expired

**Symptom**:
```bash
kubeadm join 10.0.1.100:6443 --token abcdef.0123456789abcdef ...
# error: unable to authenticate: token id "abcdef" expired
```

**Root Cause**: Bootstrap tokens expire after 24 hours by default.

**Diagnosis**:
```bash
# On control plane - list tokens
kubeadm token list
# TOKEN                     TTL         EXPIRES               USAGES
# abcdef.0123456789abcdef   <invalid>   2024-02-08T10:00:00Z  authentication,signing
```

**Solution**:
```bash
# Generate new token
kubeadm token create --ttl 24h
# ghijkl.0123456789ghijkl

# Get CA cert hash (needed for join)
openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | \
  openssl rsa -pubin -outform der 2>/dev/null | \
  openssl dgst -sha256 -hex | sed 's/^.* //'
# 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef

# Join with new token
kubeadm join 10.0.1.100:6443 \
  --token ghijkl.0123456789ghijkl \
  --discovery-token-ca-cert-hash sha256:1234567890abcdef... \
  -v=5
```

**Prevention**: Create tokens with longer TTL or use certificate-based join.

---

### Issue 5: Proxy Blocks Container Registry

**Symptom**:
```bash
kubeadm init --config kubeadm.yaml
# [ERROR ImagePull]: failed to pull image registry.k8s.io/kube-apiserver:v1.28.0: Get "https://registry.k8s.io/v2/": proxyconnect tcp: dial tcp 192.168.1.1:8080: connect: connection refused
```

**Root Cause**: Proxy is set but blocks registry access.

**Diagnosis**:
```bash
# Check proxy settings
cat /etc/default/kubelet
# HTTP_PROXY=http://proxy.corp.com:8080
# HTTPS_PROXY=http://proxy.corp.com:8080
# NO_PROXY=localhost,127.0.0.1,10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local

# Test registry access
curl -I https://registry.k8s.io/
# ❌ Connection timeout or refused
```

**Solution Option 1**: Add registry to NO_PROXY
```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1,registry.k8s.io,gcr.io,quay.io"
```

**Solution Option 2**: Pre-import images (air-gap)
```yaml
cluster:
  import_images: true
```

---

### Issue 6: Pods Cannot Reach Services

**Symptom**:
```bash
kubectl exec test-pod -- curl kubernetes.default.svc
# curl: (28) Connection timed out after 30001 milliseconds
```

**Root Cause**: Service subnet going via proxy instead of direct.

**Diagnosis**:
```bash
# Check NO_PROXY
cat /etc/default/kubelet | grep NO_PROXY
# NO_PROXY=localhost,127.0.0.1
# ❌ Missing service CIDR and .svc domains

# From pod, check DNS resolution
kubectl exec test-pod -- nslookup kubernetes.default.svc
# Server:    10.96.0.10
# Address:   10.96.0.10:53
#
# Name:      kubernetes.default.svc.cluster.local
# Address:   10.96.0.1
# ✅ DNS works

# But connection fails
kubectl exec test-pod -- curl -m 5 https://10.96.0.1
# ❌ Timeout - traffic going via proxy
```

**Solution**: Provider should auto-append, but verify:
```bash
# Check provider added service CIDR to NO_PROXY
cat /run/systemd/system/containerd.service.d/http-proxy.conf
# NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1

# If missing, restart containerd with correct NO_PROXY
systemctl daemon-reload
systemctl restart containerd
```

---

### Issue 7: Control Plane Certificate Key Expired

**Symptom**:
```bash
kubeadm join 10.0.1.100:6443 --control-plane --certificate-key abc123...
# error: unable to fetch the kubeadm-certs Secret: Secret "kubeadm-certs" was not found in the "kube-system" Namespace
```

**Root Cause**: Certificate key expires after 2 hours.

**Diagnosis**:
```bash
# On control plane - check if secret exists
kubectl get secret -n kube-system kubeadm-certs
# Error from server (NotFound): secrets "kubeadm-certs" not found
# ❌ Certificate key expired
```

**Solution**:
```bash
# Generate new certificate key
kubeadm init phase upload-certs --upload-certs
# [upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
# [upload-certs] Using certificate key:
# def456ghi789...

# Join with new certificate key
kubeadm join 10.0.1.100:6443 \
  --token ghijkl.0123456789ghijkl \
  --discovery-token-ca-cert-hash sha256:1234567890abcdef... \
  --control-plane \
  --certificate-key def456ghi789... \
  -v=5
```

---

### Issue 8: Wrong Containerd Socket (Agent Mode)

**Symptom**:
```bash
# Agent mode (STYLUS_ROOT=/persistent/spectro)
kubeadm init --config kubeadm.yaml
# [ERROR CRI]: container runtime is not running: output: E0208 10:00:00.000000   12345 remote_runtime.go:925] "Status from runtime service failed" err="rpc error: code = Unavailable desc = connection error: desc = \"transport: Error while dialing dial unix /run/containerd/containerd.sock: connect: no such file or directory\""
```

**Root Cause**: Config specifies standard containerd socket but agent mode uses spectro-containerd.

**Diagnosis**:
```bash
# Check STYLUS_ROOT
echo $STYLUS_ROOT
# /persistent/spectro  ← Agent mode

# Check kubeadm config
cat /opt/kubeadm/kubeadm.yaml
# nodeRegistration:
#   criSocket: unix:///run/containerd/containerd.sock
#   ❌ Wrong socket for agent mode

# Check spectro-containerd running
systemctl status spectro-containerd
# Active: active (running)

# Check socket exists
ls -la /run/spectro-containerd/spectro-containerd.sock
# srw-rw---- 1 root root 0 Feb  8 10:00 /run/spectro-containerd/spectro-containerd.sock
```

**Solution**: Update kubeadm config
```yaml
nodeRegistration:
  criSocket: unix:///run/spectro-containerd/spectro-containerd.sock
```

---

### Issue 9: IP Detection Issues (Multi-Interface)

**Symptom**:
```bash
# Node has eth0 (192.168.1.100) and eth1 (10.0.1.100)
# After join, kubelet uses wrong IP

kubectl get nodes -o wide
# NAME     STATUS   ROLES    AGE   VERSION   INTERNAL-IP
# node-1   Ready    <none>   2m    v1.28.0   192.168.1.100
# ❌ Using eth0 (NAT interface) instead of eth1 (data plane)
```

**Root Cause**: kubelet auto-detected wrong interface.

**Diagnosis**:
```bash
# Check kubelet logs
journalctl -u kubelet | grep "node ip"
# Using Node IP: "192.168.1.100"
```

**Solution**: Force kubelet to use specific IP
```yaml
# In kubeadm config
nodeRegistration:
  kubeletExtraArgs:
    node-ip: "10.0.1.100"    # Force data plane IP
```

---

### Issue 10: Bash Not Installed (Edge Systems)

**Symptom**:
```bash
/opt/kubeadm/kube-init.sh
# /bin/bash: bad interpreter: No such file or directory
```

**Root Cause**: Minimal edge OS doesn't include bash.

**Diagnosis**:
```bash
which bash
# ❌ No output - bash not installed

ls -la /bin/sh
# /bin/sh -> dash  ← POSIX shell only
```

**Solution**: Install bash
```bash
# Alpine/Kairos
apk add bash

# Debian/Ubuntu
apt-get install bash

# RHEL/Rocky
yum install bash
```

**Alternative**: Rewrite scripts for POSIX sh (not currently supported by provider-kubeadm).

---

## Diagnostic Commands

### Check Cluster Status
```bash
# Nodes
kubectl get nodes -o wide

# Pods
kubectl get pods -A -o wide

# Services
kubectl get svc -A

# Component status
kubectl get componentstatuses
```

### Check Control Plane Components
```bash
# Control plane pods
kubectl get pods -n kube-system -l tier=control-plane

# API server logs
kubectl logs -n kube-system kube-apiserver-control-1

# etcd status
kubectl exec -n kube-system etcd-control-1 -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  endpoint health
```

### Check CNI
```bash
# CNI pods
kubectl get pods -n kube-flannel
kubectl get pods -n calico-system

# CNI logs
kubectl logs -n kube-flannel kube-flannel-ds-amd64

# Check CNI interfaces
ip link show | grep -E "(flannel|cali|cilium)"

# Check routes
ip route | grep -E "(flannel|cali|cilium)"
```

### Check Networking
```bash
# Pod IP assignment
kubectl get pods -A -o wide

# Test pod-to-pod
kubectl run test-1 --image=nginx
kubectl run test-2 --image=nginx
kubectl exec test-1 -- ping -c 3 $(kubectl get pod test-2 -o jsonpath='{.status.podIP}')

# Test service connectivity
kubectl exec test-1 -- curl kubernetes.default.svc

# DNS resolution
kubectl exec test-1 -- nslookup kubernetes.default.svc
```

### Check Proxy Configuration
```bash
# Kubelet proxy
cat /etc/default/kubelet

# Containerd proxy
cat /run/systemd/system/containerd.service.d/http-proxy.conf

# Test proxy
curl -I --proxy http://proxy.corp.com:8080 https://registry.k8s.io/
```

### Check Logs
```bash
# kubelet
journalctl -u kubelet -f

# containerd
journalctl -u containerd -f

# kubeadm init logs
cat /var/log/kube-init.log

# kubeadm join logs
cat /var/log/kube-join.log
```

---

## Integration Points

### With CNI

- Most issues stem from missing or misconfigured CNI
- Always verify CNI is installed and running
- CNI subnet must match kubeadm podSubnet

### With Proxy

- Proxy misconfigurations block registry access and pod communication
- Verify NO_PROXY includes all internal networks
- Test proxy connectivity before kubeadm init

### With Deployment Modes

- Agent mode requires spectro-containerd socket
- File paths differ between appliance and agent modes
- Verify STYLUS_ROOT is set correctly

## Reference Examples

**Troubleshooting Scripts**:
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-reset.sh` - Reset cluster for retry
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-init.sh` - Retry logic in init script

**Log Files**:
- `/var/log/kube-init.log` - Init logs (bash XTRACEFD)
- `/var/log/kube-join.log` - Join logs
- `/var/log/kube-import-images.log` - Image import logs

## Related Skills

- See `provider-kubeadm:02-cni-installation` for CNI installation details
- See `provider-kubeadm:04-proxy-configuration` for proxy setup
- See `provider-kubeadm:05-cluster-roles` for token management
- See `provider-kubeadm:06-deployment-modes` for agent vs appliance mode
- See `provider-kubeadm:07-networking` for networking architecture
- See `provider-kubeadm:10-bash-dependency` for bash requirement details

## Documentation References

**kubeadm Troubleshooting**:
- https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/troubleshooting-kubeadm/
- https://kubernetes.io/docs/tasks/debug-application-cluster/

**CNI Troubleshooting**:
- Flannel: https://github.com/flannel-io/flannel/blob/master/Documentation/troubleshooting.md
- Calico: https://docs.tigera.io/calico/latest/operations/troubleshoot/
