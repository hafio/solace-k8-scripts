# Graph Report - solace-k8-scripts  (2026-08-19)

## Corpus Check
- 88 files · ~193,441 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1744 nodes · 5451 edges · 72 communities (67 shown, 5 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 854 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `02e7a38d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- cli_test.go
- manager_test.go
- Cluster
- captureStdout
- config_test.go
- Commands
- Manager
- Config
- testing.T
- newTestOps
- capRunner
- NewCluster
- eqArgs
- convert_test.go
- internal/k8s
- ctrCfg
- cli/platform_test.go
- Cluster
- dev.sh
- prep_test.go
- dev.ps1
- CLAUDE.md
- judge
- verify_local_test.go
- Role
- scaling_test.go
- Platform
- strings.Builder
- solace
- Command
- Command reference
- .Preflight
- completion_test.go
- cliScriptPath
- .resolveSecretRefs
- recRunner
- Load
- Compose
- captureStderr
- haCfg
- Ops
- load.go
- secrets_test.go
- context.Context
- coverage_test.go
- .Run
- runRoot
- render.go
- scripts.go
- newRootCmd
- scripts_test.go
- ranContains
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
- RenderOperator
- ResolveEnvPath
- .releaseToBackup
- .AdditionalUsers
- assertContainerBlockDefaults
- .ResolveNode
- .Preflight

## God Nodes (most connected - your core abstractions)
1. `ctrCfg()` - 82 edges
2. `Role` - 81 edges
3. `Config` - 80 edges
4. `newCapMgr()` - 74 edges
5. `newTestOps()` - 73 edges
6. `bg()` - 70 edges
7. `Platform` - 67 edges
8. `Commands` - 59 edges
9. `Manager` - 54 edges
10. `matchCLI()` - 52 edges

## Surprising Connections (you probably didn't know these)
- `activity()` --calls--> `countContains()`  [INFERRED]
  internal/broker/verify_ops.go → internal/broker/broker.go
- `matchCLI()` --calls--> `cliArg()`  [INFERRED]
  internal/broker/broker_test.go → internal/broker/transport.go
- `TestAdditionalUsersRunCLITransportError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestDisableDefaultUsersShowVPNError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestDisableDefaultVPNShowError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (72 total, 5 thin omitted)

### Community 0 - "bg"
Cohesion: 0.08
Nodes (91): TestCtrManagerConfirmWiring(), containerWhat(), ctrLogin(), ctrManager(), ctrOps(), App, opCtrCheck(), opCtrCLI() (+83 more)

### Community 1 - "cli_test.go"
Cohesion: 0.15
Nodes (34): allowRuntime(), fakeBinaryOnPath(), firstLine(), runCtr(), runStandalone(), TestAnnounceCommandsNamesResolvedBinaries(), TestBinaryAnnouncementWiring(), TestCtrConfigAllArms() (+26 more)

### Community 2 - "manager_test.go"
Cohesion: 0.08
Nodes (52): bytes.Buffer, fileExists(), assertMode(), containsStr(), Manager, hasCall(), maskedKeys(), newEchoMgr() (+44 more)

### Community 3 - "Cluster"
Cohesion: 0.11
Nodes (11): HARoles(), lbServiceName(), pvcName(), RestartOrder(), stsName(), TestResourceNames(), TestRestartOrder(), filterLines() (+3 more)

### Community 4 - "captureStdout"
Cohesion: 0.14
Nodes (22): opCall, opRunner, captureStdout(), failDisableDefaultUsersUpload(), k8sConfigAllRunner(), k8sUpOutputHook(), loadDirect(), opArgvMatch() (+14 more)

### Community 5 - "config_test.go"
Cohesion: 0.07
Nodes (42): Config, haNodesConfig(), TestApplyDefaultsK8s(), TestApplyDefaultsK8sTLS(), TestApplyDefaultsPodmanRootlessHomeDir(), TestApplyDefaultsPodmanRootlessXDG(), TestContainerBlock(), TestContainerRuntime() (+34 more)

### Community 6 - "Commands"
Cohesion: 0.03
Nodes (59): Commands, solace-util, solace-util check, solace-util cli, solace-util completion, solace-util completion bash, solace-util completion fish, solace-util completion powershell (+51 more)

### Community 7 - "Manager"
Cohesion: 0.11
Nodes (10): os.FileMode, defaultGenPSK(), exactName(), Manager, orNone(), platformTitle(), replacePSKLine(), secretSummary() (+2 more)

### Community 8 - "Config"
Cohesion: 0.11
Nodes (24): Admin, Broker, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, Network (+16 more)

### Community 9 - "testing.T"
Cohesion: 0.12
Nodes (35): testing.T, decodeRuntime(), TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds() (+27 more)

### Community 10 - "newTestOps"
Cohesion: 0.12
Nodes (33): appUsers(), Ops, newTestOps(), TestAdditionalUsers(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDisableDefaultUsers() (+25 more)

### Community 11 - "capRunner"
Cohesion: 0.23
Nodes (11): capCall, capRunner, NewTransport(), dockerCfg(), eqArgs(), podmanCfg(), TestTransportCopy(), TestTransportEchoHidesUploadBody() (+3 more)

### Community 12 - "NewCluster"
Cohesion: 0.12
Nodes (34): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckOperatorNS(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass() (+26 more)

### Community 13 - "eqArgs"
Cohesion: 0.15
Nodes (26): Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived(), TestOperatorNSExplicit() (+18 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (55): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+47 more)

### Community 15 - "internal/k8s"
Cohesion: 0.17
Nodes (12): check_test.go, cluster_test.go, deploy_test.go, internal/k8s, names_test.go, operator_test.go, ops_test.go, preflight_test.go (+4 more)

### Community 16 - "ctrCfg"
Cohesion: 0.11
Nodes (44): ctrCfg(), newCapMgr(), TestContainerRunningMatchesNameExactly(), TestManagerCheckDNSFailsLoudInHA(), TestManagerCheckPodmanEUID(), TestManagerCheckReachableError(), TestManagerCheckStandaloneDNSWarnsOnly(), TestManagerCopy() (+36 more)

### Community 17 - "cli/platform_test.go"
Cohesion: 0.28
Nodes (16): App, runPlatform(), TestMultiPlatformNonInteractiveIsRefused(), TestMultiPlatformPromptRejectsBadAnswer(), TestMultiPlatformPromptSelects(), TestNoPlatformSectionIsRefused(), TestPlatformFlagAcceptsAbbreviations(), TestPlatformFlagRejectsUndeclaredSection() (+8 more)

### Community 18 - "Cluster"
Cohesion: 0.13
Nodes (11): bufio.Reader, Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), rolePlacementLabels(), splitLabel(), TestIsBuiltinLabel() (+3 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "prep_test.go"
Cohesion: 0.21
Nodes (20): saCfg(), adminCfg(), Cluster, labelCluster(), TestCreateSecretsAdminOnly(), TestCreateSecretsPreflight(), TestCreateSecretsStopsOnPreflightFailure(), TestDeleteSecrets() (+12 more)

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
Cohesion: 0.12
Nodes (14): downloadErrTransport, fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrMatchTransport (+6 more)

### Community 26 - "scaling_test.go"
Cohesion: 0.12
Nodes (18): scalingTier, containerMem(), Config, Config, setContainerMem(), TestApplyScalingTierDefaultsContainerBlocks(), TestApplyScalingTierDefaultsK8s(), TestApplyScalingTierDefaultsMemOverride() (+10 more)

### Community 27 - "Platform"
Cohesion: 0.06
Nodes (108): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/cobra.ShellCompDirective, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, TestConfirmFlagShortcuts() (+100 more)

### Community 28 - "strings.Builder"
Cohesion: 0.20
Nodes (16): WeightedNodeTerm, strings.Builder, NodeAffinity, NodeMatchExpr, Placement, PodAffinityTerm, parseToleration(), sortedKeys() (+8 more)

### Community 31 - "Command"
Cohesion: 0.06
Nodes (43): commandRules, keyValueEntries, Command, checkBinary(), CheckCommand(), checkFlagShape(), checkToken(), clusterRules() (+35 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 34 - "completion_test.go"
Cohesion: 0.24
Nodes (16): runComplete(), TestAllowCommandOffersNoFiles(), TestCompletionHelpStillWorks(), TestCompletionNeedsAShell(), TestCompletionNoDescriptions(), TestCompletionScriptsGenerate(), TestDirFlagCompletesDirectories(), TestEnvFlagCompletesEnvFiles() (+8 more)

### Community 35 - "cliScriptPath"
Cohesion: 0.20
Nodes (12): TestDiagnostics(), TestDomainCerts(), TestPathHelpers(), TestServerCert(), TestValidName(), validName(), TestDisableDefaultUsersShowVPNError(), TestReleaseToBackupReleasedTimeout() (+4 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.17
Nodes (10): TestCanIAnswerReadsTheLastLine(), NewTransport(), isCanI(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest() (+2 more)

### Community 38 - "Load"
Cohesion: 0.23
Nodes (16): minimalK8s(), TestLoadBashEnvFileHint(), TestLoadNotYAMLHint(), TestLoadParseError(), TestLoadReadError(), TestLoadRejectsTheOldK8sSection(), TestLoadResolvesSecretRefs(), TestLoadSecretRefErrors() (+8 more)

### Community 39 - "Compose"
Cohesion: 0.16
Nodes (28): emitCtrArtifact(), NodeIdentity, Compose(), ContainerSecrets(), EnvFile(), EnvPairs(), ContainerSecret, healthCmd() (+20 more)

### Community 40 - "captureStderr"
Cohesion: 0.15
Nodes (15): os.File, capture(), captureStderr(), runStatusStderr(), TestBashEnvGivenToEnvFlag(), TestConvertErrorPaths(), TestConvertRoundTrip(), TestConvertToFile() (+7 more)

### Community 41 - "haCfg"
Cohesion: 0.20
Nodes (14): haCfg(), TestHARoles(), TestProductKeyRoles(), TestShowAll(), TestShowAllWrapsGetError(), TestCreateSecretsFailsWithoutAdminFields(), TestDeleteSecretsSkipsUnconfiguredAdminSecret(), TestClusterHonoursRuntime() (+6 more)

### Community 42 - "Ops"
Cohesion: 0.24
Nodes (5): time.Duration, Ops, New(), TestNewDefaults(), Transport

### Community 43 - "load.go"
Cohesion: 0.22
Nodes (12): TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config, parseError() (+4 more)

### Community 44 - "secrets_test.go"
Cohesion: 0.17
Nodes (20): GenSecrets(), TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), DockerRegistrySecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden() (+12 more)

### Community 45 - "context.Context"
Cohesion: 0.15
Nodes (5): context.Context, Runner, Cluster, Cluster, Cluster

### Community 46 - "coverage_test.go"
Cohesion: 0.07
Nodes (43): matchCLI(), writeFile(), TestDiagnosticsMkdirError(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultVPNDisableError(), TestDomainCertsBadFilename() (+35 more)

### Community 47 - ".Run"
Cohesion: 0.13
Nodes (17): Echo, Exec, os/exec.Cmd, TestChildEnvNamesAreNotSystemVariables(), TestExecEchoesOnEveryMethod(), TestExecIsSilentWithoutVerbose(), TestExecVerboseAnnouncesEveryCommand(), TestResolveMissingBinaryIsActionable() (+9 more)

### Community 48 - "runRoot"
Cohesion: 0.12
Nodes (28): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandIsRepeatable(), TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), TestEscalationIsRefusedEndToEnd(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb(), TestPathRuntimeIsRefused() (+20 more)

### Community 49 - "render.go"
Cohesion: 0.12
Nodes (22): Cluster, boolStr(), BrokerCR(), containerSecretSpecs(), cut(), escapePercent(), groupKey(), itoa() (+14 more)

### Community 50 - "scripts.go"
Cohesion: 0.12
Nodes (20): showCmd, concatFiles(), Ops, disableDefaultUsersScript(), disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), removeDomainCertsScript() (+12 more)

### Community 51 - "newRootCmd"
Cohesion: 0.11
Nodes (20): TestAllowCommandIsRegisteredWhereItExecutes(), collectPaths(), findCmd(), TestEmit(), TestEveryRunnableCommandIsWired(), TestExecute(), TestFlagsRegistered(), TestNotImplemented() (+12 more)

### Community 52 - "scripts_test.go"
Cohesion: 0.21
Nodes (10): domainCertsScript(), parseVPNNames(), sortedKeys(), TestDomainCertsScriptSorted(), TestGatherConfigsScript(), TestParseVPNNames(), TestParseVPNNamesNoSeparator(), TestSortedKeys() (+2 more)

### Community 53 - "ranContains"
Cohesion: 0.33
Nodes (6): ranContains(), TestExecCLI(), TestDisableDefaultVPNShowError(), TestExecCLIRunError(), TestRemoveCLIWarnsOnFailure(), TestRemoveDomainCertsRunCLIError()

### Community 55 - "Test catalogue"
Cohesion: 0.18
Nodes (10): convert_test.go, Coverage, internal/convert, internal/render, internal/tools/vulnjudge, main_test.go, render_test.go, Running the tests (+2 more)

### Community 56 - ".LeaderLocal"
Cohesion: 0.16
Nodes (11): TestLastLines(), TestLastLinesEqualCount(), assertLeaderScript(), showRedundancyDetailScript(), TestAssertLeaderScript(), Ops, hostMatches(), roleName() (+3 more)

### Community 57 - "internal/cli"
Cohesion: 0.33
Nodes (6): allowcommand_test.go, cli_test.go, commanddoc_test.go, completion_test.go, internal/cli, platform_test.go

### Community 58 - "internal/broker"
Cohesion: 0.40
Nodes (5): broker_test.go, coverage_test.go, internal/broker, scripts_test.go, verify_local_test.go

### Community 59 - "internal/config"
Cohesion: 0.33
Nodes (6): command_test.go, config_test.go, execguard_test.go, internal/config, platform_test.go, scaling_test.go

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

### Community 65 - "RenderOperator"
Cohesion: 0.27
Nodes (9): orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), operatorImage(), RenderOperator(), watchNamespace() (+1 more)

### Community 66 - "ResolveEnvPath"
Cohesion: 0.50
Nodes (5): envTree(), TestResolveEnvPath(), TestResolveEnvPathDefaultInBaseDir(), TestResolveEnvPathEmptyBaseDir(), ResolveEnvPath()

### Community 67 - ".releaseToBackup"
Cohesion: 0.20
Nodes (11): field(), TestField(), TestHTTPStatusHelpers(), TestPrimaryRedundancyUp(), TestFieldLabelWithoutColon(), activity(), Ops, httpStatusLines() (+3 more)

### Community 68 - ".AdditionalUsers"
Cohesion: 0.19
Nodes (11): containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), validCLILine(), validCLIPassword(), additionalUsersScript(), productKeysScript() (+3 more)

### Community 69 - "assertContainerBlockDefaults"
Cohesion: 0.67
Nodes (4): assertContainerBlockDefaults(), assertContainerScaling(), TestApplyDefaultsDocker(), TestApplyDefaultsPodmanRootful()

## Knowledge Gaps
- **117 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+112 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Platform` connect `Platform` to `manager_test.go`, `captureStdout`, `config_test.go`, `Load`, `Manager`, `Config`, `Compose`, `load.go`, `capRunner`, `convert_test.go`, `ctrCfg`, `containerTransport`, `scaling_test.go`, `NewManager`, `Command`?**
  _High betweenness centrality (0.075) - this node is a cross-community bridge._
- **Why does `Config` connect `Config` to `bg`, `cli_test.go`, `manager_test.go`, `Cluster`, `captureStdout`, `Manager`, `newTestOps`, `capRunner`, `NewCluster`, `convert_test.go`, `ctrCfg`, `Cluster`, `prep_test.go`, `verify_local_test.go`, `Role`, `Platform`, `strings.Builder`, `recRunner`, `Compose`, `haCfg`, `Ops`, `secrets_test.go`, `context.Context`, `render.go`, `containerTransport`, `NewManager`, `RenderOperator`?**
  _High betweenness centrality (0.069) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `bg`, `cliScriptPath`, `.AdditionalUsers`, `.releaseToBackup`, `.ResolveNode`, `Manager`, `Compose`, `Cluster`, `Ops`, `scripts.go`, `Cluster`, `scripts_test.go`, `containerTransport`, `.LeaderLocal`, `verify_local_test.go`, `Platform`?**
  _High betweenness centrality (0.035) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _117 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.07760635811126694 - nodes in this community are weakly interconnected._
- **Should `cli_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.14761904761904762 - nodes in this community are weakly interconnected._