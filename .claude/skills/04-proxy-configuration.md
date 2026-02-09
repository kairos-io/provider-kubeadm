# kubeadm Proxy Configuration

## Overview

Provider-kubeadm supports HTTP/HTTPS proxy configuration for edge deployments behind corporate proxies. Proxy settings are applied to kubelet and containerd, with automatic NO_PROXY calculation to exclude internal Kubernetes networks.

## Key Concepts

### Proxy Configuration Files

Provider-kubeadm creates two proxy configuration files:

1. **`/etc/default/kubelet`** - Kubelet environment variables
2. **`/run/systemd/system/containerd.service.d/http-proxy.conf`** - Containerd systemd override

### NO_PROXY Auto-Calculation

The provider automatically appends Kubernetes internal networks to user NO_PROXY:

**Auto-appended values**:
- **podSubnet**: From kubeadm `networking.podSubnet` (e.g., `10.244.0.0/16`)
- **serviceSubnet**: From kubeadm `networking.serviceSubnet` (e.g., `10.96.0.0/12`)
- **k8s service domains**: `.svc,.svc.cluster,.svc.cluster.local`

**Constant** (same as provider-k3s/rke2):
```go
k8sNoProxy = ".svc,.svc.cluster,.svc.cluster.local"
```

### Proxy vs Non-Proxy Modes

**Non-Proxy Mode** (default):
- No `HTTP_PROXY`/`HTTPS_PROXY` in `cluster.env`
- No proxy files created
- Direct internet access

**Proxy Mode**:
- `HTTP_PROXY` and/or `HTTPS_PROXY` set in `cluster.env`
- Proxy files created automatically
- All external traffic via proxy
- Internal Kubernetes traffic bypasses proxy (NO_PROXY)

## Implementation Patterns

### Pattern 1: Basic Proxy Configuration

**Use Case**: Simple corporate proxy with standard configuration.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  cluster_token: "abcdef.0123456789abcdef"
  control_plane_host: "10.0.1.100"
  role: init
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    HTTPS_PROXY: "https://proxy.corp.com:8443"
    NO_PROXY: "localhost,127.0.0.1,.corp.com"
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated `/etc/default/kubelet`**:
```bash
HTTP_PROXY=http://proxy.corp.com:8080
HTTPS_PROXY=https://proxy.corp.com:8443
NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1,.corp.com
```

**Generated `/run/systemd/system/containerd.service.d/http-proxy.conf`**:
```ini
[Service]
Environment="HTTP_PROXY=http://proxy.corp.com:8080"
Environment="HTTPS_PROXY=https://proxy.corp.com:8443"
Environment="NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1,.corp.com"
```

**Systemd Service Reload**:
```bash
# Provider automatically reloads systemd after creating proxy config
systemctl daemon-reload
systemctl restart containerd
```

---

### Pattern 2: Agent Mode with Spectro Containerd

**Use Case**: Palette-managed clusters using spectro-containerd.

**Cloud-init Configuration**:
```yaml
#cloud-config
cluster:
  role: init
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    STYLUS_ROOT: "/persistent/spectro"   # Agent mode
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated Files**:
1. **`${STYLUS_ROOT}/etc/default/kubelet`** = `/persistent/spectro/etc/default/kubelet`
2. **`/run/systemd/system/spectro-containerd.service.d/http-proxy.conf`** (spectro-containerd)

**File Contents Same as Pattern 1**

**containerd Socket**:
```yaml
# In kubeadm config, criSocket points to spectro-containerd
nodeRegistration:
  criSocket: unix:///run/spectro-containerd/spectro-containerd.sock
```

---

### Pattern 3: NO_PROXY with Custom Node CIDR

**Use Case**: Include node management network in NO_PROXY.

**Note**: Unlike provider-k3s/rke2, provider-kubeadm does NOT auto-detect node CIDR. You must manually add it to NO_PROXY.

**Cloud-init Configuration**:
```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1,10.0.0.0/8,.corp.com"
    #         ^^^ Node network 10.0.0.0/8 manually added
  config: |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Generated NO_PROXY**:
```bash
NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1,10.0.0.0/8,.corp.com
```

---

### Pattern 4: Proxy with Authentication

**Use Case**: Corporate proxy requiring username/password.

**Cloud-init Configuration**:
```yaml
cluster:
  env:
    HTTP_PROXY: "http://user:password@proxy.corp.com:8080"
    HTTPS_PROXY: "http://user:password@proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
```

**Security Consideration**: Credentials visible in:
- `/etc/default/kubelet` (readable by root)
- `/run/systemd/system/containerd.service.d/http-proxy.conf`
- Process environment (visible in `ps` output)

**Best Practice**: Use proxy without authentication if possible, or IP-based authentication.

---

### Pattern 5: Different HTTP and HTTPS Proxies

**Use Case**: Separate proxies for HTTP and HTTPS traffic.

**Cloud-init Configuration**:
```yaml
cluster:
  env:
    HTTP_PROXY: "http://http-proxy.corp.com:8080"
    HTTPS_PROXY: "https://https-proxy.corp.com:8443"
    NO_PROXY: "localhost,127.0.0.1"
```

