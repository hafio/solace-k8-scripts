# Graph Report - solace-k8-scripts  (2026-08-16)

## Corpus Check
- 82 files · ~174,154 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1704 nodes · 5140 edges · 61 communities (56 shown, 5 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 784 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c27a05aa`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- scripts.go
- manager_test.go
- context.Context
- cli_test.go
- testing.T
- Commands
- Manager
- config.go
- runner_test.go
- newTestOps
- capRunner
- NewCluster
- eqArgs
- convert_test.go
- Test catalogue
- newEchoMgr
- command_test.go
- NewTransport
- dev.sh
- prep_test.go
- dev.ps1
- CLAUDE.md
- judge
- verify_local_test.go
- Role
- tierFor
- github.com/spf13/cobra.Command
- render.go
- Go Module Definition
- CheckCommand
- Command reference
- manager.go
- Config
- .AdditionalUsers
- .resolveSecretRefs
- recRunner
- Platform
- Compose
- execguard_test.go
- haCfg
- kubectlTransport
- load.go
- secrets_test.go
- Cluster
- coverage_test.go
- .Run
- cliScriptPath
- EnvPairs
- Cluster
- scripts_test.go
- .releaseToBackup
- .LeaderLocal
- containerTransport
- ranContains
- container/preflight_test.go
- .Login
- .ServerCert
- .Preflight

## God Nodes (most connected - your core abstractions)
1. `Commands` - 115 edges
2. `ctrCfg()` - 82 edges
3. `Role` - 81 edges
4. `Config` - 76 edges
5. `newCapMgr()` - 74 edges
6. `newTestOps()` - 73 edges
7. `bg()` - 70 edges
8. `Manager` - 54 edges
9. `matchCLI()` - 52 edges
10. `Platform` - 51 edges

## Surprising Connections (you probably didn't know these)
- `TestFieldLabelWithoutColon()` --calls--> `field()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker.go
- `activity()` --calls--> `countContains()`  [INFERRED]
  internal/broker/verify_ops.go → internal/broker/broker.go
- `matchCLI()` --calls--> `cliArg()`  [INFERRED]
  internal/broker/broker_test.go → internal/broker/transport.go
- `TestAdditionalUsersRunCLITransportError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestDisableDefaultVPNShowError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (61 total, 5 thin omitted)

### Community 0 - "bg"
Cohesion: 0.06
Nodes (99): time.Duration, Ops, New(), TestNewDefaults(), Transport, TestConfirmFlagShortcuts(), TestCtrManagerConfirmWiring(), confirmDelete() (+91 more)

### Community 1 - "scripts.go"
Cohesion: 0.21
Nodes (13): showCmd, disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), revertActivityConfigureScript(), revertActivityScript(), showRedundancyDetailScript(), showRedundancyScript() (+5 more)

### Community 2 - "manager_test.go"
Cohesion: 0.10
Nodes (69): assertMode(), containsStr(), ctrCfg(), hasCall(), maskedKeys(), newCapMgr(), TestContainerRunningMatchesNameExactly(), TestManagerCheckDNSFailsLoudInHA() (+61 more)

### Community 3 - "context.Context"
Cohesion: 0.12
Nodes (12): context.Context, Cluster, HARoles(), lbServiceName(), podName(), pvcName(), RestartOrder(), stsName() (+4 more)

### Community 4 - "cli_test.go"
Cohesion: 0.05
Nodes (95): opCall, opRunner, TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandIsRegisteredOnPlatformTrees(), TestAllowCommandIsRepeatable(), TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), TestEscalationIsRefusedEndToEnd() (+87 more)

### Community 5 - "testing.T"
Cohesion: 0.06
Nodes (80): testing.T, assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, haNodesConfig(), minimalK8s(), TestApplyDefaultsDocker() (+72 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (115): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+107 more)

### Community 8 - "config.go"
Cohesion: 0.11
Nodes (22): Container, ContainerSecurity, DockerConfig, DomainCerts, Network, Node, Nodes, Operator (+14 more)

### Community 9 - "runner_test.go"
Cohesion: 0.12
Nodes (26): bytes.Buffer, captureResolve(), TestExecEchoesOnEveryMethod(), TestExecEchoesResolvedPath(), TestExecMissingBinaryIsActionable(), TestExecRefusesCurrentDirectoryResolution(), captureStdout(), helperCommand() (+18 more)

### Community 10 - "newTestOps"
Cohesion: 0.13
Nodes (28): appUsers(), Ops, newTestOps(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDisableDefaultUsersNoVPNs(), TestDomainCertsEmptySkips() (+20 more)

### Community 11 - "capRunner"
Cohesion: 0.15
Nodes (20): capCall, capRunner, Manager, mgrOver(), TestCtrExecutorRefusesUnapprovedRuntime(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), TestManagerReachableProbesRuntimeThenCompose() (+12 more)

### Community 12 - "NewCluster"
Cohesion: 0.13
Nodes (32): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster() (+24 more)

### Community 13 - "eqArgs"
Cohesion: 0.15
Nodes (26): Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived(), TestOperatorNSExplicit() (+18 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (54): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+46 more)

### Community 15 - "Test catalogue"
Cohesion: 0.04
Nodes (48): allowcommand_test.go, broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go (+40 more)

### Community 16 - "newEchoMgr"
Cohesion: 0.10
Nodes (22): fileExists(), NewManager(), Manager, newEchoMgr(), rootlessNoFileMgr(), setNoFile(), TestManagerCheckDryRun(), TestManagerDeletePodmanPurgeRootless() (+14 more)

### Community 17 - "command_test.go"
Cohesion: 0.15
Nodes (14): decodeRuntime(), TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults() (+6 more)

### Community 18 - "NewTransport"
Cohesion: 0.10
Nodes (16): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels() (+8 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "prep_test.go"
Cohesion: 0.22
Nodes (19): saCfg(), adminCfg(), Cluster, labelCluster(), TestCreateSecretsAdminOnly(), TestCreateSecretsPreflight(), TestCreateSecretsStopsOnPreflightFailure(), TestDeleteSecrets() (+11 more)

### Community 21 - "dev.ps1"
Cohesion: 0.19
Nodes (16): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+8 more)

### Community 22 - "CLAUDE.md"
Cohesion: 0.12
Nodes (11): bash/000-env.sh, bash/010-deploy-operator.sh, bash/020-deploy-broker.sh, bash/059-execute-cli.sh, docker-podman/000-env.sh, internal/broker, internal/cli, internal/config (+3 more)

### Community 23 - "judge"
Cohesion: 0.16
Nodes (21): describe(), judge(), main(), plural(), run(), load(), TestJudge(), TestJudgeEmptyTrace() (+13 more)

### Community 24 - "verify_local_test.go"
Cohesion: 0.15
Nodes (34): uploadedForRole(), Ops, localCfg(), newLocalOps(), rd(), seqTransport(), TestLeaderLocalAssertLeaderError(), TestLeaderLocalBadRoleArg() (+26 more)

### Community 25 - "Role"
Cohesion: 0.14
Nodes (11): downloadErrTransport, fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrMatchTransport (+3 more)

### Community 26 - "tierFor"
Cohesion: 0.27
Nodes (7): scalingTier, containerMem(), Config, TestContainerMem(), TestScalingTiers(), TestTierForRejectsOffTierValues(), tierFor()

### Community 27 - "github.com/spf13/cobra.Command"
Cohesion: 0.07
Nodes (78): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, os.File, TestFirstArg() (+70 more)

### Community 28 - "render.go"
Cohesion: 0.20
Nodes (23): strings.Builder, PodAffinityTerm, boolStr(), BrokerCR(), cut(), parsePort(), parseToleration(), sortedKeys() (+15 more)

### Community 31 - "CheckCommand"
Cohesion: 0.18
Nodes (15): commandRules, checkBinary(), CheckCommand(), checkFlagShape(), checkToken(), clusterRules(), composeRules(), execBase() (+7 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "manager.go"
Cohesion: 0.21
Nodes (9): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), splitLimit() (+1 more)

### Community 34 - "Config"
Cohesion: 0.18
Nodes (10): Image, Replication, Scaling, TLS, atoiPrefix(), Config, GenOperator(), RenderOperator() (+2 more)

### Community 35 - ".AdditionalUsers"
Cohesion: 0.14
Nodes (14): Admin, containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), validCLILine(), validCLIPassword(), Ops (+6 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.18
Nodes (8): TestCanIAnswerReadsTheLastLine(), isCanI(), TestTransportCopy(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest(), recRunner, rrCall

### Community 38 - "Platform"
Cohesion: 0.20
Nodes (13): keyValueEntries, Platform, TestValidateUnknownPlatform(), foldToEnvVar(), Config, missingErr(), platformKey(), requireAll() (+5 more)

### Community 39 - "Compose"
Cohesion: 0.14
Nodes (29): emitCtrArtifact(), Compose(), ContainerSecrets(), escapePercent(), ContainerSecret, healthCmd(), Quadlet(), quadletEscape() (+21 more)

### Community 40 - "execguard_test.go"
Cohesion: 0.19
Nodes (15): decodeStrict(), Config, guardCommandOf(), guardConfig(), setGuardCommand(), TestAllowCommandIsNotASchemaKey(), TestAllowCommandsAccepts(), TestAllowCommandsRejects() (+7 more)

### Community 41 - "haCfg"
Cohesion: 0.18
Nodes (16): haCfg(), TestHARoles(), TestProductKeyRoles(), TestResourceNames(), TestRestartOrder(), TestShowAll(), TestShowAllWrapsGetError(), TestCreateSecretsFailsWithoutAdminFields() (+8 more)

### Community 43 - "load.go"
Cohesion: 0.19
Nodes (12): TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config, parseError() (+4 more)

### Community 44 - "secrets_test.go"
Cohesion: 0.15
Nodes (18): TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden(), decodeDataValue(), TestAdminSecretDecodes() (+10 more)

### Community 45 - "Cluster"
Cohesion: 0.12
Nodes (11): TestWarnAndStep(), App, step(), warn(), convertPlatform(), App, newConvertCmd(), runConvert() (+3 more)

### Community 46 - "coverage_test.go"
Cohesion: 0.06
Nodes (47): matchCLI(), writeFile(), TestDiagnosticsMkdirError(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultUsersShowVPNError(), TestDisableDefaultVPNDisableError() (+39 more)

### Community 47 - ".Run"
Cohesion: 0.18
Nodes (11): Echo, Exec, os/exec.Cmd, TestChildEnvNamesAreNotSystemVariables(), command(), MaskEnv(), Quote(), quoteTok() (+3 more)

### Community 48 - "cliScriptPath"
Cohesion: 0.19
Nodes (15): TestAdditionalUsers(), TestDiagnostics(), TestDisableDefaultUsers(), TestDisableDefaultVPN(), TestDomainCerts(), TestPathHelpers(), TestProductKeys(), TestRemoveDomainCerts() (+7 more)

### Community 49 - "EnvPairs"
Cohesion: 0.18
Nodes (10): Config, NodeIdentity, containerSecretSpecs(), EnvFile(), EnvPairs(), groupKey(), itoa(), secretFilePath() (+2 more)

### Community 50 - "Cluster"
Cohesion: 0.27
Nodes (5): Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

### Community 51 - "scripts_test.go"
Cohesion: 0.16
Nodes (13): disableDefaultUsersScript(), domainCertsScript(), gatherConfigsScript(), parseVPNNames(), sortedKeys(), TestDisableDefaultUsersScriptQuoting(), TestDomainCertsScriptSorted(), TestGatherConfigsScript() (+5 more)

### Community 52 - ".releaseToBackup"
Cohesion: 0.24
Nodes (9): field(), TestField(), TestLastLines(), TestPrimaryRedundancyUp(), activity(), Ops, lastLines(), primaryRedundancyUp() (+1 more)

### Community 53 - ".LeaderLocal"
Cohesion: 0.23
Nodes (7): assertLeaderScript(), TestAssertLeaderScript(), Ops, hostMatches(), roleName(), shortHost(), TestRoleName()

### Community 55 - "ranContains"
Cohesion: 0.40
Nodes (5): ranContains(), TestDisableDefaultVPNShowError(), TestExecCLIRunError(), TestRemoveCLIWarnsOnFailure(), TestRemoveDomainCertsRunCLIError()

### Community 56 - "container/preflight_test.go"
Cohesion: 0.40
Nodes (4): TestComposeSecretEnvIsTheOnlyChildEnvironment(), TestComposeSecretEnvNamesCannotBeSystemVars(), TestPreflightIsPreviewableUnderDryRun(), TestPreflightRunsBeforeAnything()

### Community 57 - ".Login"
Cohesion: 0.67
Nodes (3): TestHTTPStatusHelpers(), httpStatusLines(), isHTTP2xx()

### Community 61 - ".ServerCert"
Cohesion: 0.47
Nodes (4): concatFiles(), serverCertFile(), serverCertScript(), TestServerCertScript()

## Knowledge Gaps
- **170 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+165 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `manager_test.go`, `context.Context`, `cli_test.go`, `Manager`, `config.go`, `newTestOps`, `capRunner`, `NewCluster`, `convert_test.go`, `newEchoMgr`, `NewTransport`, `prep_test.go`, `verify_local_test.go`, `render.go`, `.AdditionalUsers`, `Compose`, `haCfg`, `kubectlTransport`, `secrets_test.go`, `Cluster`, `EnvPairs`, `containerTransport`?**
  _High betweenness centrality (0.076) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `manager.go`, `manager_test.go`, `cli_test.go`, `testing.T`, `Manager`, `config.go`, `execguard_test.go`, `Compose`, `load.go`, `capRunner`, `Cluster`, `convert_test.go`, `newEchoMgr`, `containerTransport`, `tierFor`, `github.com/spf13/cobra.Command`, `CheckCommand`?**
  _High betweenness centrality (0.069) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `bg`, `scripts.go`, `.AdditionalUsers`, `context.Context`, `Manager`, `Compose`, `kubectlTransport`, `cliScriptPath`, `EnvPairs`, `NewTransport`, `scripts_test.go`, `.releaseToBackup`, `.LeaderLocal`, `containerTransport`, `verify_local_test.go`, `.Login`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _170 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06167176350662589 - nodes in this community are weakly interconnected._
- **Should `manager_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09979296066252588 - nodes in this community are weakly interconnected._