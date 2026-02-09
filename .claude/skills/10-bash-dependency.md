# kubeadm Bash Dependency

## Overview

**CRITICAL**: Provider-kubeadm is **NOT POSIX compliant**. All 9 shell scripts use `#!/bin/bash` and bash-specific features. This differs from provider-k3s and provider-rke2 which are POSIX-compliant and work with minimal shells like `dash` or `ash`.

## Key Concepts

### POSIX vs Bash

**POSIX Shell** (`/bin/sh`):
- Minimal, standardized shell
- Available on all Unix-like systems
- Common implementations: `dash`, `ash`, `busybox sh`
- No advanced features (arrays, associative arrays, process substitution)

**Bash** (`/bin/bash`):
- Enhanced shell with many features
- Not always available on minimal systems
- Provides: arrays, process substitution, extended globs, etc.
- Larger binary size (~1MB vs 100KB for dash)

### Why Provider-kubeadm Requires Bash

Provider-kubeadm scripts use bash-specific features:

1. **Process Substitution**: `>(command)`
2. **File Descriptor Assignment**: `exec 19>>`
3. **BASH_XTRACEFD**: Bash 4.1+ variable for debug output
4. **Extended Logging**: Sophisticated log redirection

### Comparison with Other Providers

| Provider | Shell Requirement | POSIX Compliant | Impact |
|----------|-------------------|-----------------|--------|
| **provider-k3s** | `#!/bin/sh` | ✅ Yes | Works on minimal systems |
| **provider-rke2** | `#!/bin/sh` | ✅ Yes | Works on minimal systems |
| **provider-kubeadm** | `#!/bin/bash` | ❌ No | Requires bash installation |

## Implementation Patterns

### Pattern 1: Bash-Specific Features in Scripts

**Use Case**: Understanding why bash is required.

**kube-init.sh** (lines 1-7):
```bash
#!/bin/bash

exec   > >(tee -ia /var/log/kube-init.log)
exec  2> >(tee -ia /var/log/kube-init.log >& 2)
exec 19>> /var/log/kube-init.log

export BASH_XTRACEFD="19"
set -ex
```

**Bash Features Used**:

1. **Process Substitution** `>(tee -ia ...)`
   - Bash 2.0+ feature
   - Creates a named pipe for command output
   - **Not available in POSIX sh**

2. **File Descriptor Assignment** `exec 19>>`
   - Opens file descriptor 19 for appending
   - **Not standard in POSIX sh**

3. **BASH_XTRACEFD** Variable
   - Bash 4.1+ feature
   - Redirects `set -x` (debug) output to specific file descriptor
   - **Not available in POSIX sh**

**Why These Features**:
- Advanced logging: stdout, stderr, and debug output all go to same log file
- Prevents log interleaving
- Debug output (`set -x`) separate from regular output

---

### Pattern 2: POSIX Alternative (Not Implemented)

**Use Case**: How scripts could be rewritten for POSIX compliance (future work).

**Current Bash Version**:
```bash
#!/bin/bash

exec   > >(tee -ia /var/log/kube-init.log)
exec  2> >(tee -ia /var/log/kube-init.log >& 2)
exec 19>> /var/log/kube-init.log

export BASH_XTRACEFD="19"
set -ex
```

**POSIX Alternative** (simplified logging):
```sh
#!/bin/sh

# Redirect stdout and stderr to log file
exec >> /var/log/kube-init.log 2>&1

# Enable debug output (goes to same log)
set -ex
```

**Trade-offs**:
- ✅ POSIX compliant - works on any shell
- ❌ Less sophisticated logging
- ❌ Debug output mixed with regular output
- ❌ Cannot separate stdout/stderr/debug

**Current Decision**: Provider-kubeadm prioritizes advanced logging over POSIX compliance.

---

### Pattern 3: Bash Availability Check

**Use Case**: Detect if bash is available on edge host.

**Pre-flight Check** (example script):
```bash
#!/bin/sh
# This part runs with POSIX sh to check for bash

if ! command -v bash >/dev/null 2>&1; then
    echo "ERROR: bash is required but not installed"
    echo "Install bash and retry:"
    echo "  Alpine: apk add bash"
    echo "  Debian: apt-get install bash"
    echo "  RHEL:   yum install bash"
    exit 1
fi

# Switch to bash for actual execution
exec bash "$0" "$@"
```