**Generated Configuration** (applies both):
```bash
HTTP_PROXY=http://http-proxy.corp.com:8080
HTTPS_PROXY=https://https-proxy.corp.com:8443
NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1
```

---

## Common Pitfalls

### ❌ WRONG: Missing pod/service subnets in NO_PROXY

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
    # ❌ Missing pod and service CIDRs
  config: |
    networking:
      podSubnet: 10.244.0.0/16
      serviceSubnet: 10.96.0.0/12
```

**Result**: Pods cannot communicate, services unreachable
```bash
kubectl get pods -o wide
# NAME     IP            NODE      STATUS
# pod-1    10.244.0.5    node-1    Running

kubectl exec pod-1 -- curl 10.96.0.1
# ❌ Hangs or times out - trying to reach service via proxy
```

### ✅ CORRECT: Provider auto-appends subnets

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
    # ✅ Provider appends pod/service subnets automatically
```

**Generated NO_PROXY**:
```bash
NO_PROXY=10.244.0.0/16,10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1
```

---

### ❌ WRONG: Forgetting node network in NO_PROXY

```yaml
# Nodes on 10.0.1.0/24 network
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
    # ❌ Missing 10.0.1.0/24 - node-to-node traffic may go via proxy
```

**Result**: Kubelet API calls between nodes may fail or be slow
```bash
# From node-2 trying to reach node-1:6443
curl https://10.0.1.100:6443
# ❌ Goes via proxy (slow or blocked)
```

### ✅ CORRECT: Include node network

```yaml
cluster:
  env:
    NO_PROXY: "localhost,127.0.0.1,10.0.1.0/24"
    #                              ^^^ Node network
```

**Note**: Unlike provider-k3s/rke2, provider-kubeadm does NOT auto-detect node network. You MUST add it manually.

---

### ❌ WRONG: Using lowercase proxy variables

```yaml
cluster:
  env:
    http_proxy: "http://proxy.corp.com:8080"   # ❌ Lowercase
    https_proxy: "http://proxy.corp.com:8080"  # ❌ Lowercase
```

**Result**: Some tools respect lowercase, some require uppercase - inconsistent behavior.

### ✅ CORRECT: Use UPPERCASE variables

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"   # ✅ Uppercase
    HTTPS_PROXY: "http://proxy.corp.com:8080"  # ✅ Uppercase
```

**Best Practice**: Provider converts to both upper and lowercase in generated files:
```bash
# /etc/default/kubelet contains both
HTTP_PROXY=http://proxy.corp.com:8080
http_proxy=http://proxy.corp.com:8080
```

---

### ❌ WRONG: Proxy blocks container registry

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1"
    # ❌ Missing registry.k8s.io - image pulls will fail
```

**Result**: kubeadm init fails pulling images
```bash
kubeadm init
# [ERROR ImagePull]: failed to pull image registry.k8s.io/kube-apiserver:v1.28.0
# ❌ Proxy blocks registry access
```

### ✅ CORRECT: Add registry to NO_PROXY

```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.corp.com:8080"
    NO_PROXY: "localhost,127.0.0.1,registry.k8s.io,gcr.io,quay.io"
    #                              ^^^ Kubernetes registries
```

**OR**: Pre-import images for air-gap deployment (see `01-architecture.md`)

---

## Proxy Configuration Logic

### Function: `kubeletProxyEnv()`

**Source**: `stages/proxy.go:35-60`

```go
func kubeletProxyEnv(clusterCtx *domain.ClusterContext) string {
    var proxy []string
    proxyMap := clusterCtx.EnvConfig

    httpProxy := proxyMap["HTTP_PROXY"]
    httpsProxy := proxyMap["HTTPS_PROXY"]
    userNoProxy := proxyMap["NO_PROXY"]

    if utils.IsProxyConfigured(proxyMap) {
        noProxy := utils.GetDefaultNoProxy(clusterCtx)

        if len(httpProxy) > 0 {
            proxy = append(proxy, fmt.Sprintf("HTTP_PROXY=%s", httpProxy))
        }
        if len(httpsProxy) > 0 {
            proxy = append(proxy, fmt.Sprintf("HTTPS_PROXY=%s", httpsProxy))
        }
        if len(userNoProxy) > 0 {
            noProxy = noProxy + "," + userNoProxy
        }
        proxy = append(proxy, fmt.Sprintf("NO_PROXY=%s", noProxy))
    }
    return strings.Join(proxy, "\n")
}
```

### Function: `GetDefaultNoProxy()`

**Source**: `utils/proxy.go:24-38`

```go
func GetDefaultNoProxy(clusterCtx *domain.ClusterContext) string {
    var noProxy string

    clusterCidr := clusterCtx.ClusterCidr    // From podSubnet
    serviceCidr := clusterCtx.ServiceCidr    // From serviceSubnet

    if len(clusterCidr) > 0 {
        noProxy = clusterCidr
    }
    if len(serviceCidr) > 0 {
        noProxy = noProxy + "," + serviceCidr
    }
    return noProxy + "," + k8sNoProxy
}
```

