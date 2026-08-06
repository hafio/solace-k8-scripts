# Graph Report - .  (2026-08-04)

## Corpus Check
- 62 files · ~77,765 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1012 nodes · 2998 edges · 31 communities (30 shown, 1 thin omitted)
- Extraction: 83% EXTRACTED · 17% INFERRED · 0% AMBIGUOUS · INFERRED: 495 edges (avg confidence: 0.8)
- Token cost: 33,174 input · 1,780 output

## Community Hubs (Navigation)
- CLI Operations & Commands
- Broker Config & Verify Ops
- Config Validation
- Container Transport & Ops
- CLI Test Harness
- Config Defaults & Loading
- K8s Prep & Secrets
- Container Manager
- Node Rendering & Roles
- Command Runner (Exec)
- Broker Test Helpers
- Local Verify Tests
- Env Checks & Operator
- Broker Coverage Tests
- K8s Prep Tests
- Cluster Ops Tests
- Config Types & Accessors
- Transport Test Doubles
- Deploy & Operator Tests
- dev.sh Build Script
- K8s Transport Tests
- dev.ps1 Build Script
- Docs & Script Index
- Broker CLI Tests
- Broker Runtime Helpers
- K8s Cluster Apply/Delete
- Env Check Tests
- Broker ExecCLI Helpers
- Go Module Definition

## God Nodes (most connected - your core abstractions)
1. `App` - 113 edges
2. `Role` - 73 edges
3. `bg()` - 64 edges
4. `ctrCfg()` - 51 edges
5. `newCapMgr()` - 47 edges
6. `newTestOps()` - 42 edges
7. `Manager` - 39 edges
8. `k8sCluster()` - 32 edges
9. `NewCluster()` - 32 edges
10. `eqArgs()` - 32 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `Execute()`  [INFERRED]
  main.go → internal/cli/root.go
- `ctrOps()` --calls--> `New()`  [INFERRED]
  internal/cli/ops_container.go → internal/broker/broker.go
- `k8sOps()` --calls--> `New()`  [INFERRED]
  internal/cli/ops_k8s.go → internal/broker/broker.go
- `TestTransportEchoHidesUploadBody()` --calls--> `New()`  [INFERRED]
  internal/container/transport_test.go → internal/broker/broker.go
- `TestOperatorNSDefaultOnError()` --calls--> `New()`  [INFERRED]
  internal/k8s/cluster_test.go → internal/broker/broker.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]
- **Web-based Configuration Generators** — solace_repl_gen_html, solace_replication_generator_html, solace_yaml_generator_html [INFERRED 0.90]
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]

## Communities (31 total, 1 thin omitted)

### Community 0 - "CLI Operations & Commands"
Cohesion: 0.05
Nodes (136): App, opFunc, roleOpFunc, Command, newContainerCmd(), newCtrConfigCmd(), newCtrDeleteCmd(), newCtrDeployCmd() (+128 more)

### Community 1 - "Broker Config & Verify Ops"
Cohesion: 0.06
Nodes (53): showCmd, field(), TestHTTPStatusHelpers(), concatFiles(), Ops, Context, assertLeaderScript(), disableDefaultUsersScript() (+45 more)

### Community 2 - "Config Validation"
Cohesion: 0.10
Nodes (67): Platform, Config, missingErr(), platformKey(), requireAll(), sortStrings(), fileExists(), NewManager() (+59 more)

### Community 3 - "Container Transport & Ops"
Cohesion: 0.07
Nodes (28): fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrTransport, Role (+20 more)

### Community 4 - "CLI Test Harness"
Cohesion: 0.09
Nodes (55): capture(), captureStderr(), captureStdout(), collectPaths(), findCmd(), firstLine(), Command, File (+47 more)

### Community 5 - "Config Defaults & Loading"
Cohesion: 0.08
Nodes (50): assertContainerBlockDefaults(), assertContainerScaling(), Config, T, haNodesConfig(), TestApplyDefaultsDocker(), TestApplyDefaultsK8s(), TestApplyDefaultsK8sTLS() (+42 more)

### Community 6 - "K8s Prep & Secrets"
Cohesion: 0.09
Nodes (29): Config, Context, Cluster, Reader, Writer, isBuiltinLabel(), joinManifests(), namespaceManifest() (+21 more)

### Community 7 - "Container Manager"
Cohesion: 0.16
Nodes (10): Manager, defaultGenPSK(), Config, Context, Reader, Writer, orNone(), platformTitle() (+2 more)

### Community 8 - "Node Rendering & Roles"
Cohesion: 0.11
Nodes (33): Builder, NodeIdentity, Config, Context, Cluster, boolStr(), BrokerCR(), Compose() (+25 more)

### Community 9 - "Command Runner (Exec)"
Cohesion: 0.13
Nodes (23): Echo, Exec, Runner, Context, Writer, Quote(), quoteTok(), captureStdout() (+15 more)

