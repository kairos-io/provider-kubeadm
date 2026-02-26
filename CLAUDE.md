# CLAUDE.md — provider-kubeadm

## What This Repo Is

This is the kubeadm cluster provider plugin for Kairos, an immutable Linux OS for Kubernetes. It implements the `clusterplugin.ClusterPlugin` interface from `kairos-sdk` and produces `yip.YipConfig` stage graphs that orchestrate kubeadm cluster bootstrap. It supports two deployment modes: appliance-mode (rootPath `/`) and agent-mode (rootPath `/persistent/spectro`), plus two kubeadm API versions (v1beta3 for k8s < 1.31, v1beta4 for k8s >= 1.31).

---

## Provider Structure

The entry point is `main.go`. The `clusterProvider` function is the core: it takes a `clusterplugin.Cluster`, builds a `*domain.ClusterContext`, version-dispatches to `getV1Beta3FinalStage` or `getV1Beta4FinalStage`, and returns a `yip.YipConfig` with all stages keyed under `"boot.before"`.

Stage construction is split across packages:
- `stages/` — builds `yip.Stage` values for pre, init, join, proxy
- `utils/` — pure helpers: defaults mutation, cert hashing, kubelet args, proxy config, misc
- `domain/` — plain structs: `ClusterContext`, `KubeadmConfigBeta3/4`, constants
- `log/` — logger init with lumberjack rotation and version tagging

The plugin also registers a `clusterplugin.EventClusterReset` handler (`handleClusterReset`) that execs `kube-reset.sh` via `exec.Command`.

---

## ClusterContext

`domain.ClusterContext` is the single carrier struct that flows through all stage builders. It is built once in `CreateClusterContext` and mutated before stage building:
- `RootPath` — set from `cluster.ProviderOptions["cluster_root_path"]`, defaults to `"/"`
- `NodeRole` — set directly from `string(cluster.Role)` (values: `"init"`, `"controlplane"`, `"worker"`)
- `ClusterToken` — always passed through `utils.TransformToken` (SHA256-derived, `xxxxxx.xxxxxxxxxxxxxxxx` format)
- `LocalImagesPath` — defaults to `filepath.Join(RootPath, "opt/content/images")` if not set on cluster
- `ContainerdServiceFolderName` — `"spectro-containerd"` if provider option `"spectro-containerd-service-name"` is present, else `"containerd"`
- `KubeletArgs`, `CertSansRevision`, `CustomNodeIp`, `ServiceCidr`, `ClusterCidr` — set after config defaults mutation

Do not add new fields to `ClusterContext` unless they need to flow across multiple stage builder functions. Keep the struct as a plain data carrier with no methods.

---

## Kubeadm API Version Handling

Both v1beta3 and v1beta4 paths are fully parallel. Every function that touches kubeadm config objects has a `Beta3` and a `Beta4` variant. This is intentional and not to be collapsed — the APIs have real structural differences (`KubeletExtraArgs` is `map[string]string` in v1beta3 and `[]Arg` in v1beta4).

When adding behavior that applies to both versions, add it to both functions. Do not try to unify them with interface tricks or generics.

The version check calls the real `kubeadm` binary via `exec.Command` at startup. This is by design.

---

## Stage Building Pattern

Every public stage function returns `yip.Stage` or `[]yip.Stage`. The naming convention is:

- `Get<Name>Stage(...)` for exported functions (called from `main.go`)
- `get<Name>Stage(...)` for unexported functions (called within `stages/`)

Stages are built imperatively: construct the struct, conditionally fill `Commands` based on proxy state, return it. No builders, no chaining, no functional options.

The proxy branch is always `if utils.IsProxyConfigured(clusterCtx.EnvConfig)`. When true, proxy args (`HTTP_PROXY`, `HTTPS_PROXY`, and the computed no-proxy string) are appended as positional shell arguments to the bash invocation. When false, the command is shorter. Both branches are always spelled out — no ternary, no helper that picks one or the other.

