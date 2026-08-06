# Graph Report - solace-k8-scripts  (2026-08-06)

## Corpus Check
- 56 files · ~67,661 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1015 nodes · 2932 edges · 26 communities (25 shown, 1 thin omitted)
- Extraction: 83% EXTRACTED · 17% INFERRED · 0% AMBIGUOUS · INFERRED: 495 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `e07efc0c`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App
- scripts.go
- manager_test.go
- Role
- cli_test.go
- config_test.go
- Platform
- Manager
- render.go
- runner_test.go
- broker_test.go
- newLocalOps
- RenderOperator
- .validateContainer
- NewCluster
- Load
- config.go
- Transport Test Doubles
- dev.sh
- recRunner
- dev.ps1
- CLAUDE.md
- Cluster
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

## Communities (26 total, 1 thin omitted)

### Community 0 - "App"
Cohesion: 0.05
Nodes (136): App, opFunc, roleOpFunc, Command, newContainerCmd(), newCtrConfigCmd(), newCtrDeleteCmd(), newCtrDeployCmd() (+128 more)

### Community 1 - "scripts.go"
Cohesion: 0.08
Nodes (43): showCmd, field(), TestHTTPStatusHelpers(), assertLeaderScript(), disableDefaultUsersScript(), disableDefaultVPNScript(), gatherConfigsScript(), noReleaseActivityScript() (+35 more)

### Community 2 - "manager_test.go"
Cohesion: 0.13
Nodes (60): fileExists(), ctrCfg(), Buffer, Config, T, hasCall(), newCapMgr(), newEchoMgr() (+52 more)

### Community 3 - "Role"
Cohesion: 0.05
Nodes (40): fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrTransport, Role (+32 more)

### Community 4 - "cli_test.go"
Cohesion: 0.09
Nodes (55): capture(), captureStderr(), captureStdout(), collectPaths(), findCmd(), firstLine(), Command, File (+47 more)

### Community 5 - "config_test.go"
Cohesion: 0.14
Nodes (33): assertContainerBlockDefaults(), assertContainerScaling(), Config, T, haNodesConfig(), TestApplyDefaultsDocker(), TestApplyDefaultsK8s(), TestApplyDefaultsK8sTLS() (+25 more)

### Community 6 - "Platform"
Cohesion: 0.20
Nodes (10): Platform, TestResolveEnvPath(), TestValidateUnknownPlatform(), applyContainerBlockDefaults(), defaultK8sPorts(), Config, ResolveEnvPath(), setDefault() (+2 more)

### Community 7 - "Manager"
Cohesion: 0.15
Nodes (12): Manager, Runner, defaultGenPSK(), Config, Context, Reader, Writer, NewManager() (+4 more)

### Community 8 - "render.go"
Cohesion: 0.11
Nodes (33): Builder, NodeIdentity, Config, Context, Cluster, boolStr(), BrokerCR(), Compose() (+25 more)

### Community 9 - "runner_test.go"
Cohesion: 0.14
Nodes (22): Echo, Exec, Context, Writer, Quote(), quoteTok(), captureStdout(), T (+14 more)

### Community 10 - "broker_test.go"
Cohesion: 0.05
Nodes (86): Transport, Duration, containsAnyFold(), countContains(), Ops, Config, Context, Writer (+78 more)

### Community 11 - "newLocalOps"
Cohesion: 0.22
Nodes (26): uploadedForRole(), Buffer, Config, Ops, T, localCfg(), newLocalOps(), rd() (+18 more)

### Community 12 - "RenderOperator"
Cohesion: 0.14
Nodes (13): Context, Cluster, orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), Config (+5 more)

### Community 13 - ".validateContainer"
Cohesion: 0.42
Nodes (5): Config, missingErr(), platformKey(), requireAll(), sortStrings()

### Community 14 - "NewCluster"
Cohesion: 0.06
Nodes (94): T, TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster(), Cluster (+86 more)

### Community 15 - "Load"
Cohesion: 0.36
Nodes (8): TestLoadParseError(), TestLoadReadError(), TestLoadSuccess(), TestLoadUnknownField(), TestLoadValidationError(), writeTempYAML(), Config, Load()

### Community 16 - "config.go"
Cohesion: 0.16
Nodes (20): Admin, Container, DockerConfig, DomainCerts, Image, K8sConfig, LoadBalancer, Network (+12 more)

### Community 17 - "Transport Test Doubles"
Cohesion: 0.22
Nodes (15): capCall, capRunner, Config, NewTransport(), dockerCfg(), eqArgs(), Config, Context (+7 more)

### Community 19 - "dev.sh"
Cohesion: 0.18
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "recRunner"
Cohesion: 0.22
Nodes (11): Config, NewTransport(), Context, T, TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload() (+3 more)

### Community 21 - "dev.ps1"
Cohesion: 0.18
Nodes (16): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+8 more)

### Community 22 - "CLAUDE.md"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 25 - "Cluster"
Cohesion: 0.22
Nodes (5): Config, Context, Cluster, Reader, Writer

## Knowledge Gaps
- **16 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+11 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Role` connect `Role` to `App`, `scripts.go`, `Manager`, `render.go`, `broker_test.go`, `newLocalOps`?**
  _High betweenness centrality (0.309) - this node is a cross-community bridge._
- **Why does `App` connect `App` to `cli_test.go`, `Platform`, `Manager`?**
  _High betweenness centrality (0.256) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `App`, `manager_test.go`, `config_test.go`, `Manager`, `render.go`, `.validateContainer`, `Load`, `config.go`, `Transport Test Doubles`?**
  _High betweenness centrality (0.165) - this node is a cross-community bridge._
- **Are the 22 inferred relationships involving `bg()` (e.g. with `ctrLogin()` and `opCtrCheck()`) actually correct?**
  _`bg()` has 22 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _16 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App` be split into smaller, more focused modules?**
  _Cohesion score 0.05167055167055167 - nodes in this community are weakly interconnected._
- **Should `scripts.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07548076923076923 - nodes in this community are weakly interconnected._