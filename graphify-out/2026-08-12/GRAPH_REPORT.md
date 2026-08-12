# Graph Report - solace-k8-scripts  (2026-08-12)

## Corpus Check
- 71 files · ~114,215 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1452 nodes · 3970 edges · 38 communities (37 shown, 1 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 593 edges (avg confidence: 0.8)
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
- render.go
- runner_test.go
- broker_test.go
- newLocalOps
- Cluster
- NewCluster
- convert_test.go
- Test catalogue
- command_test.go
- NewTransport
- secrets_test.go
- dev.sh
- coverage_test.go
- dev.ps1
- CLAUDE.md
- judge
- eqArgs
- Ops
- Cluster
- Command
- Go Module Definition
- .ExecCLI
- Command reference
- prep_test.go
- NewTransport
- RenderOperator
- haCfg
- TestClusterHonoursRuntime

## God Nodes (most connected - your core abstractions)
1. `Commands` - 114 edges
2. `Role` - 76 edges
3. `bg()` - 69 edges
4. `ctrCfg()` - 68 edges
5. `Command` - 63 edges
6. `newCapMgr()` - 63 edges
7. `Manager` - 56 edges
8. `newTestOps()` - 46 edges
9. `eqArgs()` - 36 edges
10. `Platform` - 35 edges

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

## Communities (38 total, 1 thin omitted)

### Community 0 - "bg"
Cohesion: 0.07
Nodes (100): confirmDelete(), containerWhat(), ctrLogin(), ctrManager(), ctrOps(), App, Ops, opCtrCheck() (+92 more)

### Community 1 - "scripts.go"
Cohesion: 0.06
Nodes (54): showCmd, field(), TestHTTPStatusHelpers(), validCLILine(), concatFiles(), Ops, Context, assertLeaderScript() (+46 more)

### Community 2 - "manager_test.go"
Cohesion: 0.11
Nodes (77): fileExists(), assertMode(), ctrCfg(), Buffer, Config, FileMode, T, hasCall() (+69 more)

### Community 3 - "Role"
Cohesion: 0.05
Nodes (36): fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrTransport, Role (+28 more)

### Community 4 - "cli_test.go"
Cohesion: 0.09
Nodes (62): capture(), captureStderr(), captureStdout(), collectPaths(), findCmd(), firstLine(), File, T (+54 more)

### Community 5 - "config_test.go"
Cohesion: 0.05
Nodes (80): Platform, App, Config, step(), warn(), convertPlatform(), App, newConvertCmd() (+72 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (114): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+106 more)

### Community 7 - "Manager"
Cohesion: 0.11
Nodes (15): Manager, Runner, defaultGenPSK(), Config, Context, FileMode, Reader, Writer (+7 more)

### Community 8 - "render.go"
Cohesion: 0.05
Nodes (76): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, HealthCheck, Image, K8sConfig (+68 more)

### Community 9 - "runner_test.go"
Cohesion: 0.14
Nodes (22): Echo, Exec, Context, Writer, Quote(), quoteTok(), captureStdout(), T (+14 more)

### Community 10 - "broker_test.go"
Cohesion: 0.13
Nodes (44): Buffer, Config, Ops, T, matchCLI(), newTestOps(), ranContains(), TestDiagnostics() (+36 more)

### Community 11 - "newLocalOps"
Cohesion: 0.22
Nodes (26): uploadedForRole(), Buffer, Config, Ops, T, localCfg(), newLocalOps(), rd() (+18 more)

### Community 12 - "Cluster"
Cohesion: 0.29
Nodes (6): Context, Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

### Community 13 - "NewCluster"
Cohesion: 0.18
Nodes (26): T, TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster(), T (+18 more)

### Community 14 - "convert_test.go"
Cohesion: 0.08
Nodes (54): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+46 more)

### Community 15 - "Test catalogue"
Cohesion: 0.05
Nodes (42): broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go, convert_test.go (+34 more)

### Community 16 - "command_test.go"
Cohesion: 0.13
Nodes (24): keyValueEntries, decodeRuntime(), T, TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors() (+16 more)

