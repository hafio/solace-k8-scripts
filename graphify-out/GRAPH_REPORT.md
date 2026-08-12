# Graph Report - solace-k8-scripts  (2026-08-12)

## Corpus Check
- 71 files · ~116,986 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1463 nodes · 4007 edges · 33 communities (31 shown, 2 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 599 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `40e338f9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- scripts.go
- manager_test.go
- Role
- cli_test.go
- config_test.go
- Commands
- Manager
- config.go
- runner_test.go
- broker_test.go
- render.go
- RenderOperator
- eqArgs
- convert_test.go
- Test catalogue
- Platform
- NewTransport
- Cluster
- dev.sh
- Quadlet
- dev.ps1
- CLAUDE.md
- judge
- render_test.go
- NodeAffinity
- Cluster
- Command
- Go Module Definition
- .DeleteBroker
- Command reference

## God Nodes (most connected - your core abstractions)
1. `Commands` - 114 edges
2. `Role` - 76 edges
3. `bg()` - 69 edges
4. `ctrCfg()` - 68 edges
5. `Command` - 63 edges
6. `newCapMgr()` - 63 edges
7. `Manager` - 56 edges
8. `newTestOps()` - 46 edges
9. `Platform` - 36 edges
10. `eqArgs()` - 36 edges

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

## Communities (33 total, 2 thin omitted)

### Community 0 - "bg"
Cohesion: 0.07
Nodes (101): confirmDelete(), containerWhat(), ctrLogin(), ctrManager(), ctrOps(), emitCtrArtifact(), App, Ops (+93 more)

### Community 1 - "scripts.go"
Cohesion: 0.06
Nodes (57): showCmd, containsAnyFold(), countContains(), field(), TestHTTPStatusHelpers(), validCLILine(), validName(), concatFiles() (+49 more)

### Community 2 - "manager_test.go"
Cohesion: 0.11
Nodes (77): fileExists(), assertMode(), ctrCfg(), Buffer, Config, FileMode, T, hasCall() (+69 more)

### Community 3 - "Role"
Cohesion: 0.07
Nodes (23): fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrTransport, Role (+15 more)

### Community 4 - "cli_test.go"
Cohesion: 0.08
Nodes (71): capture(), captureStderr(), captureStdout(), collectPaths(), findCmd(), firstLine(), File, T (+63 more)

### Community 5 - "config_test.go"
Cohesion: 0.08
Nodes (61): assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, T, haNodesConfig(), TestApplyBridgePortDefaults(), TestApplyDefaultsDocker() (+53 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (114): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+106 more)

### Community 7 - "Manager"
Cohesion: 0.11
Nodes (15): NodeIdentity, Manager, Runner, defaultGenPSK(), Config, Context, FileMode, Reader (+7 more)

### Community 8 - "config.go"
Cohesion: 0.11
Nodes (23): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, HealthCheck, Image, K8sConfig (+15 more)

### Community 9 - "runner_test.go"
Cohesion: 0.14
Nodes (22): Echo, Exec, Context, Writer, Quote(), quoteTok(), captureStdout(), T (+14 more)

### Community 10 - "broker_test.go"
Cohesion: 0.05
Nodes (103): Transport, Duration, Ops, Config, Context, Writer, New(), Buffer (+95 more)

### Community 11 - "render.go"
Cohesion: 0.19
Nodes (24): Placement, PodAffinityTerm, boolStr(), BrokerCR(), cut(), Builder, groupKey(), itoa() (+16 more)

### Community 12 - "RenderOperator"
Cohesion: 0.14
Nodes (13): Context, Cluster, orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), Config (+5 more)

### Community 13 - "eqArgs"
Cohesion: 0.05
Nodes (98): T, TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster(), Cluster (+90 more)

### Community 14 - "convert_test.go"
Cohesion: 0.08
Nodes (54): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+46 more)

### Community 15 - "Test catalogue"
Cohesion: 0.05
Nodes (42): broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go, convert_test.go (+34 more)

### Community 16 - "Platform"
Cohesion: 0.06
Nodes (44): keyValueEntries, Platform, App, Config, step(), warn(), convertPlatform(), App (+36 more)

### Community 17 - "NewTransport"
Cohesion: 0.15
Nodes (23): capCall, capRunner, Config, T, TestCtrRuntimeDefaultArgvUnchanged(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), TestManagerReachableProbesRuntimeThenCompose() (+15 more)

### Community 18 - "Cluster"
Cohesion: 0.09
Nodes (31): GenSecrets(), Config, Context, Cluster, Reader, Writer, isBuiltinLabel(), joinManifests() (+23 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "Quadlet"
Cohesion: 0.23
Nodes (16): Compose(), ContainerSecrets(), EnvFile(), EnvPairs(), escapePercent(), Config, healthCmd(), Quadlet() (+8 more)

### Community 21 - "dev.ps1"
Cohesion: 0.19
Nodes (16): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+8 more)

### Community 22 - "CLAUDE.md"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 23 - "judge"
Cohesion: 0.21
Nodes (16): describe(), Builder, judge(), main(), plural(), T, load(), TestJudge() (+8 more)

### Community 24 - "render_test.go"
Cohesion: 0.27
Nodes (13): shQuote(), Config, T, healthCheckFixture(), load(), TestContainerSecretsRedundancy(), TestHealthCmdDefaultsToReadiness(), TestParsePort() (+5 more)

### Community 25 - "NodeAffinity"
Cohesion: 0.67
Nodes (3): NodeAffinity, NodeMatchExpr, WeightedNodeTerm

### Community 26 - "Cluster"
Cohesion: 0.24
Nodes (5): Config, Context, Cluster, Reader, Writer

### Community 27 - "Command"
Cohesion: 0.10
Nodes (58): opFunc, roleOpFunc, Command, FlagSet, availableSubs(), firstDiff(), Builder, T (+50 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

## Knowledge Gaps
- **162 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+157 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Role` connect `Role` to `bg`, `scripts.go`, `Manager`, `broker_test.go`, `Cluster`, `Command`?**
  _High betweenness centrality (0.220) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `manager_test.go`, `config_test.go`, `Manager`, `config.go`, `convert_test.go`, `NewTransport`, `Quadlet`, `render_test.go`, `Command`?**
  _High betweenness centrality (0.149) - this node is a cross-community bridge._
- **Why does `Command` connect `Command` to `Role`, `cli_test.go`, `Manager`, `config.go`, `Platform`, `Cluster`?**
  _High betweenness centrality (0.102) - this node is a cross-community bridge._
- **Are the 26 inferred relationships involving `bg()` (e.g. with `ctrLogin()` and `opCtrCheck()`) actually correct?**
  _`bg()` has 26 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `ctrCfg()` (e.g. with `TestCtrRuntimeDefaultArgvUnchanged()` and `wrappedCtrCfg()`) actually correct?**
  _`ctrCfg()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _162 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06646751306945482 - nodes in this community are weakly interconnected._