Shell commands are constructed with `fmt.Sprintf` and `filepath.Join`. Script paths always use the pattern `filepath.Join(rootPath, helperScriptPath, "script-name.sh")`.

Idempotency guards use `.init` / `.join` sentinel files: `[ ! -f <rootPath>/opt/kubeadm.init ]` in the `If` field. Not all stages have an `If` guard — only the ones that must not re-run.

---

## Config File Generation

Kubeadm config files are serialized using `k8s.io/cli-runtime/pkg/printers.YAMLPrinter` via `printObj([]runtime.Object{...})`. Multiple objects are concatenated with `---` separators naturally. This is the only way config is serialized — do not use `yaml.Marshal` or `json.Marshal` for kubeadm config objects.

`utils.GetFileStage(stageName, path, content string)` is the helper for producing a `yip.Stage` that writes a single file with permissions `0640`. Use it for all config file write stages.

Proxy env files use permission `0400` and are written inline in `stages/proxy.go`, not through `GetFileStage`, because they need different permissions.

---

## Defaults Mutation

`utils.MutateClusterConfigBeta3/4Defaults` and `utils.MutateKubeletDefaults` mutate config pointers in place. They use the pattern:

```go
if cfg.Field == "" {
    cfg.Field = someDefault
}
```

Never overwrite a user-supplied value. Append-only operations use the local `appendIfNotPresent` helper. These functions are called at the top of each `GetInit/JoinYipStages*` function, before any stage is built.

---

## Error Handling

This codebase has minimal error propagation. The design is:

- `main()` calls `logrus.Fatal(err)` on plugin startup errors
- `clusterProvider` calls `logrus.Fatalf` on kubeadm version check failure — this is intentional, the plugin cannot proceed without knowing the API version
- `handleClusterReset` populates `response.Error` and returns, it does not panic
- Errors from `json.Unmarshal` of user options are **silently dropped** with `_` — this is deliberate, partial configs are acceptable
- Errors from `printObj` / `initPrintr.PrintObj` are dropped with `_` — the printer rarely fails on valid k8s objects
- The `isServiceActive` helper returns `(bool, error)`; the error is discarded at the call site in `MutateKubeletDefaults`

Do not add error returns to functions that currently have none. Do not wrap errors with `fmt.Errorf("...: %w", err)` unless the function signature already returns an error. When a function must return an error, use `fmt.Errorf("description: %v", err)` (not `%w`).

---

## Code Style

**Functions:** Short, focused, one purpose. Private helper functions are fine and preferred over long exported functions. Functions that build a stage are named for what they produce, not what they do.

**No named returns.** Every return is explicit.

**No early-return chains for success.** The happy path is the primary flow. Guard clauses are used only for nil/empty checks at the top of a function.

**Variable declarations:** Declare variables at their first use with `:=`. Use `var x Type` only when the zero value is meaningful and initialization is deferred (e.g., `var finalStages []yip.Stage`).

**String construction:** `fmt.Sprintf` for anything with more than one concatenation. `filepath.Join` for all path construction.

**Imports:** Grouped as stdlib / external / internal. Aliased imports follow the pattern in existing code: `yip "github.com/mudler/yip/pkg/schema"`, `kyaml "sigs.k8s.io/yaml"`, `kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"`, etc.

**Struct tags:** All domain struct fields carry both `json` and `yaml` tags with `omitempty`.

**Constants:** Defined at package level with `const (...)`. String constants like path segments live in the package that uses them (`configurationPath` in `stages/init.go`, `helperScriptPath` in `stages/pre.go`).

**Type names:** PascalCase for exported, camelCase for unexported. Acronyms are uppercased: `RootPath`, `ClusterCidr`, `KubeletArgs`. No stutter in type names — the package name provides context.

**No interfaces defined in this repo.** The plugin satisfies the `clusterplugin` interface implicitly by function signature. Do not introduce interfaces for things that have a single implementation.

---

## Testing Conventions

Tests use `github.com/onsi/gomega` with `NewWithT(t)`, not `RegisterTestingT`. The import is dot-imported: `. "github.com/onsi/gomega"`.