### Community 10 - "Broker Test Helpers"
Cohesion: 0.16
Nodes (30): Buffer, Config, Ops, T, newTestOps(), TestContainsAnyFold(), TestCountContains(), TestDisableDefaultUsersNoVPNs() (+22 more)

### Community 11 - "Local Verify Tests"
Cohesion: 0.22
Nodes (26): uploadedForRole(), Buffer, Config, Ops, T, localCfg(), newLocalOps(), rd() (+18 more)

### Community 12 - "Env Checks & Operator"
Cohesion: 0.14
Nodes (13): Context, Cluster, orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), Config (+5 more)

### Community 13 - "Broker Coverage Tests"
Cohesion: 0.17
Nodes (24): New(), T, TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDomainCertsBadFilename(), TestExecCLIRunError(), TestExecCLIWarnsOnErrorOutput(), TestFieldLabelWithoutColon() (+16 more)

### Community 14 - "K8s Prep Tests"
Cohesion: 0.20
Nodes (24): NewCluster(), saCfg(), adminCfg(), Buffer, Cluster, Config, T, labelCluster() (+16 more)

### Community 15 - "Cluster Ops Tests"
Cohesion: 0.21
Nodes (23): Cluster, T, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived() (+15 more)

### Community 16 - "Config Types & Accessors"
Cohesion: 0.16
Nodes (20): Admin, Container, DockerConfig, DomainCerts, Image, K8sConfig, LoadBalancer, Network (+12 more)

### Community 17 - "Transport Test Doubles"
Cohesion: 0.22
Nodes (15): capCall, capRunner, Config, NewTransport(), dockerCfg(), eqArgs(), Config, Context (+7 more)

### Community 18 - "Deploy & Operator Tests"
Cohesion: 0.22
Nodes (19): T, TestDeleteBrokerNoPurge(), TestDeleteBrokerPurgeHA(), TestDeleteBrokerPurgeStandalone(), TestDeleteBrokerPurgeSwallowsPVCError(), TestDeployBrokerApply(), TestDeployBrokerKeepYAML(), boolPtr() (+11 more)

### Community 19 - "dev.sh Build Script"
Cohesion: 0.38
Nodes (20): log_init(), main(), NO_COLOR, dev.sh script, cap(), die(), ok(), step() (+12 more)

### Community 20 - "K8s Transport Tests"
Cohesion: 0.22
Nodes (11): Config, NewTransport(), Context, T, TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload() (+3 more)

### Community 21 - "dev.ps1 Build Script"
Cohesion: 0.41
Nodes (16): Cap(), Die(), Ok(), Step(), Task-All(), Task-Build(), Task-Ci(), Task-Cov() (+8 more)

### Community 22 - "Docs & Script Index"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 23 - "Broker CLI Tests"
Cohesion: 0.26
Nodes (13): matchCLI(), ranContains(), TestDiagnostics(), TestDisableDefaultUsers(), TestDisableDefaultVPN(), TestDomainCerts(), TestExecCLI(), TestPathHelpers() (+5 more)

### Community 24 - "Broker Runtime Helpers"
Cohesion: 0.25
Nodes (6): Transport, Duration, Ops, Config, Context, Writer

### Community 25 - "K8s Cluster Apply/Delete"
Cohesion: 0.22
Nodes (5): Config, Context, Cluster, Reader, Writer

### Community 26 - "Env Check Tests"
Cohesion: 0.26
Nodes (11): T, TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), Config, T (+3 more)

### Community 27 - "Broker ExecCLI Helpers"
Cohesion: 0.50
Nodes (3): containsAnyFold(), countContains(), validName()

## Knowledge Gaps
- **16 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+11 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Role` connect `Container Transport & Ops` to `CLI Operations & Commands`, `Broker Config & Verify Ops`, `K8s Prep & Secrets`, `Container Manager`, `Node Rendering & Roles`, `Local Verify Tests`, `Broker CLI Tests`, `Broker Runtime Helpers`, `Broker ExecCLI Helpers`?**
  _High betweenness centrality (0.310) - this node is a cross-community bridge._
- **Why does `App` connect `CLI Operations & Commands` to `Command Runner (Exec)`, `Config Validation`, `CLI Test Harness`?**
  _High betweenness centrality (0.257) - this node is a cross-community bridge._
- **Why does `Platform` connect `Config Validation` to `CLI Operations & Commands`, `Config Defaults & Loading`, `Container Manager`, `Node Rendering & Roles`, `Config Types & Accessors`, `Transport Test Doubles`?**
  _High betweenness centrality (0.166) - this node is a cross-community bridge._
- **Are the 22 inferred relationships involving `bg()` (e.g. with `ctrLogin()` and `opCtrCheck()`) actually correct?**
  _`bg()` has 22 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _16 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `CLI Operations & Commands` be split into smaller, more focused modules?**
  _Cohesion score 0.05167055167055167 - nodes in this community are weakly interconnected._
- **Should `Broker Config & Verify Ops` be split into smaller, more focused modules?**
  _Cohesion score 0.060759493670886074 - nodes in this community are weakly interconnected._