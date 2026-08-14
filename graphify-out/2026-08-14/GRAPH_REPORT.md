# Graph Report - solace-k8-scripts  (2026-08-14)

## Corpus Check
- 72 files · ~145,220 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1552 nodes · 4682 edges · 52 communities (47 shown, 5 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 685 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c417e533`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- scripts.go
- manager_test.go
- context.Context
- cli_test.go
- config_test.go
- Commands
- Manager
- Config
- testing.T
- newTestOps
- render.go
- NewCluster
- eqArgs
- convert_test.go
- Test catalogue
- vars
- Ops
- Cluster
- dev.sh
- prep_test.go
- dev.ps1
- CLAUDE.md
- judge
- cliScriptPath
- Role
- load.go
- github.com/spf13/cobra.Command
- strings.Builder
- Go Module Definition
- Platform
- Command reference
- .CheckEnv
- RenderOperator
- .AdditionalUsers
- .resolveSecretRefs
- recRunner
- emitYAML
- render_test.go
- newEchoMgr
- haCfg
- kubectlTransport
- container/runtime_test.go
- TestClusterHonoursRuntime
- Cluster
- coverage_test.go
- .Run
- Image
- Convert
- Cluster
- parsePort

## God Nodes (most connected - your core abstractions)
1. `Commands` - 115 edges
2. `Role` - 81 edges
3. `newTestOps()` - 73 edges
4. `ctrCfg()` - 72 edges
5. `Config` - 71 edges
6. `bg()` - 70 edges
7. `newCapMgr()` - 67 edges
8. `Manager` - 55 edges
9. `matchCLI()` - 52 edges
10. `NewCluster()` - 43 edges

## Surprising Connections (you probably didn't know these)
- `TestFieldLabelWithoutColon()` --calls--> `field()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker.go
- `matchCLI()` --calls--> `cliArg()`  [INFERRED]
  internal/broker/broker_test.go → internal/broker/transport.go
- `TestAdditionalUsersRunCLITransportError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestReleaseToBackupReleasedTimeout()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestServerCertRunCLIError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (52 total, 5 thin omitted)

### Community 0 - "bg"
Cohesion: 0.08
Nodes (91): TestCtrManagerConfirmWiring(), containerWhat(), ctrLogin(), ctrManager(), ctrOps(), App, opCtrCheck(), opCtrCLI() (+83 more)

### Community 1 - "scripts.go"
Cohesion: 0.05
Nodes (55): showCmd, countContains(), field(), TestCountContains(), TestField(), TestHTTPStatusHelpers(), TestLastLines(), TestPrimaryRedundancyUp() (+47 more)

### Community 2 - "manager_test.go"
Cohesion: 0.11
Nodes (65): assertMode(), containsStr(), ctrCfg(), hasCall(), maskedKeys(), newCapMgr(), TestContainerRunningMatchesNameExactly(), TestManagerCheckDNSFailsLoudInHA() (+57 more)

### Community 3 - "context.Context"
Cohesion: 0.11
Nodes (13): context.Context, Cluster, HARoles(), lbServiceName(), podName(), pvcName(), RestartOrder(), stsName() (+5 more)

### Community 4 - "cli_test.go"
Cohesion: 0.06
Nodes (90): opCall, opRunner, capture(), captureStderr(), captureStdout(), collectPaths(), failDisableDefaultUsersUpload(), findCmd() (+82 more)

### Community 5 - "config_test.go"
Cohesion: 0.06
Nodes (60): assertContainerBlockDefaults(), assertContainerScaling(), Config, haNodesConfig(), minimalK8s(), TestApplyDefaultsDocker(), TestApplyDefaultsK8s(), TestApplyDefaultsK8sTLS() (+52 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (115): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+107 more)

### Community 8 - "Config"
Cohesion: 0.11
Nodes (24): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, Network, Node, Nodes (+16 more)

### Community 9 - "testing.T"
Cohesion: 0.13
Nodes (31): testing.T, decodeRuntime(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults() (+23 more)

### Community 10 - "newTestOps"
Cohesion: 0.09
Nodes (44): appUsers(), Ops, newTestOps(), TestAdditionalUsers(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDiagnostics() (+36 more)

### Community 11 - "render.go"
Cohesion: 0.17
Nodes (22): emitCtrArtifact(), NodeIdentity, Compose(), ContainerSecrets(), containerSecretSpecs(), EnvFile(), EnvPairs(), escapePercent() (+14 more)

### Community 12 - "NewCluster"
Cohesion: 0.17
Nodes (26): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckStorageClass(), TestResolveStorageClass(), NewCluster(), TestDeleteBrokerNoPurge() (+18 more)

### Community 13 - "eqArgs"
Cohesion: 0.17
Nodes (25): TestReachable(), Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived() (+17 more)

### Community 14 - "convert_test.go"
Cohesion: 0.23
Nodes (24): checkGolden(), convertOK(), hasWarning(), strictDecode(), TestConvertBadBooleanWarns(), TestConvertBadNumberWarns(), TestConvertBashPlumbingIsSilent(), TestConvertContainer() (+16 more)

### Community 15 - "Test catalogue"
Cohesion: 0.05
Nodes (42): broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go, convert_test.go (+34 more)

### Community 16 - "vars"
Cohesion: 0.13
Nodes (15): vars, countMarkers(), resolvePlatform(), TestParseAssignmentForms(), TestParseCRLF(), TestParseEscapedQuote(), TestParseScalarListFallback(), TestUnmappedTracksFileOrder() (+7 more)

### Community 17 - "Ops"
Cohesion: 0.12
Nodes (16): capCall, capRunner, time.Duration, Ops, New(), TestNewDefaults(), Transport, NewTransport() (+8 more)

### Community 18 - "Cluster"
Cohesion: 0.07
Nodes (28): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), rolePlacementLabels(), splitLabel() (+20 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "prep_test.go"
Cohesion: 0.17
Nodes (23): saCfg(), adminCfg(), Cluster, labelCluster(), TestCreateNamespaceApplyFails(), TestCreateSecretsAdminOnly(), TestCreateSecretsAllThree(), TestCreateSecretsPreflight() (+15 more)

### Community 21 - "dev.ps1"
Cohesion: 0.19
Nodes (16): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+8 more)

### Community 22 - "CLAUDE.md"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 23 - "judge"
Cohesion: 0.16
Nodes (21): describe(), judge(), main(), plural(), run(), load(), TestJudge(), TestJudgeEmptyTrace() (+13 more)

### Community 24 - "cliScriptPath"
Cohesion: 0.15
Nodes (36): uploadedForRole(), TestReleaseToBackupReleasedTimeout(), cliScriptPath(), Ops, localCfg(), newLocalOps(), rd(), seqTransport() (+28 more)

### Community 25 - "Role"
Cohesion: 0.10
Nodes (14): downloadErrTransport, fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrMatchTransport (+6 more)

### Community 26 - "load.go"
Cohesion: 0.19
Nodes (15): envTree(), TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), TestResolveEnvPath(), TestResolveEnvPathDefaultInBaseDir(), TestResolveEnvPathEmptyBaseDir(), applyBridgePortDefaults(), applyContainerBlockDefaults() (+7 more)

### Community 27 - "github.com/spf13/cobra.Command"
Cohesion: 0.08
Nodes (72): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, os.File, TestConfirmFlagShortcuts() (+64 more)

### Community 28 - "strings.Builder"
Cohesion: 0.19
Nodes (19): WeightedNodeTerm, strings.Builder, NodeAffinity, NodeMatchExpr, Placement, PodAffinityTerm, boolStr(), BrokerCR() (+11 more)

### Community 31 - "Platform"
Cohesion: 0.14
Nodes (17): keyValueEntries, TestValidateCommandAccepts(), TestValidateCommandRejects(), Platform, TestValidateUnknownPlatform(), Config, foldToEnvVar(), Config (+9 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - ".CheckEnv"
Cohesion: 0.31
Nodes (7): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing()

### Community 34 - "RenderOperator"
Cohesion: 0.60
Nodes (4): GenOperator(), RenderOperator(), watchNamespace(), operatorTmplVars

### Community 35 - ".AdditionalUsers"
Cohesion: 0.21
Nodes (9): containsAnyFold(), TestContainsAnyFold(), TestValidName(), validCLILine(), validCLIPassword(), validName(), additionalUsersScript(), TestAdditionalUsersScript() (+1 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.21
Nodes (8): NewTransport(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest(), recRunner, rrCall

### Community 38 - "emitYAML"
Cohesion: 0.28
Nodes (8): doc, boolOf(), commentSafe(), emitYAML(), kubeCommand(), redundancy(), scalar(), TestScalarQuoting()

### Community 39 - "render_test.go"
Cohesion: 0.23
Nodes (14): parseToleration(), shQuote(), envLines(), healthCheckFixture(), load(), TestAdditionalUsersReachBothHalves(), TestContainerSecretNamesAreHostScoped(), TestContainerSecretsRedundancy() (+6 more)

### Community 40 - "newEchoMgr"
Cohesion: 0.15
Nodes (14): bytes.Buffer, fileExists(), NewManager(), newEchoMgr(), TestManagerCheckDryRun(), TestManagerDeletePodmanPurgeRootless(), TestManagerDeployDockerDryRunMasksSecretEnv(), TestManagerDeployPodmanDryRunHidesSecretBytes() (+6 more)

### Community 41 - "haCfg"
Cohesion: 0.29
Nodes (9): haCfg(), TestHARoles(), TestProductKeyRoles(), TestResourceNames(), TestRestartOrder(), TestShowAll(), TestShowAllWrapsGetError(), TestCreateSecretsFailsWithoutAdminFields() (+1 more)

### Community 43 - "container/runtime_test.go"
Cohesion: 0.57
Nodes (6): TestCtrRuntimeDefaultArgvUnchanged(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), TestManagerReachableProbesRuntimeThenCompose(), withSudo(), wrappedCtrCfg()

### Community 44 - "TestClusterHonoursRuntime"
Cohesion: 0.60
Nodes (5): TestClusterHonoursRuntime(), TestRuntimeDefaultArgvUnchanged(), TestTransportHonoursRuntime(), withLeading(), wrappedCfg()

### Community 46 - "coverage_test.go"
Cohesion: 0.06
Nodes (47): matchCLI(), ranContains(), TestExecCLI(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultUsersShowVPNError(), TestDisableDefaultVPNDisableError() (+39 more)

### Community 47 - ".Run"
Cohesion: 0.11
Nodes (16): Echo, Exec, App, step(), warn(), convertPlatform(), App, newConvertCmd() (+8 more)

### Community 49 - "Convert"
Cohesion: 0.40
Nodes (5): Result, Convert(), TestConvertUnterminatedArray(), TestGeneratedHeader(), validateOutput()

### Community 50 - "Cluster"
Cohesion: 0.27
Nodes (5): Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

### Community 51 - "parsePort"
Cohesion: 0.40
Nodes (5): cut(), parsePort(), splitUser(), TestParsePort(), portSpec

## Knowledge Gaps
- **163 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `NO_COLOR` (+158 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `manager_test.go`, `context.Context`, `cli_test.go`, `Manager`, `newTestOps`, `render.go`, `NewCluster`, `convert_test.go`, `Ops`, `Cluster`, `prep_test.go`, `cliScriptPath`, `strings.Builder`, `RenderOperator`, `recRunner`, `render_test.go`, `newEchoMgr`, `haCfg`, `kubectlTransport`, `container/runtime_test.go`, `TestClusterHonoursRuntime`, `Cluster`, `.Run`, `Image`?**
  _High betweenness centrality (0.078) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `.CheckEnv`, `manager_test.go`, `cli_test.go`, `config_test.go`, `emitYAML`, `Manager`, `Config`, `newEchoMgr`, `render_test.go`, `render.go`, `convert_test.go`, `.Run`, `vars`, `Convert`, `Ops`, `load.go`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.064) - this node is a cross-community bridge._
- **Why does `Manager` connect `Manager` to `bg`, `.CheckEnv`, `manager_test.go`, `Config`, `newEchoMgr`, `.Run`, `github.com/spf13/cobra.Command`, `Platform`?**
  _High betweenness centrality (0.040) - this node is a cross-community bridge._
- **Are the 34 inferred relationships involving `newTestOps()` (e.g. with `TestAdditionalUsersRunCLITransportError()` and `TestDiagnosticsMkdirError()`) actually correct?**
  _`newTestOps()` has 34 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _163 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.07760635811126694 - nodes in this community are weakly interconnected._
- **Should `scripts.go` be split into smaller, more focused modules?**
  _Cohesion score 0.05030864197530864 - nodes in this community are weakly interconnected._