Table-driven tests use a `[]struct{ name string; ... }` literal. The `name` field is always `snake_case`. Each case is run with `t.Run(tt.name, ...)`. The `g := NewWithT(t)` call goes inside `t.Run`.

Test functions that test a single scenario (not table-driven) follow: `g := NewWithT(t)` at the top, then assertions directly.

Test files live next to the source they test (e.g., `utils/proxy_test.go` tests `utils/proxy.go`). The `tests/` directory contains integration and unit packages that test cross-package behavior or scenarios requiring more setup.

Tests do not mock filesystem state or exec calls — they test the output of the stage builders directly by inspecting the returned `yip.Stage` struct fields. Tests that require an actual `kubeadm` binary use `t.Skip(...)`.

When a function is private, it is tested through the public function that calls it (e.g., `kubeletProxyEnv` is tested via `GetPreKubeadmProxyStage`).

`validateCommands` and similar sub-validators are inline `func(*testing.T, ...)` fields in the test case struct, not separate named functions.

---

## Patterns to Avoid

- Do not add error return values to functions that currently have none. Errors from serialization and defaults mutation are intentionally swallowed.
- Do not create wrapper types or interfaces around `domain.ClusterContext`. It is a plain struct, keep it that way.
- Do not try to unify the v1beta3/v1beta4 code paths. The parallelism is load-bearing.
- Do not use `context.Context` anywhere. This plugin is not a long-running server.
- Do not use `errors.Is` / `errors.As` / `%w` wrapping. The repo uses `%v` for error formatting.
- Do not use `logrus.WithField` chains in the hot path. Use `logrus.Info`, `logrus.Infof`, `logrus.Error`, `logrus.Errorf` directly.
- Do not put business logic in `main.go` beyond wiring. The `clusterProvider` function and `CreateClusterContext` are at the boundary of acceptable complexity in `main.go`.
- Do not generate config YAML by hand-building strings. Always use `printObj` with the typed kubeadm API structs.
- Do not add new packages. The existing `domain`, `stages`, `utils`, `log`, `version` split is deliberate and sufficient.
- Do not add struct methods to domain types. Behavior belongs in `utils` or `stages` functions that take the struct as an argument.

## Function Design & Testability

- **Every function does one thing and fits in ~20–30 lines.** If it grows beyond that, extract named helpers.
- **Write functions so they can be unit tested in isolation** — no hidden side effects, no global state access, no I/O buried inside business logic.
- **Most business logic must be unit testable** without spinning up a server, database, or Kubernetes cluster. Separate I/O at the boundary.
- **Use guard clauses / early returns** to reduce nesting. Flat code is easier to read and test than deeply nested.
- **Accept interfaces, return concrete types.** This makes callers mockable without reflection or code generation.
- **Keep interfaces small** — 1–3 methods. Large interfaces are hard to mock and signal poor separation of concerns.

## General Go Practices

- **Dependency injection over globals.** Pass dependencies via constructors or function parameters — not package-level singletons (except logging).
- **`context.Context` is always the first parameter** on any function that performs I/O. Never store it in a struct field.
- **Table-driven tests** for any function with multiple input/output cases: `[]struct{ name, input, expected }` with `t.Run`.
- **Test naming:** `TestFuncName_Scenario` — e.g. `TestCreateCluster_MissingName`.
- **Prefer `switch` over long `if/else if` chains.**
- **Short variable names in small scopes** (`i`, `v`, `err`) are idiomatic; use descriptive names in wider scopes.
- **No goroutines unless concurrency is genuinely required.** Sequential code is easier to test and reason about.
- **Avoid `init()` for anything except registering handlers or loggers.** Never use it for config loading or side-effectful setup.
- **Respect context cancellation** in any loop that calls external services.
- **Import grouping:** stdlib / external / internal — separated by blank lines, sorted by `goimports`.
- **Don't over-abstract.** Don't create an interface or wrapper until there are ≥2 concrete implementations or a clear testing need.
- **No naked `panic` in library code.** Panics are only acceptable in `main` or test setup for truly unrecoverable state.
