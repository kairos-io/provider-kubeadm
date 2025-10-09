# Provider-Kubeadm Knowledge Base

## Reference Files

This knowledge base is supported by additional machine-readable reference files:

- **Machine-readable index**: [ai-kb-index/provider-kubeadm-kb-index.json](ai-kb-index/provider-kubeadm-kb-index.json)
  - Structured patterns for automated PR review and troubleshooting
  - Security patterns, error patterns, runtime patterns with grep hints
  - Ready-to-run diagnostic commands for AI agents

- **Code pattern database**: [ai-kb-index/provider-kubeadm-repo-findings.csv](ai-kb-index/provider-kubeadm-repo-findings.csv)  
  - 85+ critical code patterns with exact line references
  - Security vulnerabilities, operational patterns, configuration issues
  - Evidence base for automated code review and issue detection

## 1) What is provider-kubeadm?

Provider-kubeadm is a Kairos cluster plugin that orchestrates Palette Extended Kubernetes Edge (PXKE) cluster initialization, joining, and lifecycle management using kubeadm on Palette Edge hosts. It acts as the bridge between Palette Edge cluster operations and the underlying kubeadm toolchain, handling both agent-mode (software agents on existing hardware with STYLUS_ROOT custom paths) and appliance-mode (pre-configured hardware appliances) deployments. The provider receives merged cloud-config as input from kairos-agent/palette-agent and generates YIP stages to provision Kubernetes clusters. It integrates with container runtime (containerd) using custom socket detection and manages cluster state transitions through structured YIP (Yet another Init Program) stages.

**Architecture flow:** `Palette Edge → kairos-agent/palette-agent → provider-kubeadm → YIP stages → kubeadm phases → containerd/kubelet → K8s cluster`

## 2) Development Guide

### Repo Setup & Build
• Clone from kairos-io/kairos/provider-kubeadm
• Build: `go build -o agent-provider-kubeadm main.go`
• Test: `go test ./...` (check for test files first)
• Container: `earthly +build-provider` or use Earthfile targets

### Agent Mode: Custom Path & Containerd
• STYLUS_ROOT handling: `scripts/kube-reset.sh:10-56` (PATH extension, cleanup paths for agent mode)
• Agent mode requires custom paths: `/oem` becomes `${STYLUS_ROOT}/oem`, binaries in `${STYLUS_ROOT}/usr/bin`
• Containerd socket detection: `scripts/{kube-init,kube-join,kube-reset,import}.sh` check `/run/spectro/containerd/containerd.sock` vs standard `/run/containerd/containerd.sock`
• Service folder selection: `main.go:182-187` (spectro-containerd vs containerd based on provider options)
• Configuration merging order: `/system/oem` → `/oem` → `/usr/local/cloud-config`

### Local Test Hooks
• Validate shell scripts: `shellcheck scripts/*.sh`  
• Format check: `shfmt -d scripts/*.sh`
• Unit tests: `go test -v ./...`
• Build verification: `go build -o /tmp/test-binary main.go`

## 3) PR Guide

### Regression Safety
• Ensure custom STYLUS_ROOT and containerd socket code paths are covered
• Files: `scripts/kube-reset.sh:10-56`, `scripts/{kube-init,kube-join,import}.sh:31-35`, `main.go:182-187`
• Test both agent-mode (STYLUS_ROOT set) and appliance-mode (standard paths)
• Verify YIP stage generation: `/usr/local/cloud-config/cluster.kairos.yaml` created correctly
• Check provider discovery: binaries ending with `-provider` in `/system/providers` and `/usr/local/system/providers`

### Shell Scripts
• No typos; commands exist; args correct
• Run: `shellcheck scripts/*.sh && shfmt -d scripts/*.sh`
• Verify commands: `command -v kubeadm ctr kubelet` 
• Test help flags: `kubeadm --help`, `ctr --help`, `kubelet --help`

### Docs & Commands
• Validate documented commands exist: `command -v earthly go`
• Dry-run build: `go build -o /dev/null main.go`
• Check script args: `bash -n scripts/*.sh`

## Developer Code Review Checklist

### Visual Code Inspection (No Tools Required)

**Shell Script Changes:**
• Check for typos in command names (kubeadm, kubelet, ctr, crictl)
• Verify all variables are defined before use (check $root_path, $STYLUS_ROOT)
• Look for missing quotes around file paths with spaces
• Ensure consistent error handling patterns across similar scripts

**Go Code Changes:**
• Verify all imports are actually used in the code
• Check that error returns from exec.Command() are handled
• Look for hardcoded paths that should use STYLUS_ROOT
• Ensure JSON/YAML unmarshaling has proper error handling