**Logic**:
1. Extract `podSubnet` from kubeadm ClusterConfiguration
2. Extract `serviceSubnet` from kubeadm ClusterConfiguration
3. Append k8sNoProxy constant (`.svc,.svc.cluster,.svc.cluster.local`)
4. Merge with user-provided NO_PROXY

---

## Shell Script Proxy Handling

### kube-init.sh Proxy Logic

**Source**: `scripts/kube-init.sh:61-80`

```bash
if [ "$PROXY_CONFIGURED" = true ]; then
  until HTTP_PROXY=$proxy_http http_proxy=$proxy_http \
        HTTPS_PROXY=$proxy_https https_proxy=$proxy_https \
        NO_PROXY=$proxy_no no_proxy=$proxy_no \
        kubeadm init --config "$root_path"/opt/kubeadm/kubeadm.yaml \
                     --upload-certs \
                     --ignore-preflight-errors=NumCPU \
                     --ignore-preflight-errors=Mem \
                     -v=5 > /dev/null
  do
    echo "failed to apply kubeadm init, retrying in 10s"
    do_kubeadm_reset
    sleep 10;
  done;
else
  # Non-proxy branch: kubeadm init without proxy env vars
  until kubeadm init --config "$root_path"/opt/kubeadm/kubeadm.yaml ...
  do
    # retry logic
  done;
fi
```

**Behavior**:
- If `PROXY_CONFIGURED=true`: Set proxy env vars for kubeadm command
- If `PROXY_CONFIGURED=false`: Run kubeadm without proxy env vars
- Retry logic with `kubeadm reset` on failure

---

## Integration Points

### With kubeadm Configuration

- Provider extracts `networking.podSubnet` for NO_PROXY
- Provider extracts `networking.serviceSubnet` for NO_PROXY
- Proxy settings apply to `kubeadm init` and `kubeadm join` commands

### With CNI

- CNI pods inherit kubelet proxy settings
- CNI image pulls use containerd proxy config
- CNI pod-to-pod traffic bypasses proxy (in NO_PROXY)

### With Stylus

- **Appliance mode**: `/etc/default/kubelet` and `/run/systemd/system/containerd.service.d/`
- **Agent mode**: `${STYLUS_ROOT}/etc/default/kubelet` and `/run/systemd/system/spectro-containerd.service.d/`

### With Container Runtime

- Proxy applies to:
  - **kubelet**: API calls to control plane
  - **containerd**: Image pulls from registries
  - **kube-proxy**: (No proxy needed - in-cluster traffic)
  - **CNI pods**: Image pulls and external API calls

---

## Troubleshooting

### Issue: Image pull fails with proxy

**Symptom**:
```bash
kubectl get pods -n kube-system
# NAME                          READY   STATUS             RESTARTS   AGE
# coredns-5d78c9869d-abc123     0/1     ImagePullBackOff   0          2m
```

**Diagnosis**:
```bash
kubectl describe pod coredns-5d78c9869d-abc123 -n kube-system
# Events:
#   Failed to pull image "registry.k8s.io/coredns/coredns:v1.10.1": ProxyConnectionRefused
```

**Solution**: Add registry to NO_PROXY or configure proxy to allow registry access

---

### Issue: Pods cannot reach services

**Symptom**:
```bash
kubectl exec test-pod -- curl kubernetes.default.svc
# curl: (28) Connection timed out after 30001 milliseconds
```

**Diagnosis**:
```bash
# Check NO_PROXY includes service CIDR
cat /etc/default/kubelet | grep NO_PROXY
# NO_PROXY=localhost,127.0.0.1
# ❌ Missing service CIDR!
```

**Solution**: Provider should auto-append, but if missing, manually add:
```bash
echo 'NO_PROXY=10.96.0.0/12,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1' >> /etc/default/kubelet
systemctl restart kubelet
```

---

## Reference Examples

**Provider Implementation**:
- `/Users/rishi/work/src/provider-kubeadm/stages/proxy.go` - Proxy configuration logic
- `/Users/rishi/work/src/provider-kubeadm/utils/proxy.go` - NO_PROXY calculation
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-init.sh` - Proxy env vars in init script

## Related Skills

- See `provider-kubeadm:01-architecture` for proxy overview
- See `provider-kubeadm:03-configuration-patterns` for podSubnet/serviceSubnet config
- See `provider-kubeadm:07-networking` for CNI and proxy interaction
- See `provider-kubeadm:08-troubleshooting` for proxy-related issues

**Related Provider Skills**:
- See `provider-k3s:06-proxy-configuration` for k3s proxy (same logic, auto node CIDR detection)
- See `provider-rke2:06-proxy-configuration` for rke2 proxy (same logic, auto node CIDR detection)

## Documentation References

**Kubernetes Proxy Documentation**:
- https://kubernetes.io/docs/tasks/administer-cluster/configure-kubernetes-proxy/
- https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/control-plane-flags/
