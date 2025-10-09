# kb_index.json - Machine-Readable Knowledge Index

Purpose: Structured lookup table for AI agents to quickly find relevant troubleshooting patterns and commands without parsing the full markdown.

Usage in PR Review:
```
  {
    "troubleshooting_patterns": {
      "cri_socket_errors": {
        "patterns": ["socket.*not.*found", "/run/spectro/containerd/containerd.sock"],
        "log_files": ["var/log/provider-kubeadm.log"],
        "category": "CRI Socket"
      }
    },
    "key_commands": {
      "support_bundle_analysis": [
        "grep -E '(failed|error|fatal)' journald/spectro-stylus-agent.log"
      ]
    }
  }
```

PR Review Usage:
- Pattern Matching: When a PR touches containerd socket code, AI agents can instantly find related error patterns
- Command Generation: Auto-generate testing commands for specific areas of code changes
- Context Reduction: Avoid loading the entire markdown knowledge base

repo_findings.csv - Critical Code Pattern Database

Purpose: Tabular evidence of security-sensitive and regression-prone code locations for automated scanning.

Usage in PR Review:
```
topic,file,line_start,line_end,code_context_excerpt
COMMAND_INJECTION_EXEC,main.go,68,69,"cmd := exec.Command(""/bin/sh"", ""-c"", filepath.Join(...))"
SHELL_INJECTION_SUDO,scripts/kube-upgrade.sh,148,148,"if sudo -E bash -c ""$upgrade_command"""
```

PR Review Usage:
- Automated Security Scanning: Flag changes near command injection points
- Regression Detection: Alert when modifications touch critical STYLUS_ROOT or socket detection logic
- Risk Assessment: Assign higher review priority to PRs touching high-risk patterns