**Path Handling Consistency:**
• Any new file path should check existing patterns in same file
• Agent-mode paths: Must include $STYLUS_ROOT prefix
• Standard paths: Should work without STYLUS_ROOT
• Service folders: Check main.go:182-187 for containerd vs spectro-containerd logic

### Code Review Red Flags (Human-Detectable)

**Immediate Rejection Patterns:**
• New hardcoded /usr/bin or /etc/kubernetes paths without STYLUS_ROOT handling
• Service restart commands without checking service existence first
• Registry URL changes without updating both init.go and join.go consistently
• New rm -rf commands without proper safety checks
• exec.Command() calls without error handling

**Requires Extra Scrutiny:**
• Changes to scripts/ directory - must preserve dual-mode compatibility
• Version comparison logic - affects v1beta3 vs v1beta4 config generation
• Socket detection logic - must handle both spectro and standard containerd
• Certificate or token handling - check crypto operations are consistent

### Peer Review Questions to Ask

**For Shell Script Changes:**
• "Did you test this path exists in both agent-mode and appliance-mode?"
• "Are all the commands in this script actually available on target systems?"
• "Does this script handle the case where STYLUS_ROOT is unset?"

**For Go Code Changes:**
• "What happens if this exec.Command fails?"
• "Did you check if this path works with custom root directories?"
• "Is this change consistent with similar patterns elsewhere in the codebase?"

**For Configuration Changes:**
• "Does this preserve backward compatibility with existing clusters?"
• "Did you verify the YAML syntax is valid?"
• "Are registry URLs accessible from target environments?"

### Context-Aware Review Guidelines

**When Reviewing Changes to:**

**scripts/kube-*.sh files:**
- Check line 9-10: PATH extension must include both /usr/bin and /usr/local/bin
- Check socket detection: Must handle both /run/spectro/containerd and /run/containerd
- Check service operations: Must verify service exists before stop/start/restart

**main.go version logic (lines 83-92):**
- Ensure both v1beta3 and v1beta4 code paths are updated consistently
- Version comparisons must handle edge cases (missing kubeadm, etc.)

