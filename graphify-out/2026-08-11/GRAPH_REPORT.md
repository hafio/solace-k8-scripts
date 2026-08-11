# Graph Report - solace-k8-scripts  (2026-08-11)

## Corpus Check
- 71 files · ~95,723 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1318 nodes · 3512 edges · 29 communities (28 shown, 1 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 544 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f2649362`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App
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
- newLocalOps
- RenderOperator
- eqArgs
- convert_test.go
- Test catalogue
- command_test.go
- capRunner
- secrets_test.go
- dev.sh
- dev.ps1
- CLAUDE.md
- judge
- Cluster
- commanddoc_test.go
- Go Module Definition
- Command reference

## God Nodes (most connected - your core abstractions)
1. `App` - 113 edges
2. `Commands` - 101 edges
3. `Role` - 73 edges
4. `bg()` - 64 edges
5. `Command` - 56 edges
6. `ctrCfg()` - 53 edges
7. `newCapMgr()` - 49 edges
8. `newTestOps()` - 42 edges
9. `Manager` - 39 edges
10. `eqArgs()` - 35 edges

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

## Communities (29 total, 1 thin omitted)

### Community 0 - "App"
Cohesion: 0.05
Nodes (134): App, opFunc, roleOpFunc, Command, newContainerCmd(), newCtrConfigCmd(), newCtrDeleteCmd(), newCtrDeployCmd() (+126 more)

### Community 1 - "scripts.go"
Cohesion: 0.07
Nodes (46): showCmd, field(), TestHTTPStatusHelpers(), assertLeaderScript(), disableDefaultUsersScript(), gatherConfigsScript(), noReleaseActivityScript(), parseVPNNames() (+38 more)

### Community 2 - "manager_test.go"
Cohesion: 0.13
Nodes (60): fileExists(), ctrCfg(), Buffer, Config, T, hasCall(), newCapMgr(), newEchoMgr() (+52 more)

### Community 3 - "Role"
Cohesion: 0.05
Nodes (43): fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrTransport, Role (+35 more)

### Community 4 - "cli_test.go"
Cohesion: 0.08
Nodes (63): capture(), captureStderr(), captureStdout(), collectPaths(), findCmd(), firstLine(), File, T (+55 more)

### Community 5 - "config_test.go"
Cohesion: 0.07
Nodes (59): Platform, assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, T, haNodesConfig(), TestApplyDefaultsDocker() (+51 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (101): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+93 more)

### Community 7 - "Manager"
Cohesion: 0.11
Nodes (15): containerTransport, Manager, Runner, defaultGenPSK(), Config, Context, Reader, Writer (+7 more)

### Community 8 - "config.go"
Cohesion: 0.06
Nodes (54): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, K8sConfig, LoadBalancer (+46 more)

### Community 9 - "runner_test.go"
Cohesion: 0.14
Nodes (22): Echo, Exec, Context, Writer, Quote(), quoteTok(), captureStdout(), T (+14 more)

### Community 10 - "broker_test.go"
Cohesion: 0.05
Nodes (83): Transport, Duration, containsAnyFold(), countContains(), Ops, Config, Context, Writer (+75 more)

### Community 11 - "newLocalOps"
Cohesion: 0.22
Nodes (26): uploadedForRole(), Buffer, Config, Ops, T, localCfg(), newLocalOps(), rd() (+18 more)

### Community 12 - "RenderOperator"
Cohesion: 0.14
Nodes (13): Context, Cluster, orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), Config (+5 more)

### Community 13 - "eqArgs"
Cohesion: 0.05
Nodes (91): T, TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster(), Cluster (+83 more)

### Community 14 - "convert_test.go"
Cohesion: 0.08
Nodes (52): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+44 more)

### Community 15 - "Test catalogue"
Cohesion: 0.05
Nodes (42): broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go, convert_test.go (+34 more)

### Community 16 - "command_test.go"
Cohesion: 0.17
Nodes (16): decodeRuntime(), T, TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults() (+8 more)

### Community 17 - "capRunner"
Cohesion: 0.16
Nodes (22): capCall, capRunner, Config, T, TestCtrRuntimeDefaultArgvUnchanged(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), withSudo() (+14 more)

### Community 18 - "secrets_test.go"
Cohesion: 0.23
Nodes (17): AdminSecret(), dockerRegistrySecret(), Config, operatorRegcred(), checkGolden(), decodeDataValue(), T, TestAdminSecretDecodes() (+9 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 21 - "dev.ps1"
Cohesion: 0.19
Nodes (16): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+8 more)

### Community 22 - "CLAUDE.md"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 23 - "judge"
Cohesion: 0.21
Nodes (16): describe(), Builder, judge(), main(), plural(), T, load(), TestJudge() (+8 more)

### Community 26 - "Cluster"
Cohesion: 0.24
Nodes (5): Config, Context, Cluster, Reader, Writer

### Community 27 - "commanddoc_test.go"
Cohesion: 0.30
Nodes (11): FlagSet, availableSubs(), firstDiff(), Builder, T, mdCell(), renderCommandDocs(), TestCommandDocs() (+3 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

## Knowledge Gaps
- **149 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+144 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Role` connect `Role` to `App`, `scripts.go`, `Manager`, `config.go`, `broker_test.go`, `newLocalOps`?**
  _High betweenness centrality (0.204) - this node is a cross-community bridge._
- **Why does `Platform` connect `config_test.go` to `App`, `manager_test.go`, `cli_test.go`, `Manager`, `config.go`, `convert_test.go`, `command_test.go`, `capRunner`?**
  _High betweenness centrality (0.196) - this node is a cross-community bridge._
- **Why does `App` connect `App` to `cli_test.go`, `config_test.go`, `Manager`?**
  _High betweenness centrality (0.173) - this node is a cross-community bridge._
- **Are the 22 inferred relationships involving `bg()` (e.g. with `ctrLogin()` and `opCtrCheck()`) actually correct?**
  _`bg()` has 22 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _149 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App` be split into smaller, more focused modules?**
  _Cohesion score 0.05324143442213565 - nodes in this community are weakly interconnected._
- **Should `scripts.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07281772953414745 - nodes in this community are weakly interconnected._