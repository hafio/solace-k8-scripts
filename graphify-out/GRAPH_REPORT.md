# Graph Report - solace-k8-scripts  (2026-08-10)

## Corpus Check
- 68 files · ~91,325 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1281 nodes · 3392 edges · 38 communities (37 shown, 1 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 527 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f631755c`
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
- render.go
- runner_test.go
- broker_test.go
- newLocalOps
- RenderOperator
- NewCluster
- Platform
- Test catalogue
- config.go
- Transport Test Doubles
- Cluster
- dev.sh
- coverage_test.go
- dev.ps1
- CLAUDE.md
- judge
- cliScriptPath
- Ops
- Cluster
- commanddoc_test.go
- Go Module Definition
- .ExecCLI
- Command reference
- ops_container.go
- newK8sCmd
- helpers.go
- newContainerCmd
- HARoles

## God Nodes (most connected - your core abstractions)
1. `App` - 113 edges
2. `Commands` - 101 edges
3. `Role` - 73 edges
4. `bg()` - 64 edges
5. `ctrCfg()` - 51 edges
6. `newCapMgr()` - 47 edges
7. `newTestOps()` - 42 edges
8. `Manager` - 39 edges
9. `k8sCluster()` - 32 edges
10. `NewCluster()` - 32 edges

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

### Community 0 - "App"
Cohesion: 0.13
Nodes (51): App, Config, bg(), domainCANames(), Cluster, Config, Context, Ops (+43 more)

### Community 1 - "scripts.go"
Cohesion: 0.06
Nodes (53): showCmd, field(), TestHTTPStatusHelpers(), concatFiles(), Ops, Context, assertLeaderScript(), disableDefaultUsersScript() (+45 more)

### Community 2 - "manager_test.go"
Cohesion: 0.13
Nodes (60): fileExists(), ctrCfg(), Buffer, Config, T, hasCall(), newCapMgr(), newEchoMgr() (+52 more)

### Community 3 - "Role"
Cohesion: 0.08
Nodes (22): fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrTransport, Role (+14 more)

### Community 4 - "cli_test.go"
Cohesion: 0.10
Nodes (59): capture(), captureStderr(), captureStdout(), collectPaths(), findCmd(), firstLine(), Command, File (+51 more)

### Community 5 - "config_test.go"
Cohesion: 0.09
Nodes (51): step(), assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, T, haNodesConfig(), TestApplyDefaultsDocker() (+43 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (101): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+93 more)

### Community 7 - "Manager"
Cohesion: 0.15
Nodes (12): Manager, Runner, defaultGenPSK(), Config, Context, Reader, Writer, NewManager() (+4 more)

### Community 8 - "render.go"
Cohesion: 0.11
Nodes (34): NodeIdentity, Config, Context, Cluster, boolStr(), BrokerCR(), Compose(), cut() (+26 more)

### Community 9 - "runner_test.go"
Cohesion: 0.14
Nodes (22): Echo, Exec, Context, Writer, Quote(), quoteTok(), captureStdout(), T (+14 more)

### Community 10 - "broker_test.go"
Cohesion: 0.16
Nodes (30): Buffer, Config, Ops, T, newTestOps(), TestContainsAnyFold(), TestCountContains(), TestDisableDefaultUsersNoVPNs() (+22 more)

### Community 11 - "newLocalOps"
Cohesion: 0.22
Nodes (26): uploadedForRole(), Buffer, Config, Ops, T, localCfg(), newLocalOps(), rd() (+18 more)

### Community 12 - "RenderOperator"
Cohesion: 0.14
Nodes (13): Context, Cluster, orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), Config (+5 more)

### Community 13 - "NewCluster"
Cohesion: 0.06
Nodes (88): T, TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster(), Cluster (+80 more)

### Community 14 - "Platform"
Cohesion: 0.06
Nodes (58): Platform, doc, Result, vars, convertPlatform(), Command, newConvertCmd(), runConvert() (+50 more)

### Community 15 - "Test catalogue"
Cohesion: 0.05
Nodes (39): broker_test.go, check_test.go, cli_test.go, cluster_test.go, commanddoc_test.go, config_test.go, convert_test.go, Coverage (+31 more)

### Community 16 - "config.go"
Cohesion: 0.09
Nodes (29): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, K8sConfig, LoadBalancer (+21 more)

### Community 17 - "Transport Test Doubles"
Cohesion: 0.22
Nodes (15): capCall, capRunner, Config, NewTransport(), dockerCfg(), eqArgs(), Config, Context (+7 more)

### Community 18 - "Cluster"
Cohesion: 0.09
Nodes (30): Config, Context, Cluster, Reader, Writer, isBuiltinLabel(), joinManifests(), namespaceManifest() (+22 more)

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

### Community 24 - "cliScriptPath"
Cohesion: 0.26
Nodes (13): matchCLI(), ranContains(), TestDiagnostics(), TestDisableDefaultUsers(), TestDisableDefaultVPN(), TestDomainCerts(), TestExecCLI(), TestPathHelpers() (+5 more)

### Community 25 - "Ops"
Cohesion: 0.25
Nodes (6): Transport, Duration, Ops, Config, Context, Writer

### Community 26 - "Cluster"
Cohesion: 0.22
Nodes (5): Config, Context, Cluster, Reader, Writer

### Community 27 - "commanddoc_test.go"
Cohesion: 0.31
Nodes (12): FlagSet, availableSubs(), firstDiff(), Builder, Command, T, mdCell(), renderCommandDocs() (+4 more)

### Community 31 - ".ExecCLI"
Cohesion: 0.50
Nodes (3): containsAnyFold(), countContains(), validName()

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "ops_container.go"
Cohesion: 0.13
Nodes (30): containerWhat(), ctrLogin(), ctrManager(), ctrOps(), emitCtrArtifact(), Ops, opCtrCheck(), opCtrCLI() (+22 more)

### Community 34 - "newK8sCmd"
Cohesion: 0.24
Nodes (24): addDataFlags(), genCapable(), Command, leaf(), roleLeaf(), Command, newK8sCheckCmd(), newK8sCmd() (+16 more)

### Community 35 - "helpers.go"
Cohesion: 0.17
Nodes (18): opFunc, roleOpFunc, TestConfirmFlagShortcuts(), warn(), confirmDelete(), confirmPurge(), emit(), firstArgOr() (+10 more)

### Community 36 - "newContainerCmd"
Cohesion: 0.38
Nodes (12): Command, newContainerCmd(), newCtrConfigCmd(), newCtrDeleteCmd(), newCtrDeployCmd(), newCtrDownCmd(), newCtrGenCmd(), newCtrPrepCmd() (+4 more)

### Community 37 - "HARoles"
Cohesion: 0.31
Nodes (8): opCtrVerifyDiagnostics(), nowStamp(), opK8sVerifyDiagnostics(), Config, HARoles(), lbServiceName(), ProductKeyRoles(), pvcName()

## Knowledge Gaps
- **146 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+141 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Platform` connect `Platform` to `App`, `manager_test.go`, `newContainerCmd`, `config_test.go`, `Manager`, `render.go`, `config.go`, `Transport Test Doubles`?**
  _High betweenness centrality (0.242) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `App`, `scripts.go`, `ops_container.go`, `newContainerCmd`, `HARoles`, `Manager`, `render.go`, `newLocalOps`, `Cluster`, `cliScriptPath`, `Ops`, `.ExecCLI`?**
  _High betweenness centrality (0.229) - this node is a cross-community bridge._
- **Why does `App` connect `App` to `ops_container.go`, `newK8sCmd`, `helpers.go`, `newContainerCmd`, `config_test.go`, `HARoles`, `Manager`, `cli_test.go`, `Platform`?**
  _High betweenness centrality (0.225) - this node is a cross-community bridge._
- **Are the 22 inferred relationships involving `bg()` (e.g. with `ctrLogin()` and `opCtrCheck()`) actually correct?**
  _`bg()` has 22 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _146 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App` be split into smaller, more focused modules?**
  _Cohesion score 0.13499245852187028 - nodes in this community are weakly interconnected._
- **Should `scripts.go` be split into smaller, more focused modules?**
  _Cohesion score 0.060759493670886074 - nodes in this community are weakly interconnected._