**stages/*.go files:**
- Registry changes must be applied to all relevant files
- File path generation must use filepath.Join() not string concatenation
- Commands must include proper proxy environment variable handling

### What Previous Reviews Missed

**Recent commit patterns that human reviewers should have caught:**

• **Service restart without validation** - Adding systemctl commands without checking service exists
• **Registry URL inconsistency** - Changing one file but not updating related files
• **Path handling gaps** - New file operations not following existing STYLUS_ROOT patterns
• **Binary cleanup oversight** - Adding destructive operations without considering all deployment modes

**Human-detectable signs:**
- Code that looks "different" from surrounding patterns
- Missing error handling where other similar code has it
- Inconsistent variable naming or path construction
- Changes that affect one deployment mode but ignore the other

## 4) Troubleshooting Guide

### 4.1 Live Edge Host

#### Mode Detection & Path Verification

**Appliance Mode (Standard Paths):**
```bash
# Verify standard paths and no STYLUS_ROOT
echo $STYLUS_ROOT  # Should be empty
which kubeadm kubelet  # Should be in /usr/bin or /usr/local/bin
ls -la /system/providers/agent-provider-kubeadm
```

**Agent Mode (Custom STYLUS_ROOT Paths):**
```bash
# Verify agent-mode environment
echo $STYLUS_ROOT  # Should point to custom root (e.g., /persistent/spectro)
echo $PATH | grep $STYLUS_ROOT  # Should include $STYLUS_ROOT/usr/bin
ls -la $STYLUS_ROOT/usr/bin/kubeadm $STYLUS_ROOT/usr/local/bin/kubelet
ls -la $STYLUS_ROOT/system/providers/agent-provider-kubeadm
```

#### Configuration Flow Debugging

**YIP Stage Execution Services:**
```bash
systemctl status cos-setup-boot.service     # Provider execution stage
systemctl status cos-setup-fs.service cos-setup-network.service
ps aux | grep -E "(kairos-agent|palette-agent|stylus-agent)"
```

**Configuration Merging:**
```bash
cat /usr/local/cloud-config/cluster.kairos.yaml  # Generated YIP stages
ls -la /system/oem/ /oem/ /usr/local/cloud-config/  # Merge order
```

#### Pre-kube-init / kube-init / Post-kube-init failures

**Pre-init Phase:**
```bash
# Service status (mode-aware)
systemctl status kubelet containerd spectro-containerd
journalctl -u kubelet --since "1 hour ago"

# Socket detection (critical for both modes)
ls -la /run/spectro/containerd/containerd.sock /run/containerd/containerd.sock
crictl info

# Path verification (agent mode)
[[ -n "$STYLUS_ROOT" ]] && ls -la $STYLUS_ROOT/usr/bin/ || which kubeadm kubelet

# Virtual interface (cluster stability)
ip addr show scbr-100  # K8s binds here vs physical interfaces
```

**Init Phase:**
```bash
# Core logs and config
journalctl -u kubelet --since "30 minutes ago" | grep -E "(error|failed)"
ls -la /etc/kubernetes/manifests/
kubeadm config print-default

# Certificates and static pods  
ls -la /etc/kubernetes/pki/
kubeadm certs check-expiration
```

**Post-init Phase:**
```bash
# Node and pod status
kubectl get nodes -o wide
kubectl get pods -A --field-selector spec.nodeName=$(hostname)

# CNI verification
ls -la /opt/cni/bin/
kubectl get pods -n kube-system | grep -E "(cni|network)"
```

#### K8s Upgrade Failures
```bash
# Version comparison (critical mismatch detection)
kubeadm version && kubelet --version
kubectl get pods -n kube-system -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[0].image}{"\n"}{end}'

# Upgrade status and service restart
kubeadm upgrade plan
systemctl status kubelet
```

### 4.2 Support Bundle

Support bundles are compressed archives containing comprehensive diagnostic data extracted from edge hosts.

#### Bundle Structure Overview

**Core Directories:**
• `/k8s/` - All Kubernetes-specific data
  - `cluster-resources/` - K8s resources organized by type and namespace
  - `cluster-info/dump/` - Detailed dumps with pod logs by namespace
  - `metrics/` - Node, pod, and container resource metrics  
  - `pod-logs/` - Current and previous pod logs by namespace/pod/container
• `/journald/` - SystemD service logs (spectro-palette-agent-*, spectro-stylus-*, k3s, rke2-*)
• `/var/log/` - Application and system logs (palette-agent.log, stylus-*.log, provider-*.log)
• `/networking/` - Network diagnostics (IP config, firewall rules, socket stats, CNI config)
• `/oem/` - Edge-specific configurations (stylus, network, cluster configs)
• `/usr/local/cloud-config/` - Generated YIP stages and local cloud configurations
• `/opt/spectrocloud/` - Binary checksums and validation data
• `/systeminfo/` - Basic system information (hostname, OS, DNS)

#### Automated Bundle Triage

**Key File Paths to Search:**
• `journald/stylus-agent.log`, `journald/spectro-stylus-agent.log` - Agent execution logs
• `journald/k3s.log`, `journald/spectro-palette-agent-*.log` - Distribution-specific logs
• `var/log/provider-kubeadm.log` - Provider-specific execution logs
• `k8s/cluster-resources/custom-resources/spectrosystemtasks.cluster.spectrocloud.com/` - Upgrade task status
• `var/log/stylus-upgrade-script-*.log` - Upgrade execution logs
• `usr/local/cloud-config/cluster.kairos.yaml` - Generated provider YIP stages

**Key Grep Patterns:**
```bash
# Provider and YIP stage execution failures
grep -E "(failed|error|fatal)" journald/spectro-stylus-agent.log
grep -E "provider.*kubeadm" journald/stylus-agent.log
grep -E "(boot.before|initramfs)" usr/local/cloud-config/cluster.kairos.yaml

# Kubeadm phase failures  
grep -E "(kubeadm.*failed|kubeadm.*error)" var/log/provider-kubeadm.log
grep -E "failed.*cluster.*init" journald/k3s.log

# CRI socket issues specific to provider-kubeadm
grep -E "(containerd.*sock|cri-socket)" var/log/provider-kubeadm.log
grep "socket.*not.*found" journald/stylus-agent.log
grep "/run/spectro/containerd/containerd.sock" var/log/ -R

# STYLUS_ROOT path issues  
grep "STYLUS_ROOT" journald/stylus-agent.log
grep "command.*not.*found.*usr/bin" var/log/ -R

# Upgrade failures
grep -E "(upgrade.*failed|SpectroSystemTask)" var/log/stylus-upgrade-script-*.log
grep -E "control-plane|worker-plane" k8s/cluster-resources/custom-resources/spectrosystemtasks.cluster.spectrocloud.com/ -R

# Certificate/API connectivity (kubelet port 10250)
grep -E "proxy error.*10250" var/log/stylus-agent.log
grep "connection refused.*6443" journald/stylus-agent.log
```

**Version Mismatch Detection:**
```bash
# Extract component versions
grep "Version:" cluster-info/nodes.yaml
jq -r '.status.nodeInfo.kubeletVersion' cluster-info/nodes.yaml  
grep "image:" kubernetes/manifests/kube-*.yaml | sort | uniq

# Containerd socket issues
grep -E "(containerd\.sock|cri-socket)" logs/ -R
grep "STYLUS_ROOT" logs/ -R
```

#### Decision Table

| Category | Symptom | Likely Cause | How to Confirm | Fix / Next Step |
|----------|---------|--------------|----------------|-----------------|
| Provider Discovery | Provider not found/executed | Missing binary or wrong path | Check `/system/providers/agent-provider-kubeadm` exists | Verify provider binary placement and permissions |
| YIP Stage Generation | No cluster.kairos.yaml created | Provider execution failed | Check `journald/spectro-stylus-agent.log` for provider errors | Fix provider input/config, restart agent |
| CRI Socket | "socket not found" in kubelet logs | Wrong containerd socket path | Check `/run/spectro/containerd/containerd.sock` vs `/run/containerd/containerd.sock` | Verify socket detection logic in scripts |
| STYLUS_ROOT | Binary not found errors | Missing agent-mode paths | Check `$STYLUS_ROOT` env var, verify `$STYLUS_ROOT/usr/bin` in PATH | Set STYLUS_ROOT env, fix path extension |
| Boot Stage Failed | cos-setup-boot.service failed | YIP stage execution error | Check `systemctl status cos-setup-boot.service` and journalctl | Examine usr/local/cloud-config/ content |
| Kubelet Version Mismatch | Old kubelet version post-upgrade | Service restart failed | Compare `kubelet --version` vs cluster version | Restart K8s service (k3s, rke2, etc.) |
| Upgrade Timeout | SpectroSystemTask stuck | Pod scheduling/image pull issues | Check upgrade pod logs in `var/log/stylus-upgrade-script-*.log` | Resolve image availability, node resources |
| API Server Unreachable | 502 Bad Gateway on port 10250 | Kubelet not restarted after upgrade | Check `proxy error from 127.0.0.1:6443 while dialing X.X.X.X:10250` | Restart kubelet service |
| Configuration Merge | Invalid cluster config | Wrong merge order or syntax | Verify `/system/oem`, `/oem`, `/usr/local/cloud-config` order | Fix configuration files, restart agent |
| Virtual Interface | Cluster connectivity issues | scbr-100 interface missing/misconfigured | Check `ip addr show scbr-100` interface status | Verify virtual interface configuration in YIP stages |

#### Ready-to-run Pipelines

**Extract Errors by Timeline:**
```bash
# Correlate timestamps across logs
for log in logs/*.log; do echo "=== $log ==="; grep -E "$(date -d '1 hour ago' '+%Y-%m-%d %H')" "$log" | grep -E "(error|failed|fatal)"; done
```

**Detect Component Version Mismatches:**
```bash
# Compare kubelet vs control plane versions
kubectl_ver=$(grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+" cluster-info/nodes.yaml | head -1)
api_ver=$(grep "image:" kubernetes/manifests/kube-apiserver.yaml | grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+")
echo "Kubelet: $kubectl_ver, API Server: $api_ver"
```

**Surface Containerd Socket Errors:**
```bash
# Find socket path mismatches
grep -r "cri-socket" logs/ | grep -E "(unix://|/run/)"
grep -r "containerd.*sock" systemd/ 
```

## PR Gate Failures

**Shell Script Quality:**
• All scripts in `scripts/` directory pass syntax validation (`bash -n`)
• Commands referenced: `kubeadm`, `kubelet`, `ctr`, `crictl` - verify availability in target environment
• Socket detection logic verified for both spectro and standard containerd paths

**Command Verification:**
• Earthly build commands: `earthly +build-provider`, `earthly +build-provider-package`  
• Go build: `go build -o agent-provider-kubeadm main.go`
• Container runtime: `ctr`, `crictl` with appropriate socket paths

## Edge Cluster Debugging Context

### Palette Edge Architecture Overview

Palette Edge supports two deployment modes:
• **Agent Mode**: Software agents on existing hardware with custom STYLUS_ROOT paths
• **Appliance Mode**: Pre-configured hardware appliances with standard paths

### Key Debugging Concepts from Support Bundle Example

**Kubelet Version Mismatch After Upgrade:**
This is a common issue where K8s binary is upgraded but kubelet version remains old. Root cause analysis:

1. **Upgrade Process**: K8s upgrades create SpectroSystemTask custom resource
2. **Version Check Commands**: 
   - `k3s --version` shows v1.31.5+k3s1 (correct)  
   - `kubectl get nodes` shows VERSION v1.30.9+k3s1 (incorrect)
3. **Fix**: Restart the K8s service (k3s, rke2, kubeadm) to reload kubelet
4. **Support Bundle Locations**:
   - Upgrade logs: `var/log/stylus-upgrade-script-*.log`
   - Task status: `k8s/cluster-resources/custom-resources/spectrosystemtasks.cluster.spectrocloud.com/`
   - Service restart logs: `journalctl -u k3s.service | grep "Starting Lightweight Kubernetes"`

**502 Bad Gateway Errors on Port 10250:**
Indicates kubelet API is unreachable, often after failed service restart during upgrades.
