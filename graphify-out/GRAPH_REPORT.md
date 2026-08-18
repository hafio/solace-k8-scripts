# Graph Report - solace-k8-scripts  (2026-08-18)

## Corpus Check
- cluster-only mode — file stats not available

## Summary
- 1760 nodes · 5322 edges · 71 communities (65 shown, 6 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 812 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6f1ae9f3`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- cli_test.go
- ctrCfg
- Cluster
- captureStdout
- config_test.go
- Commands
- Manager
- config.go
- testing.T
- newTestOps
- capRunner
- NewCluster
- eqArgs
- convert_test.go
- internal/k8s
- manager_test.go
- Command
- Cluster
- dev.sh
- prep_test.go
- dev.ps1
- CLAUDE.md
- judge
- verify_local_test.go
- Role
- tierFor
- github.com/spf13/cobra.Command
- strings.Builder
- solace
- CheckCommand
- Command reference
- manager.go
- completion_test.go
- cliScriptPath
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
- hasCall
- render.go
- scripts.go
- newRootCmd
- scripts_test.go
- Load
- containerTransport
- Test catalogue
- .LeaderLocal
- internal/cli
- internal/broker
- internal/config
- internal/container
- Fixtures and doubles
- NewManager
- internal/engine
- Config
- .redundancyLocalPrimary
- .releaseToBackup
- Ops
- .ServerCert
- .Preflight

## God Nodes (most connected - your core abstractions)
1. `Commands` - 121 edges
2. `ctrCfg()` - 82 edges
3. `Role` - 81 edges
4. `Config` - 79 edges
5. `newCapMgr()` - 74 edges
6. `newTestOps()` - 73 edges
7. `bg()` - 70 edges
8. `Manager` - 54 edges
9. `matchCLI()` - 52 edges
10. `NewCluster()` - 51 edges

## Surprising Connections (you probably didn't know these)
- `TestCtrLoginOutcomes()` --calls--> `ctrLogin()`  [INFERRED]
  internal/cli/cli_test.go → internal/cli/ops_container.go
- `ctrManager()` --calls--> `confirmRestart()`  [INFERRED]
  internal/cli/ops_container.go → internal/cli/helpers.go
- `TestCtrLoginOutcomes()` --calls--> `ctrOps()`  [INFERRED]
  internal/cli/cli_test.go → internal/cli/ops_container.go
- `TestOpCtrConfigAllAborts()` --calls--> `opCtrConfigAll()`  [INFERRED]
  internal/cli/cli_test.go → internal/cli/ops_container.go
- `newCtrConfigCmd()` --calls--> `opCtrConfigAll()`  [INFERRED]
  internal/cli/container.go → internal/cli/ops_container.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (71 total, 6 thin omitted)

### Community 0 - "bg"
Cohesion: 0.07
Nodes (93): time.Duration, Ops, TestCtrManagerConfirmWiring(), containerWhat(), ctrLogin(), ctrManager(), ctrOps(), App (+85 more)

### Community 1 - "cli_test.go"
Cohesion: 0.08
Nodes (66): allowRuntime(), capture(), captureStderr(), fakeBinaryOnPath(), firstLine(), App, runCtr(), runRoot() (+58 more)

### Community 2 - "ctrCfg"
Cohesion: 0.12
Nodes (39): ctrCfg(), newCapMgr(), TestContainerRunningMatchesNameExactly(), TestManagerCheckDNSFailsLoudInHA(), TestManagerCheckPodmanEUID(), TestManagerCheckReachableError(), TestManagerCheckStandaloneDNSWarnsOnly(), TestManagerCopy() (+31 more)

### Community 3 - "Cluster"
Cohesion: 0.13
Nodes (10): Cluster, HARoles(), lbServiceName(), podName(), pvcName(), stsName(), TestResourceNames(), filterLines() (+2 more)

### Community 4 - "captureStdout"
Cohesion: 0.13
Nodes (24): opCall, opRunner, captureStdout(), failDisableDefaultUsersUpload(), k8sConfigAllRunner(), k8sUpOutputHook(), loadDirect(), opArgvMatch() (+16 more)

### Community 5 - "config_test.go"
Cohesion: 0.05
Nodes (65): assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, haNodesConfig(), TestApplyBridgePortDefaults(), TestApplyDefaultsDocker(), TestApplyDefaultsK8s() (+57 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (121): Commands, solace-util, solace-util completion, solace-util completion bash, solace-util completion fish, solace-util completion powershell, solace-util completion zsh, solace-util convert (+113 more)

### Community 8 - "config.go"
Cohesion: 0.09
Nodes (28): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, Network, Node (+20 more)

### Community 9 - "testing.T"
Cohesion: 0.12
Nodes (34): testing.T, TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults(), TestRuntimeExplicitValueSurvivesDefaults() (+26 more)

### Community 10 - "newTestOps"
Cohesion: 0.11
Nodes (35): appUsers(), Ops, newTestOps(), TestAdditionalUsers(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDiagnostics() (+27 more)

### Community 11 - "capRunner"
Cohesion: 0.18
Nodes (15): capCall, capRunner, New(), TestNewDefaults(), Transport, TestManagerLogsCLIShell(), NewTransport(), dockerCfg() (+7 more)

### Community 12 - "NewCluster"
Cohesion: 0.13
Nodes (32): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckOperatorNS(), TestCheckStorageClass(), TestResolveStorageClass(), NewCluster() (+24 more)

### Community 13 - "eqArgs"
Cohesion: 0.17
Nodes (25): TestReachable(), Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived() (+17 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (55): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+47 more)

### Community 15 - "internal/k8s"
Cohesion: 0.17
Nodes (12): check_test.go, cluster_test.go, deploy_test.go, internal/k8s, names_test.go, operator_test.go, ops_test.go, preflight_test.go (+4 more)

### Community 16 - "manager_test.go"
Cohesion: 0.10
Nodes (35): bytes.Buffer, defaultGenPSK(), fileExists(), assertMode(), Manager, maskedKeys(), newEchoMgr(), rootlessNoFileMgr() (+27 more)

### Community 17 - "Command"
Cohesion: 0.20
Nodes (5): decodeRuntime(), TestCommandUnmarshal(), Command, TestCharsetAgreesAcrossBothYAMLForms(), yaml.Node

### Community 18 - "Cluster"
Cohesion: 0.12
Nodes (14): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels() (+6 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "prep_test.go"
Cohesion: 0.19
Nodes (21): saCfg(), adminCfg(), Cluster, labelCluster(), TestCreateNamespaceApplyFails(), TestCreateSecretsAdminOnly(), TestCreateSecretsPreflight(), TestCreateSecretsStopsOnPreflightFailure() (+13 more)

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
Cohesion: 0.06
Nodes (91): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/cobra.ShellCompDirective, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, os.File (+83 more)

### Community 28 - "strings.Builder"
Cohesion: 0.20
Nodes (16): strings.Builder, PodAffinityTerm, cut(), parseToleration(), sortedKeys(), splitUser(), TestParseToleration(), writeKeyValueEntry() (+8 more)

### Community 31 - "CheckCommand"
Cohesion: 0.19
Nodes (15): commandRules, checkBinary(), CheckCommand(), checkFlagShape(), checkToken(), clusterRules(), composeRules(), execBase() (+7 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "manager.go"
Cohesion: 0.19
Nodes (10): exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), splitLimit(), TestReplacePSKLine() (+2 more)

### Community 34 - "completion_test.go"
Cohesion: 0.30
Nodes (14): runComplete(), TestAllowCommandOffersNoFiles(), TestCompletionNeedsAShell(), TestCompletionNoDescriptions(), TestCompletionScriptsGenerate(), TestDirFlagCompletesDirectories(), TestEnvFlagCompletesEnvFiles(), TestEnvFlagPrefixFilters() (+6 more)

### Community 35 - "cliScriptPath"
Cohesion: 0.13
Nodes (16): containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), TestPathHelpers(), TestServerCert(), TestValidName(), validCLILine() (+8 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.17
Nodes (10): TestCanIAnswerReadsTheLastLine(), NewTransport(), isCanI(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest() (+2 more)

### Community 38 - "Platform"
Cohesion: 0.24
Nodes (10): keyValueEntries, Platform, foldToEnvVar(), Config, missingErr(), platformKey(), requireAll(), requireKeyValue() (+2 more)

### Community 39 - "Compose"
Cohesion: 0.19
Nodes (22): boolStr(), BrokerCR(), Compose(), parsePort(), envLines(), healthCheckFixture(), load(), TestAdditionalUsersReachBothHalves() (+14 more)

### Community 40 - "execguard_test.go"
Cohesion: 0.19
Nodes (15): decodeStrict(), Config, guardCommandOf(), guardConfig(), setGuardCommand(), TestAllowCommandIsNotASchemaKey(), TestAllowCommandsAccepts(), TestAllowCommandsRejects() (+7 more)

### Community 41 - "haCfg"
Cohesion: 0.18
Nodes (16): RestartOrder(), haCfg(), TestHARoles(), TestProductKeyRoles(), TestRestartOrder(), TestShowAll(), TestShowAllWrapsGetError(), TestCreateSecretsFailsWithoutAdminFields() (+8 more)

### Community 43 - "load.go"
Cohesion: 0.23
Nodes (10): applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config, parseError(), setDefault(), setDefaultCmd() (+2 more)

### Community 44 - "secrets_test.go"
Cohesion: 0.15
Nodes (19): TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), DockerRegistrySecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden(), decodeDataValue() (+11 more)

### Community 45 - "context.Context"
Cohesion: 0.16
Nodes (4): context.Context, Cluster, Cluster, Cluster

### Community 46 - "coverage_test.go"
Cohesion: 0.06
Nodes (49): matchCLI(), ranContains(), TestExecCLI(), writeFile(), TestDiagnosticsMkdirError(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError() (+41 more)

### Community 47 - ".Run"
Cohesion: 0.12
Nodes (18): Echo, Exec, os/exec.Cmd, App, step(), TestChildEnvNamesAreNotSystemVariables(), TestExecEchoesOnEveryMethod(), TestExecIsSilentWithoutVerbose() (+10 more)

### Community 48 - "hasCall"
Cohesion: 0.13
Nodes (20): containsStr(), hasCall(), TestManagerDeleteDockerComposeDownWhenFileExists(), TestManagerDeleteDockerComposeNoFileFallsBackToStopRm(), TestManagerDeleteDockerPurgeRemovesDataDir(), TestManagerDeployPodmanWritesUnit(), TestManagerDescribe(), TestManagerDockerComposeCommandOverride() (+12 more)

### Community 49 - "render.go"
Cohesion: 0.14
Nodes (22): emitCtrArtifact(), Config, NodeIdentity, ContainerSecrets(), containerSecretSpecs(), EnvFile(), EnvPairs(), escapePercent() (+14 more)

### Community 50 - "scripts.go"
Cohesion: 0.21
Nodes (13): showCmd, disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), revertActivityConfigureScript(), revertActivityScript(), showRedundancyDetailScript(), showRedundancyScript() (+5 more)

### Community 51 - "newRootCmd"
Cohesion: 0.12
Nodes (22): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandIsRegisteredOnPlatformTrees(), TestAllowCommandIsRepeatable(), TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), TestEscalationIsRefusedEndToEnd(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb() (+14 more)

### Community 52 - "scripts_test.go"
Cohesion: 0.14
Nodes (14): additionalUsersScript(), assertLeaderScript(), disableDefaultUsersScript(), parseVPNNames(), productKeysScript(), TestAdditionalUsersScript(), TestAssertLeaderScript(), TestDisableDefaultUsersScriptQuoting() (+6 more)

### Community 53 - "Load"
Cohesion: 0.24
Nodes (15): minimalK8s(), TestLoadBashEnvFileHint(), TestLoadNotYAMLHint(), TestLoadParseError(), TestLoadReadError(), TestLoadResolvesSecretRefs(), TestLoadSecretRefErrors(), TestLoadSuccess() (+7 more)

### Community 55 - "Test catalogue"
Cohesion: 0.18
Nodes (10): convert_test.go, Coverage, internal/convert, internal/render, internal/tools/vulnjudge, main_test.go, render_test.go, Running the tests (+2 more)

### Community 56 - ".LeaderLocal"
Cohesion: 0.21
Nodes (8): TestLastLines(), TestLastLinesEqualCount(), Ops, hostMatches(), roleName(), shortHost(), TestRoleName(), lastLines()

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

### Community 62 - "NewManager"
Cohesion: 0.26
Nodes (12): NewManager(), TestManagerNilSinks(), Manager, mgrOver(), TestCtrExecutorRefusesUnapprovedRuntime(), TestCtrRuntimeDefaultArgvUnchanged(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime() (+4 more)

### Community 63 - "internal/engine"
Cohesion: 0.67
Nodes (3): internal/engine, resolve_test.go, runner_test.go

### Community 65 - "Config"
Cohesion: 0.18
Nodes (14): Replication, Scaling, TLS, Config, orNone(), orValue(), setOrMissing(), setOrNone() (+6 more)

### Community 66 - ".redundancyLocalPrimary"
Cohesion: 0.26
Nodes (9): field(), TestField(), TestHTTPStatusHelpers(), TestPrimaryRedundancyUp(), TestFieldLabelWithoutColon(), httpStatusLines(), isHTTP2xx(), primaryRedundancyUp() (+1 more)

### Community 68 - "Ops"
Cohesion: 0.29
Nodes (6): Ops, domainCertsScript(), removeDomainCertsScript(), sortedKeys(), TestDomainCertsScriptSorted(), TestSortedKeys()

### Community 69 - ".ServerCert"
Cohesion: 0.47
Nodes (4): concatFiles(), serverCertFile(), serverCertScript(), TestServerCertScript()

## Knowledge Gaps
- **177 isolated node(s):** `check_test.go`, `cluster_test.go`, `deploy_test.go`, `names_test.go`, `operator_test.go` (+172 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `cli_test.go`, `ctrCfg`, `Cluster`, `captureStdout`, `Manager`, `config.go`, `newTestOps`, `capRunner`, `NewCluster`, `convert_test.go`, `manager_test.go`, `Cluster`, `prep_test.go`, `verify_local_test.go`, `strings.Builder`, `recRunner`, `Compose`, `haCfg`, `kubectlTransport`, `secrets_test.go`, `context.Context`, `.Run`, `hasCall`, `render.go`, `containerTransport`, `NewManager`?**
  _High betweenness centrality (0.071) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `ctrCfg`, `captureStdout`, `config_test.go`, `Manager`, `config.go`, `capRunner`, `convert_test.go`, `manager_test.go`, `tierFor`, `github.com/spf13/cobra.Command`, `CheckCommand`, `manager.go`, `Compose`, `execguard_test.go`, `load.go`, `.Run`, `render.go`, `Load`, `containerTransport`, `NewManager`?**
  _High betweenness centrality (0.060) - this node is a cross-community bridge._
- **Why does `bg()` connect `bg` to `context.Context`?**
  _High betweenness centrality (0.029) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `check_test.go`, `cluster_test.go`, `deploy_test.go` to the rest of the system?**
  _177 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06681896059394632 - nodes in this community are weakly interconnected._
- **Should `cli_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08472344161545214 - nodes in this community are weakly interconnected._