**Note**: Provider-kubeadm does NOT currently include this check. Host must have bash pre-installed.

---

### Pattern 4: Minimal System Workaround

**Use Case**: Installing bash on minimal edge operating systems.

**Alpine Linux** (common for edge):
```bash
# Check if bash installed
which bash
# /bin/bash not found

# Install bash
apk add bash

# Verify
bash --version
# GNU bash, version 5.2.15
```

**Debian/Ubuntu**:
```bash
apt-get update
apt-get install -y bash
```

**RHEL/Rocky/CentOS**:
```bash
yum install -y bash
```

**Kairos/C3OS**:
- Bash typically pre-installed in Kairos images
- If using custom minimal images, add bash to image build

---

### Pattern 5: Script Execution Flow

**Use Case**: How provider-kubeadm executes bash scripts.

**Provider Execution**:
```go
// pkg/provider/provider.go
files := []yip.File{
    {
        Path:        "/opt/kubeadm/kube-init.sh",
        Permissions: 0755,
        Content:     string(initScript),    // Bash script with #!/bin/bash
    },
}

// Yip executes script
// If bash not found: /bin/bash: bad interpreter: No such file or directory
```

**Script Execution** (by Yip):
```bash
# Yip runs:
/opt/kubeadm/kube-init.sh

# Which internally resolves to:
/bin/bash /opt/kubeadm/kube-init.sh

# If bash not installed → Error
```

---

## All Scripts Using Bash

**Complete List** (9 scripts):

1. **`import.sh`** - Image import script
   ```bash
   #!/bin/bash
   ```

2. **`kube-images-load.sh`** - Load container images
   ```bash
   #!/bin/bash
   ```

3. **`kube-init.sh`** - Cluster initialization
   ```bash
   #!/bin/bash
   exec   > >(tee -ia /var/log/kube-init.log)
   export BASH_XTRACEFD="19"
   ```

4. **`kube-join.sh`** - Node join
   ```bash
   #!/bin/bash
   exec   > >(tee -ia /var/log/kube-join.log)
   ```

5. **`kube-post-init.sh`** - Post-initialization
   ```bash
   #!/bin/bash
   ```

6. **`kube-pre-init.sh`** - Pre-initialization
   ```bash
   #!/bin/bash
   exec   > >(tee -ia /var/log/kube-pre-init.log)
   ```

7. **`kube-reconfigure.sh`** - Reconfiguration
   ```bash
   #!/bin/bash
   ```

8. **`kube-reset.sh`** - Cluster reset
   ```bash
   #!/bin/bash
   ```

9. **`kube-upgrade.sh`** - Cluster upgrade
   ```bash
   #!/bin/bash
   exec   > >(tee -ia /var/log/kube-upgrade.log)
   ```

**All 9 scripts require bash** - no POSIX alternatives provided.

---

## Common Pitfalls

### ❌ WRONG: Assuming POSIX shell is enough

```bash
# Minimal edge system with only /bin/sh (dash)
/opt/kubeadm/kube-init.sh
# /bin/bash: bad interpreter: No such file or directory
# ❌ Script cannot execute
```

### ✅ CORRECT: Ensure bash is installed

```bash
# Pre-install bash on edge image
apk add bash

# OR add to Dockerfile
RUN apk add --no-cache bash

# OR add to Kairos cloud-init
stages:
  boot.before:
    - commands:
      - apk add bash
```

---

### ❌ WRONG: Trying to run with POSIX sh

```bash
# Force POSIX sh
/bin/sh /opt/kubeadm/kube-init.sh
# Syntax error: "(" unexpected
# ❌ Process substitution not supported in POSIX sh
```

### ✅ CORRECT: Use bash explicitly

```bash
/bin/bash /opt/kubeadm/kube-init.sh
# ✅ Runs correctly with bash
```

---

### ❌ WRONG: Symlinking /bin/sh to dash on Alpine

