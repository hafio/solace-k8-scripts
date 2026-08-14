# Graph Report - solace-k8-scripts  (2026-08-13)

## Corpus Check
- 72 files · ~125,863 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1417 nodes · 4122 edges · 52 communities (46 shown, 6 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 555 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c417e533`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- scripts.go
- manager_test.go
- Cluster
- cli_test.go
- testing.T
- Commands
- Manager
- config.go
- runner_test.go
- broker_test.go
- Compose
- secrets_test.go
- eqArgs
- convert_test.go
- Test catalogue
- loadK8s
- Ops
- Cluster
- dev.sh
- .LeaderLocal
- dev.ps1
- CLAUDE.md
- judge
- newLocalOps
- Role
- haCfg
- github.com/spf13/cobra.Command
- render.go
- Go Module Definition
- Platform
- Command reference
- .CheckEnv
- Config
- .ExecCLI
- .resolveSecretRefs
- Image
- .releaseToBackup
- prep_test.go
- newEchoMgr
- TestClusterHonoursRuntime
- kubectlTransport
- containerTransport
- TestServerCert
- Cluster
- coverage_test.go
- Echo
- context.Context
- Load
- Cluster
- ContainerSecret

## God Nodes (most connected - your core abstractions)
1. `Commands` - 114 edges
2. `Role` - 76 edges
3. `ctrCfg()` - 72 edges
4. `bg()` - 69 edges
5. `Config` - 67 edges
6. `newCapMgr()` - 67 edges
7. `Manager` - 55 edges
8. `newTestOps()` - 46 edges
9. `Platform` - 37 edges
10. `eqArgs()` - 36 edges

## Surprising Connections (you probably didn't know these)
- `TestFieldLabelWithoutColon()` --calls--> `field()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker.go
- `activity()` --calls--> `countContains()`  [INFERRED]
  internal/broker/verify_ops.go → internal/broker/broker.go
- `matchCLI()` --calls--> `cliArg()`  [INFERRED]
  internal/broker/broker_test.go → internal/broker/transport.go
- `seqTransport()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/verify_local_test.go → internal/broker/broker_test.go
- `TestLeaderLocalSuccess()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/verify_local_test.go → internal/broker/broker_test.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (52 total, 6 thin omitted)

### Community 0 - "bg"
Cohesion: 0.07
Nodes (94): TestConfirmFlagShortcuts(), TestConfirmNonTTY(), confirmDelete(), confirmPurge(), isTTY(), containerWhat(), ctrLogin(), ctrManager() (+86 more)

### Community 1 - "scripts.go"
Cohesion: 0.10
Nodes (30): showCmd, assertLeaderScript(), disableDefaultUsersScript(), disableDefaultVPNScript(), domainCertsScript(), gatherConfigsScript(), noReleaseActivityScript(), parseVPNNames() (+22 more)

### Community 2 - "manager_test.go"
Cohesion: 0.10
Nodes (66): assertMode(), containsStr(), ctrCfg(), hasCall(), maskedKeys(), newCapMgr(), TestContainerRunningMatchesNameExactly(), TestManagerCheckDNSFailsLoudInHA() (+58 more)

### Community 3 - "Cluster"
Cohesion: 0.12
Nodes (12): Cluster, HARoles(), lbServiceName(), podName(), pvcName(), RestartOrder(), stsName(), TestResourceNames() (+4 more)

### Community 4 - "cli_test.go"
Cohesion: 0.06
Nodes (66): os.File, capture(), captureStderr(), captureStdout(), collectPaths(), findCmd(), firstLine(), runCtr() (+58 more)

### Community 5 - "testing.T"
Cohesion: 0.08
Nodes (60): testing.T, decodeRuntime(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults() (+52 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (114): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+106 more)

### Community 8 - "config.go"
Cohesion: 0.13
Nodes (22): AdditionalUser, Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, Network, Node (+14 more)

### Community 9 - "runner_test.go"
Cohesion: 0.20
Nodes (16): captureStdout(), helperCommand(), TestEchoDefaultWriter(), TestEchoOutput(), TestEchoRun(), TestEchoRunEnv(), TestEchoRunEnvNoEnv(), TestEchoRunInput() (+8 more)

### Community 10 - "broker_test.go"
Cohesion: 0.13
Nodes (36): Ops, matchCLI(), newTestOps(), TestDiagnostics(), TestDisableDefaultUsers(), TestDisableDefaultUsersNoVPNs(), TestDisableDefaultVPN(), TestDomainCerts() (+28 more)

### Community 11 - "Compose"
Cohesion: 0.16
Nodes (27): emitCtrArtifact(), NodeIdentity, Compose(), ContainerSecrets(), EnvFile(), EnvPairs(), escapePercent(), healthCmd() (+19 more)

### Community 12 - "secrets_test.go"
Cohesion: 0.15
Nodes (17): TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden(), decodeDataValue(), TestAdminSecretDecodes() (+9 more)

### Community 13 - "eqArgs"
Cohesion: 0.14
Nodes (30): TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster(), Cluster, newCluster() (+22 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (52): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+44 more)

### Community 15 - "Test catalogue"
Cohesion: 0.05
Nodes (42): broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go, convert_test.go (+34 more)

### Community 16 - "loadK8s"
Cohesion: 0.20
Nodes (16): TestDeleteBrokerNoPurge(), TestDeleteBrokerPurgeHA(), TestDeleteBrokerPurgeStandalone(), TestDeleteBrokerPurgeSwallowsPVCError(), TestDeployBrokerApply(), TestDeployBrokerKeepYAML(), boolPtr(), mustContain() (+8 more)

### Community 17 - "Ops"
Cohesion: 0.11
Nodes (21): capCall, capRunner, time.Duration, Ops, New(), TestNewDefaults(), Transport, TestCtrTransportHonoursRuntime() (+13 more)

### Community 18 - "Cluster"
Cohesion: 0.12
Nodes (13): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels() (+5 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - ".LeaderLocal"
Cohesion: 0.29
Nodes (5): Ops, hostMatches(), roleName(), shortHost(), TestRoleName()

### Community 21 - "dev.ps1"
Cohesion: 0.19
Nodes (16): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+8 more)

### Community 22 - "CLAUDE.md"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 23 - "judge"
Cohesion: 0.21
Nodes (14): describe(), judge(), main(), plural(), load(), TestJudge(), TestJudgeMalformedInput(), TestJudgeModuleFixHint() (+6 more)

### Community 24 - "newLocalOps"
Cohesion: 0.22
Nodes (22): uploadedForRole(), Ops, localCfg(), newLocalOps(), rd(), seqTransport(), TestLeaderLocalRejectsNonPrimary(), TestLeaderLocalStandaloneSkips() (+14 more)

### Community 25 - "Role"
Cohesion: 0.16
Nodes (9): fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrTransport, Config (+1 more)

### Community 26 - "haCfg"
Cohesion: 0.21
Nodes (10): Runner, haCfg(), NewTransport(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest() (+2 more)

### Community 27 - "github.com/spf13/cobra.Command"
Cohesion: 0.12
Nodes (55): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/pflag.FlagSet, TestFirstArgOr(), availableSubs(), firstDiff(), mdCell() (+47 more)

### Community 28 - "render.go"
Cohesion: 0.17
Nodes (25): strings.Builder, PodAffinityTerm, boolStr(), BrokerCR(), cut(), groupKey(), itoa(), parsePort() (+17 more)

### Community 31 - "Platform"
Cohesion: 0.08
Nodes (31): keyValueEntries, TestCommandArgsDoesNotAliasCommand(), TestValidateCommandAccepts(), TestValidateCommandRejects(), Command, Platform, TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator() (+23 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - ".CheckEnv"
Cohesion: 0.15
Nodes (7): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing()

### Community 34 - "Config"
Cohesion: 0.23
Nodes (10): Replication, Scaling, TLS, Config, GenOperator(), RenderOperator(), watchNamespace(), containerSecretSpecs() (+2 more)

### Community 35 - ".ExecCLI"
Cohesion: 0.19
Nodes (9): containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), TestValidName(), validCLILine(), validName(), Ops (+1 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 38 - ".releaseToBackup"
Cohesion: 0.18
Nodes (13): field(), TestField(), TestHTTPStatusHelpers(), TestLastLines(), TestPrimaryRedundancyUp(), TestLastLinesEqualCount(), activity(), Ops (+5 more)

### Community 39 - "prep_test.go"
Cohesion: 0.23
Nodes (16): saCfg(), TestHARoles(), TestProductKeyRoles(), adminCfg(), Cluster, labelCluster(), TestCreateSecretsAdminOnly(), TestCreateSecretsPreflight() (+8 more)

### Community 40 - "newEchoMgr"
Cohesion: 0.15
Nodes (14): bytes.Buffer, fileExists(), NewManager(), newEchoMgr(), TestManagerCheckDryRun(), TestManagerDeletePodmanPurgeRootless(), TestManagerDeployDockerDryRunMasksSecretEnv(), TestManagerDeployPodmanDryRunHidesSecretBytes() (+6 more)

### Community 41 - "TestClusterHonoursRuntime"
Cohesion: 0.60
Nodes (5): TestClusterHonoursRuntime(), TestRuntimeDefaultArgvUnchanged(), TestTransportHonoursRuntime(), withLeading(), wrappedCfg()

### Community 44 - "TestServerCert"
Cohesion: 0.22
Nodes (10): TestPathHelpers(), TestServerCert(), writeFile(), concatFiles(), TestServerCertCAReadError(), serverCertFile(), serverCertScript(), TestServerCertScript() (+2 more)

### Community 45 - "Cluster"
Cohesion: 0.17
Nodes (9): io.Reader, io.Writer, TestPromptYes(), TestPromptYesNo(), confirmRestart(), promptLine(), promptYes(), promptYesNo() (+1 more)

### Community 46 - "coverage_test.go"
Cohesion: 0.12
Nodes (16): ranContains(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDomainCertsBadFilename(), TestExecCLIRunError(), TestExecCLIWarnsOnErrorOutput(), TestFieldLabelWithoutColon(), TestOutDefaultsToStdout() (+8 more)

### Community 47 - "Echo"
Cohesion: 0.29
Nodes (7): Echo, MaskEnv(), Quote(), quoteTok(), TestMaskEnv(), TestQuote(), TestQuoteTok()

### Community 48 - "context.Context"
Cohesion: 0.30
Nodes (3): Exec, context.Context, Cluster

### Community 49 - "Load"
Cohesion: 0.24
Nodes (15): minimalK8s(), TestLoadBashEnvFileHint(), TestLoadNotYAMLHint(), TestLoadParseError(), TestLoadReadError(), TestLoadResolvesSecretRefs(), TestLoadSecretRefErrors(), TestLoadSuccess() (+7 more)

### Community 50 - "Cluster"
Cohesion: 0.27
Nodes (5): Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

## Knowledge Gaps
- **162 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+157 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `manager_test.go`, `Cluster`, `cli_test.go`, `Manager`, `config.go`, `broker_test.go`, `Compose`, `secrets_test.go`, `eqArgs`, `convert_test.go`, `loadK8s`, `Ops`, `Cluster`, `newLocalOps`, `haCfg`, `render.go`, `Image`, `prep_test.go`, `newEchoMgr`, `TestClusterHonoursRuntime`, `kubectlTransport`, `Cluster`?**
  _High betweenness centrality (0.090) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `.CheckEnv`, `manager_test.go`, `cli_test.go`, `testing.T`, `Manager`, `config.go`, `newEchoMgr`, `Compose`, `convert_test.go`, `Load`, `Ops`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.062) - this node is a cross-community bridge._
- **Why does `Manager` connect `Manager` to `bg`, `.CheckEnv`, `Config`, `manager_test.go`, `newEchoMgr`, `Cluster`, `haCfg`, `Platform`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `ctrCfg()` (e.g. with `TestCtrRuntimeDefaultArgvUnchanged()` and `wrappedCtrCfg()`) actually correct?**
  _`ctrCfg()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _162 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.07478070175438596 - nodes in this community are weakly interconnected._
- **Should `scripts.go` be split into smaller, more focused modules?**
  _Cohesion score 0.0957983193277311 - nodes in this community are weakly interconnected._