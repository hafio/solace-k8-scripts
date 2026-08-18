# Graph Report - solace-k8-scripts  (2026-08-17)

## Corpus Check
- 84 files · ~180,768 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1744 nodes · 5259 edges · 65 communities (60 shown, 5 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 803 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6f1ae9f3`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- cli_test.go
- manager_test.go
- Cluster
- captureStdout
- testing.T
- Commands
- Manager
- Config
- runner_test.go
- newTestOps
- capRunner
- NewCluster
- eqArgs
- convert_test.go
- internal/k8s
- newEchoMgr
- command_test.go
- Cluster
- dev.sh
- prep_test.go
- dev.ps1
- CLAUDE.md
- judge
- cliScriptPath
- Role
- tierFor
- github.com/spf13/cobra.Command
- strings.Builder
- Go Module Definition
- CheckCommand
- Command reference
- manager.go
- runRoot
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
- context.Context
- coverage_test.go
- .Run
- .Run
- render.go
- Cluster
- newRootCmd
- scripts.go
- completion_test.go
- containerTransport
- Test catalogue
- container/preflight_test.go
- internal/cli
- internal/broker
- internal/config
- internal/container
- Fixtures and doubles
- ContainerSecret
- internal/engine

## God Nodes (most connected - your core abstractions)
1. `Commands` - 120 edges
2. `ctrCfg()` - 82 edges
3. `Role` - 81 edges
4. `Config` - 77 edges
5. `newCapMgr()` - 74 edges
6. `newTestOps()` - 73 edges
7. `bg()` - 70 edges
8. `Manager` - 54 edges
9. `matchCLI()` - 52 edges
10. `Platform` - 51 edges

## Surprising Connections (you probably didn't know these)
- `activity()` --calls--> `countContains()`  [INFERRED]
  internal/broker/verify_ops.go → internal/broker/broker.go
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

## Communities (65 total, 5 thin omitted)

### Community 0 - "bg"
Cohesion: 0.07
Nodes (93): time.Duration, Ops, TestCtrManagerConfirmWiring(), containerWhat(), ctrLogin(), ctrManager(), ctrOps(), App (+85 more)

### Community 1 - "cli_test.go"
Cohesion: 0.15
Nodes (28): capture(), captureStderr(), runCtr(), runStatusStderr(), TestBashEnvGivenToEnvFlag(), TestConvertErrorPaths(), TestConvertRoundTrip(), TestConvertToFile() (+20 more)

### Community 2 - "manager_test.go"
Cohesion: 0.09
Nodes (76): fileExists(), NewManager(), assertMode(), containsStr(), ctrCfg(), hasCall(), maskedKeys(), newCapMgr() (+68 more)

### Community 3 - "Cluster"
Cohesion: 0.12
Nodes (11): Cluster, HARoles(), lbServiceName(), podName(), pvcName(), RestartOrder(), stsName(), TestResourceNames() (+3 more)

### Community 4 - "captureStdout"
Cohesion: 0.14
Nodes (23): opCall, opRunner, captureStdout(), failDisableDefaultUsersUpload(), k8sConfigAllRunner(), k8sUpOutputHook(), loadDirect(), opArgvMatch() (+15 more)

### Community 5 - "testing.T"
Cohesion: 0.07
Nodes (78): testing.T, assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, haNodesConfig(), minimalK8s(), TestApplyDefaultsDocker() (+70 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (120): Commands, solace, solace completion, solace completion bash, solace completion fish, solace completion powershell, solace completion zsh, solace convert (+112 more)

### Community 8 - "Config"
Cohesion: 0.10
Nodes (25): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, Network, Node (+17 more)

### Community 9 - "runner_test.go"
Cohesion: 0.12
Nodes (26): bytes.Buffer, captureResolve(), TestExecEchoesOnEveryMethod(), TestExecEchoesResolvedPath(), TestExecMissingBinaryIsActionable(), TestExecRefusesCurrentDirectoryResolution(), captureStdout(), helperCommand() (+18 more)

### Community 10 - "newTestOps"
Cohesion: 0.11
Nodes (37): Ops, newTestOps(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestDiagnostics(), TestDisableDefaultUsers(), TestDisableDefaultUsersNoVPNs(), TestDisableDefaultVPN() (+29 more)

### Community 11 - "capRunner"
Cohesion: 0.15
Nodes (20): capCall, capRunner, Manager, mgrOver(), TestCtrExecutorRefusesUnapprovedRuntime(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), TestManagerReachableProbesRuntimeThenCompose() (+12 more)

### Community 12 - "NewCluster"
Cohesion: 0.13
Nodes (33): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckOperatorNS(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass() (+25 more)

### Community 13 - "eqArgs"
Cohesion: 0.14
Nodes (27): Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived(), TestOperatorNSExplicit() (+19 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (55): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+47 more)

### Community 15 - "internal/k8s"
Cohesion: 0.17
Nodes (12): check_test.go, cluster_test.go, deploy_test.go, internal/k8s, names_test.go, operator_test.go, ops_test.go, preflight_test.go (+4 more)

### Community 16 - "newEchoMgr"
Cohesion: 0.12
Nodes (18): Manager, newEchoMgr(), rootlessNoFileMgr(), setNoFile(), TestManagerCheckDryRun(), TestManagerDeletePodmanPurgeRootless(), TestManagerDeployPodmanDryRunHidesSecretBytes(), TestManagerDockerCheckProbesCompose() (+10 more)

### Community 17 - "command_test.go"
Cohesion: 0.15
Nodes (14): decodeRuntime(), TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults() (+6 more)

### Community 18 - "Cluster"
Cohesion: 0.13
Nodes (11): bufio.Reader, Cluster, isBuiltinLabel(), namespaceManifest(), roleName(), rolePlacementLabels(), splitLabel(), TestIsBuiltinLabel() (+3 more)

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

### Community 24 - "cliScriptPath"
Cohesion: 0.15
Nodes (36): uploadedForRole(), TestReleaseToBackupReleasedTimeout(), cliScriptPath(), Ops, localCfg(), newLocalOps(), rd(), seqTransport() (+28 more)

### Community 25 - "Role"
Cohesion: 0.13
Nodes (12): downloadErrTransport, fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrMatchTransport (+4 more)

### Community 26 - "tierFor"
Cohesion: 0.27
Nodes (7): scalingTier, containerMem(), Config, TestContainerMem(), TestScalingTiers(), TestTierForRejectsOffTierValues(), tierFor()

### Community 27 - "github.com/spf13/cobra.Command"
Cohesion: 0.06
Nodes (93): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/cobra.ShellCompDirective, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, os.File (+85 more)

### Community 28 - "strings.Builder"
Cohesion: 0.18
Nodes (18): WeightedNodeTerm, strings.Builder, NodeAffinity, NodeMatchExpr, Placement, PodAffinityTerm, parseToleration(), sortedKeys() (+10 more)

### Community 31 - "CheckCommand"
Cohesion: 0.18
Nodes (15): commandRules, checkBinary(), CheckCommand(), checkFlagShape(), checkToken(), clusterRules(), composeRules(), execBase() (+7 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "manager.go"
Cohesion: 0.22
Nodes (9): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), splitLimit() (+1 more)

### Community 34 - "runRoot"
Cohesion: 0.16
Nodes (19): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), TestEscalationIsRefusedEndToEnd(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb(), TestPathRuntimeIsRefused(), TestSmuggledSubcommandIsRefused() (+11 more)

### Community 35 - ".AdditionalUsers"
Cohesion: 0.09
Nodes (23): containsAnyFold(), countContains(), New(), appUsers(), TestAdditionalUsers(), TestAdditionalUsersReportsExistingUser(), TestContainsAnyFold(), TestCountContains() (+15 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.17
Nodes (10): TestCanIAnswerReadsTheLastLine(), NewTransport(), isCanI(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest() (+2 more)

### Community 38 - "Platform"
Cohesion: 0.20
Nodes (13): keyValueEntries, Platform, TestValidateUnknownPlatform(), foldToEnvVar(), Config, missingErr(), platformKey(), requireAll() (+5 more)

### Community 39 - "Compose"
Cohesion: 0.20
Nodes (23): emitCtrArtifact(), BrokerCR(), Compose(), ContainerSecrets(), SecretPreflight(), SecretScript(), shQuote(), envLines() (+15 more)

### Community 40 - "execguard_test.go"
Cohesion: 0.19
Nodes (15): decodeStrict(), Config, guardCommandOf(), guardConfig(), setGuardCommand(), TestAllowCommandIsNotASchemaKey(), TestAllowCommandsAccepts(), TestAllowCommandsRejects() (+7 more)

### Community 41 - "haCfg"
Cohesion: 0.21
Nodes (14): haCfg(), TestHARoles(), TestProductKeyRoles(), TestRestartOrder(), TestCopyFrom(), TestCreateSecretsFailsWithoutAdminFields(), TestDeleteSecretsSkipsUnconfiguredAdminSecret(), TestClusterHonoursRuntime() (+6 more)

### Community 43 - "load.go"
Cohesion: 0.19
Nodes (12): TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config, parseError() (+4 more)

### Community 44 - "secrets_test.go"
Cohesion: 0.10
Nodes (27): GenOperator(), operatorImage(), RenderOperator(), TestOperatorImage(), watchNamespace(), GenSecrets(), joinManifests(), TestCreateSecretsAllThree() (+19 more)

### Community 45 - "context.Context"
Cohesion: 0.16
Nodes (5): context.Context, Cluster, Cluster, canIAnswer(), Cluster

### Community 46 - "coverage_test.go"
Cohesion: 0.07
Nodes (45): matchCLI(), ranContains(), TestExecCLI(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultUsersShowVPNError(), TestDisableDefaultVPNDisableError() (+37 more)

### Community 47 - ".Run"
Cohesion: 0.18
Nodes (11): Echo, Exec, os/exec.Cmd, TestChildEnvNamesAreNotSystemVariables(), command(), MaskEnv(), Quote(), quoteTok() (+3 more)

### Community 48 - ".Run"
Cohesion: 0.19
Nodes (18): firstLine(), App, runRootWith(), runStandalone(), TestErrorPaths(), TestFirstArgOr(), TestGenWired(), TestK8sConfirmDeclined() (+10 more)

### Community 49 - "render.go"
Cohesion: 0.15
Nodes (23): NodeIdentity, boolStr(), containerSecretSpecs(), cut(), EnvFile(), EnvPairs(), escapePercent(), groupKey() (+15 more)

### Community 50 - "Cluster"
Cohesion: 0.26
Nodes (5): Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

### Community 51 - "newRootCmd"
Cohesion: 0.16
Nodes (14): TestAllowCommandIsRegisteredOnPlatformTrees(), TestAllowCommandIsRepeatable(), collectPaths(), findCmd(), TestExecute(), TestFlagsRegistered(), TestNotImplemented(), TestTreeStructure() (+6 more)

### Community 52 - "scripts.go"
Cohesion: 0.05
Nodes (53): showCmd, field(), TestField(), TestHTTPStatusHelpers(), TestLastLines(), TestPrimaryRedundancyUp(), concatFiles(), Ops (+45 more)

### Community 53 - "completion_test.go"
Cohesion: 0.27
Nodes (15): runComplete(), TestAllowCommandOffersNoFiles(), TestCompletionHelpStillWorks(), TestCompletionNeedsAShell(), TestCompletionNoDescriptions(), TestCompletionScriptsGenerate(), TestDirFlagCompletesDirectories(), TestEnvFlagCompletesEnvFiles() (+7 more)

### Community 55 - "Test catalogue"
Cohesion: 0.18
Nodes (10): convert_test.go, Coverage, internal/convert, internal/render, internal/tools/vulnjudge, main_test.go, render_test.go, Running the tests (+2 more)

### Community 56 - "container/preflight_test.go"
Cohesion: 0.40
Nodes (4): TestComposeSecretEnvIsTheOnlyChildEnvironment(), TestComposeSecretEnvNamesCannotBeSystemVars(), TestPreflightIsPreviewableUnderDryRun(), TestPreflightRunsBeforeAnything()

### Community 57 - "internal/cli"
Cohesion: 0.40
Nodes (5): allowcommand_test.go, cli_test.go, commanddoc_test.go, completion_test.go, internal/cli

### Community 58 - "internal/broker"
Cohesion: 0.40
Nodes (5): broker_test.go, coverage_test.go, internal/broker, scripts_test.go, verify_local_test.go

### Community 59 - "internal/config"
Cohesion: 0.40
Nodes (5): command_test.go, config_test.go, execguard_test.go, internal/config, scaling_test.go

### Community 60 - "internal/container"
Cohesion: 0.40
Nodes (5): internal/container, manager_test.go, preflight_test.go, runtime_test.go, transport_test.go

### Community 61 - "Fixtures and doubles"
Cohesion: 0.50
Nodes (4): Fixtures and doubles, Injectable seams, Per-package doubles, Shared env fixtures

### Community 63 - "internal/engine"
Cohesion: 0.67
Nodes (3): internal/engine, resolve_test.go, runner_test.go

## Knowledge Gaps
- **176 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+171 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `manager_test.go`, `Cluster`, `captureStdout`, `Manager`, `newTestOps`, `capRunner`, `NewCluster`, `convert_test.go`, `newEchoMgr`, `Cluster`, `prep_test.go`, `cliScriptPath`, `github.com/spf13/cobra.Command`, `strings.Builder`, `.AdditionalUsers`, `recRunner`, `Compose`, `haCfg`, `kubectlTransport`, `secrets_test.go`, `context.Context`, `render.go`, `containerTransport`?**
  _High betweenness centrality (0.066) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `manager.go`, `manager_test.go`, `captureStdout`, `testing.T`, `Manager`, `Config`, `execguard_test.go`, `Compose`, `load.go`, `capRunner`, `convert_test.go`, `newEchoMgr`, `containerTransport`, `tierFor`, `github.com/spf13/cobra.Command`, `CheckCommand`?**
  _High betweenness centrality (0.064) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `bg`, `.AdditionalUsers`, `Cluster`, `Manager`, `Compose`, `kubectlTransport`, `Cluster`, `scripts.go`, `containerTransport`, `cliScriptPath`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.036) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _176 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06681896059394632 - nodes in this community are weakly interconnected._
- **Should `manager_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08680792891319207 - nodes in this community are weakly interconnected._