```bash
# Alpine default: /bin/sh -> dash
ln -sf /bin/dash /bin/sh

# Provider-kubeadm scripts fail
/opt/kubeadm/kube-init.sh
# ❌ dash doesn't support bash features
```

### ✅ CORRECT: Install bash, leave /bin/sh as-is

```bash
# Install bash
apk add bash

# /bin/sh can remain dash - scripts explicitly use #!/bin/bash
```

---

## Comparison Table

| Feature | Provider-k3s | Provider-rke2 | Provider-kubeadm |
|---------|--------------|---------------|------------------|
| **Shebang** | `#!/bin/sh` | `#!/bin/sh` | `#!/bin/bash` |
| **Process Substitution** | ❌ Not used | ❌ Not used | ✅ Used |
| **BASH_XTRACEFD** | ❌ Not used | ❌ Not used | ✅ Used |
| **File Descriptors** | Standard (0,1,2) | Standard (0,1,2) | Extended (19) |
| **Works on Alpine (dash)** | ✅ Yes | ✅ Yes | ❌ No (needs bash) |
| **Works on Minimal Systems** | ✅ Yes | ✅ Yes | ⚠️ If bash installed |
| **Binary Size Impact** | None | None | +1MB (bash) |

---

## Why Not POSIX-Compliant?

**Rationale for Bash Requirement**:

1. **Advanced Logging**: Bash features enable sophisticated log handling
   - Separate file descriptors for debug vs output
   - Prevents log interleaving
   - Better troubleshooting

2. **Existing Implementation**: Scripts already use bash
   - Rewrite to POSIX would require significant effort
   - Risk of introducing bugs

3. **kubeadm Complexity**: kubeadm is more complex than k3s/rke2
   - More error handling needed
   - Better debugging output essential
   - Bash features help manage complexity

**Trade-off**: Accept bash dependency for better logging and debugging.

---

## Future Considerations

### Potential POSIX Migration

**Pros**:
- Works on minimal systems without bash
- Smaller footprint (no bash binary needed)
- Consistent with provider-k3s and provider-rke2

**Cons**:
- Simplified logging (lose BASH_XTRACEFD)
- More complex log handling in POSIX sh
- Significant rewrite effort
- Risk of introducing bugs

**Recommendation**: Keep bash requirement for now, document clearly.

---

## Integration Points

### With Kairos/C3OS

- Kairos images typically include bash by default
- If using custom minimal Kairos, ensure bash is installed
- Add bash to Kairos image build or cloud-init stages

### With Edge Operating Systems

**Alpine Linux**:
- Default shell: dash (POSIX)
- **Action Required**: `apk add bash`

**Ubuntu/Debian**:
- Default shell: bash
- **No Action Required**

**RHEL/Rocky/CentOS**:
- Default shell: bash
- **No Action Required**

### With Provider Container

- Provider container image could include bash if edge host doesn't have it
- Current implementation expects bash on host (not in container)

---

## Reference Examples

**Shell Scripts**:
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-init.sh` - Main script with bash features
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-join.sh` - Join script
- `/Users/rishi/work/src/provider-kubeadm/scripts/kube-reset.sh` - Reset script

**POSIX Comparison**:
- `/Users/rishi/work/src/provider-k3s/scripts/` - POSIX-compliant scripts for comparison
- `/Users/rishi/work/src/provider-rke2/scripts/` - POSIX-compliant scripts for comparison

## Related Skills

- See `provider-kubeadm:01-architecture` for bash requirement overview
- See `provider-kubeadm:08-troubleshooting` for bash-related errors

**Related Provider Skills**:
- See `provider-k3s:01-architecture` - POSIX-compliant (no bash required)
- See `provider-rke2:01-architecture` - POSIX-compliant (no bash required)

## Documentation References

**Bash Features**:
- Process Substitution: https://www.gnu.org/software/bash/manual/html_node/Process-Substitution.html
- File Descriptors: https://www.gnu.org/software/bash/manual/html_node/Redirections.html
- BASH_XTRACEFD: https://www.gnu.org/software/bash/manual/html_node/Bash-Variables.html

**POSIX Shell**:
- POSIX Spec: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html