### Community 17 - "NewTransport"
Cohesion: 0.15
Nodes (23): capCall, capRunner, Config, T, TestCtrRuntimeDefaultArgvUnchanged(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), TestManagerReachableProbesRuntimeThenCompose() (+15 more)

### Community 18 - "secrets_test.go"
Cohesion: 0.23
Nodes (17): AdminSecret(), dockerRegistrySecret(), Config, operatorRegcred(), checkGolden(), decodeDataValue(), T, TestAdminSecretDecodes() (+9 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "coverage_test.go"
Cohesion: 0.17
Nodes (24): New(), T, TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDomainCertsBadFilename(), TestExecCLIRunError(), TestExecCLIWarnsOnErrorOutput(), TestFieldLabelWithoutColon() (+16 more)

### Community 21 - "dev.ps1"
Cohesion: 0.19
Nodes (16): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+8 more)

### Community 22 - "CLAUDE.md"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 23 - "judge"
Cohesion: 0.21
Nodes (16): describe(), Builder, judge(), main(), plural(), T, load(), TestJudge() (+8 more)

### Community 24 - "eqArgs"
Cohesion: 0.20
Nodes (25): Cluster, T, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived() (+17 more)

### Community 25 - "Ops"
Cohesion: 0.24
Nodes (6): Transport, Duration, Ops, Config, Context, Writer

### Community 26 - "Cluster"
Cohesion: 0.24
Nodes (5): Config, Context, Cluster, Reader, Writer

### Community 27 - "Command"
Cohesion: 0.09
Nodes (66): opFunc, roleOpFunc, Command, FlagSet, availableSubs(), firstDiff(), Builder, T (+58 more)

### Community 31 - ".ExecCLI"
Cohesion: 0.29
Nodes (6): containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), TestValidName(), validName()

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "prep_test.go"
Cohesion: 0.20
Nodes (23): saCfg(), adminCfg(), Buffer, Cluster, Config, T, labelCluster(), TestCreateNamespace() (+15 more)

### Community 34 - "NewTransport"
Cohesion: 0.22
Nodes (11): Config, NewTransport(), Context, T, TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload() (+3 more)

### Community 35 - "RenderOperator"
Cohesion: 0.27
Nodes (7): GenOperator(), Config, Context, Cluster, RenderOperator(), watchNamespace(), operatorTmplVars

### Community 36 - "haCfg"
Cohesion: 0.50
Nodes (7): Config, T, haCfg(), TestHARoles(), TestProductKeyRoles(), TestResourceNames(), TestRestartOrder()

### Community 37 - "TestClusterHonoursRuntime"
Cohesion: 0.46
Nodes (7): Config, T, TestClusterHonoursRuntime(), TestRuntimeDefaultArgvUnchanged(), TestTransportHonoursRuntime(), withLeading(), wrappedCfg()

## Knowledge Gaps
- **162 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+157 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Role` connect `Role` to `bg`, `scripts.go`, `Manager`, `render.go`, `newLocalOps`, `Ops`, `Command`, `.ExecCLI`?**
  _High betweenness centrality (0.211) - this node is a cross-community bridge._
- **Why does `Platform` connect `config_test.go` to `manager_test.go`, `Manager`, `render.go`, `convert_test.go`, `command_test.go`, `NewTransport`, `Command`?**
  _High betweenness centrality (0.148) - this node is a cross-community bridge._
- **Why does `Command` connect `Command` to `Role`, `cli_test.go`, `config_test.go`, `Manager`, `render.go`, `command_test.go`, `Cluster`?**
  _High betweenness centrality (0.100) - this node is a cross-community bridge._
- **Are the 26 inferred relationships involving `bg()` (e.g. with `ctrLogin()` and `opCtrCheck()`) actually correct?**
  _`bg()` has 26 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `ctrCfg()` (e.g. with `TestCtrRuntimeDefaultArgvUnchanged()` and `wrappedCtrCfg()`) actually correct?**
  _`ctrCfg()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _162 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06700932800304588 - nodes in this community are weakly interconnected._