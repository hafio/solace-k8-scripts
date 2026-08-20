# Graph Report - solace-k8-scripts  (2026-08-20)

## Corpus Check
- 90 files · ~207,089 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1795 nodes · 5637 edges · 76 communities (72 shown, 4 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 893 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `02e7a38d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- cli_test.go
- render.go
- context.Context
- captureStdout
- config_test.go
- Commands
- Manager
- Config
- load.go
- newTestOps
- capRunner
- NewCluster
- eqArgs
- convert_test.go
- internal/k8s
- manager_test.go
- cli/platform_test.go
- secrets_test.go
- dev.sh
- prep_test.go
- dev.ps1
- CLAUDE.md
- judge
- verify_local_test.go
- Role
- newEchoMgr
- Platform
- strings.Builder
- solace
- Command
- Command reference
- io.Writer
- completion_test.go
- runRootWith
- .resolveSecretRefs
- haCfg
- kubectlTransport
- Compose
- .releaseToBackup
- k8s/runtime_test.go
- TestServerCert
- .Run
- Cluster
- runRoot
- coverage_test.go
- testing.T
- Load
- manager.go
- scripts_test.go
- newRootCmd
- bytes.Buffer
- eqArgs
- rootlessNoFileMgr
- Test catalogue
- .LeaderLocal
- internal/cli
- internal/broker
- internal/config
- internal/container
- Fixtures and doubles
- scripts.go
- internal/engine
- RenderOperator
- container/preflight_test.go
- ResolveEnvPath
- .AdditionalUsers
- capture
- scaling_test.go
- TestManagerDeployDockerPassesSecretsAsEnv
- assertContainerBlockDefaults
- .Preflight
- .redundancyLocalPrimary
- .gatherNode

## God Nodes (most connected - your core abstractions)
1. `ctrCfg()` - 91 edges
2. `newCapMgr()` - 82 edges
3. `Role` - 81 edges
4. `Config` - 78 edges
5. `newTestOps()` - 73 edges
6. `bg()` - 68 edges
7. `Platform` - 67 edges
8. `Commands` - 66 edges
9. `Manager` - 58 edges
10. `NewCluster()` - 57 edges

## Surprising Connections (you probably didn't know these)
- `activity()` --calls--> `countContains()`  [INFERRED]
  internal/broker/verify_ops.go → internal/broker/broker.go
- `matchCLI()` --calls--> `cliArg()`  [INFERRED]
  internal/broker/broker_test.go → internal/broker/transport.go
- `TestAdditionalUsersRunCLITransportError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestDisableDefaultUsersShowVPNError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestReleaseToBackupReleasedTimeout()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (76 total, 4 thin omitted)

### Community 0 - "bg"
Cohesion: 0.06
Nodes (101): time.Duration, Ops, New(), TestNewDefaults(), TestCtrManagerConfirmWiring(), App, step(), warn() (+93 more)

### Community 1 - "cli_test.go"
Cohesion: 0.11
Nodes (43): allowRuntime(), captureStderr(), collectPaths(), fakeBinaryOnPath(), firstLine(), runCtr(), runStandalone(), runStatusStderr() (+35 more)

### Community 2 - "render.go"
Cohesion: 0.14
Nodes (18): opCtrGenSecrets(), boolStr(), ContainerSecrets(), containerSecretSpecs(), cut(), ContainerSecret, parsePort(), secretFilePath() (+10 more)

### Community 3 - "context.Context"
Cohesion: 0.09
Nodes (16): context.Context, Cluster, Cluster, HARoles(), lbServiceName(), podName(), pvcName(), RestartOrder() (+8 more)

### Community 4 - "captureStdout"
Cohesion: 0.14
Nodes (20): opCall, opRunner, captureStdout(), failDisableDefaultUsersUpload(), k8sDeployAllOutputHook(), loadDirect(), opArgvMatch(), opCanI() (+12 more)

### Community 5 - "config_test.go"
Cohesion: 0.07
Nodes (42): Config, haNodesConfig(), TestApplyDefaultsK8s(), TestApplyDefaultsK8sTLS(), TestApplyDefaultsPodmanRootlessHomeDir(), TestApplyDefaultsPodmanRootlessXDG(), TestContainerBlock(), TestContainerRuntime() (+34 more)

### Community 6 - "Commands"
Cohesion: 0.03
Nodes (66): Commands, solace-util, solace-util auto-complete, solace-util auto-complete bash, solace-util auto-complete fish, solace-util auto-complete powershell, solace-util auto-complete zsh, solace-util check (+58 more)

### Community 7 - "Manager"
Cohesion: 0.13
Nodes (3): os.FileMode, fileExists(), Manager

### Community 8 - "Config"
Cohesion: 0.11
Nodes (26): Admin, Broker, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, Network (+18 more)

### Community 9 - "load.go"
Cohesion: 0.22
Nodes (12): TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config, parseError() (+4 more)

### Community 10 - "newTestOps"
Cohesion: 0.11
Nodes (39): appUsers(), Ops, newTestOps(), TestAdditionalUsers(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDiagnostics() (+31 more)

### Community 11 - "capRunner"
Cohesion: 0.23
Nodes (10): capCall, capRunner, NewTransport(), dockerCfg(), podmanCfg(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs() (+2 more)

### Community 12 - "NewCluster"
Cohesion: 0.12
Nodes (37): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckOperatorNS(), TestCheckStorageClass(), TestResolveStorageClass(), NewCluster() (+29 more)

### Community 13 - "eqArgs"
Cohesion: 0.14
Nodes (29): TestReachable(), Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived() (+21 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (55): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+47 more)

### Community 15 - "internal/k8s"
Cohesion: 0.17
Nodes (12): check_test.go, cluster_test.go, deploy_test.go, internal/k8s, names_test.go, operator_test.go, ops_test.go, preflight_test.go (+4 more)

### Community 16 - "manager_test.go"
Cohesion: 0.11
Nodes (66): ctrCfg(), hasCall(), newCapMgr(), TestContainerRunningMatchesNameExactly(), TestManagerCheckDNSFailsLoudInHA(), TestManagerCheckPodmanEUID(), TestManagerCheckReachableError(), TestManagerCheckStandaloneDNSWarnsOnly() (+58 more)

### Community 17 - "cli/platform_test.go"
Cohesion: 0.28
Nodes (16): App, runPlatform(), TestMultiPlatformNonInteractiveIsRefused(), TestMultiPlatformPromptRejectsBadAnswer(), TestMultiPlatformPromptSelects(), TestNoPlatformSectionIsRefused(), TestPlatformFlagAcceptsAbbreviations(), TestPlatformFlagRejectsUndeclaredSection() (+8 more)

### Community 18 - "secrets_test.go"
Cohesion: 0.15
Nodes (19): TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), DockerRegistrySecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden(), decodeDataValue() (+11 more)

### Community 19 - "dev.sh"
Cohesion: 0.18
Nodes (21): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+13 more)

### Community 20 - "prep_test.go"
Cohesion: 0.15
Nodes (24): ProductKeyRoles(), saCfg(), TestHARoles(), TestProductKeyRoles(), adminCfg(), Cluster, labelCluster(), TestCreateNamespaceApplyFails() (+16 more)

### Community 21 - "dev.ps1"
Cohesion: 0.18
Nodes (17): Get-Log(), Get-Now(), Build-One(), Cap(), Ok(), Step(), Task-build(), Task-cov() (+9 more)

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
Cohesion: 0.11
Nodes (14): downloadErrTransport, fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrMatchTransport (+6 more)

### Community 26 - "newEchoMgr"
Cohesion: 0.15
Nodes (13): NewManager(), newEchoMgr(), TestManagerCheckDryRun(), TestManagerDeletePodmanPurgeRootless(), TestManagerDeployDockerDryRunMasksSecretEnv(), TestManagerDeployPodmanDryRunHidesSecretBytes(), TestManagerDeployPodmanDryRunSkipsWrite(), TestManagerDockerCheckProbesCompose() (+5 more)

### Community 27 - "Platform"
Cohesion: 0.07
Nodes (96): layer, opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/cobra.ShellCompDirective, github.com/spf13/pflag.FlagSet, TestConfirmFlagShortcuts(), TestFirstArg() (+88 more)

### Community 28 - "strings.Builder"
Cohesion: 0.18
Nodes (18): WeightedNodeTerm, strings.Builder, NodeAffinity, NodeMatchExpr, Placement, PodAffinityTerm, parseToleration(), sortedKeys() (+10 more)

### Community 31 - "Command"
Cohesion: 0.06
Nodes (44): commandRules, keyValueEntries, Command, checkBinary(), CheckCommand(), checkFlagShape(), checkToken(), clusterRules() (+36 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "io.Writer"
Cohesion: 0.16
Nodes (9): io.Reader, io.Writer, TestPromptYes(), TestPromptYesNo(), promptLine(), promptYes(), promptYesNo(), solaceRows() (+1 more)

### Community 34 - "completion_test.go"
Cohesion: 0.37
Nodes (12): runComplete(), TestAllowCommandOffersNoFiles(), TestDirFlagCompletesDirectories(), TestEnvFlagCompletesEnvFiles(), TestEnvFlagPrefixFilters(), TestEnvFlagWithPathDefersToShell(), TestNoArgsLeafOffersNoFiles(), TestPlatformFlagCompletes() (+4 more)

### Community 35 - "runRootWith"
Cohesion: 0.17
Nodes (21): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), TestEscalationIsRefusedEndToEnd(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb(), TestPathRuntimeIsRefused(), TestSmuggledSubcommandIsRefused() (+13 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "haCfg"
Cohesion: 0.15
Nodes (14): haCfg(), TestShowAllReportsAndContinuesOnGetError(), TestCanIAnswerReadsTheLastLine(), TestCreateSecretsFailsWithoutAdminFields(), TestDeleteSecretsSkipsUnconfiguredAdminSecret(), NewTransport(), isCanI(), TestTransportCopy() (+6 more)

### Community 39 - "Compose"
Cohesion: 0.16
Nodes (30): opCtrGenArtifact(), NodeIdentity, BrokerCR(), Compose(), EnvFile(), EnvPairs(), escapePercent(), groupKey() (+22 more)

### Community 41 - "k8s/runtime_test.go"
Cohesion: 0.43
Nodes (7): TestClusterHonoursRuntime(), TestExecutorRefusesUnapprovedRuntime(), TestRuntimeDefaultArgvUnchanged(), TestTransportHonoursRuntime(), unapprovedCfg(), withLeading(), wrappedCfg()

### Community 42 - "TestServerCert"
Cohesion: 0.24
Nodes (9): TestPathHelpers(), TestServerCert(), concatFiles(), serverCertFile(), serverCertScript(), TestServerCertScript(), certPath(), cliArg() (+1 more)

### Community 43 - ".Run"
Cohesion: 0.24
Nodes (5): Echo, Exec, os/exec.Cmd, Quote(), TestQuote()

### Community 44 - "Cluster"
Cohesion: 0.12
Nodes (14): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels() (+6 more)

### Community 45 - "runRoot"
Cohesion: 0.14
Nodes (18): runRoot(), TestBashEnvGivenToEnvFlag(), TestConvertErrorPaths(), TestConvertParseError(), TestConvertToFile(), TestConvertToStdout(), TestConvertWriteError(), TestGenSecretsRefusesEmptyValue() (+10 more)

### Community 46 - "coverage_test.go"
Cohesion: 0.06
Nodes (48): matchCLI(), ranContains(), writeFile(), TestDiagnosticsMkdirError(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultVPNDisableError() (+40 more)

### Community 47 - "testing.T"
Cohesion: 0.10
Nodes (39): testing.T, decodeRuntime(), TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds() (+31 more)

### Community 48 - "Load"
Cohesion: 0.23
Nodes (16): minimalK8s(), TestLoadBashEnvFileHint(), TestLoadNotYAMLHint(), TestLoadParseError(), TestLoadReadError(), TestLoadRejectsTheOldK8sSection(), TestLoadResolvesSecretRefs(), TestLoadSecretRefErrors() (+8 more)

### Community 49 - "manager.go"
Cohesion: 0.21
Nodes (9): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), splitLimit() (+1 more)

### Community 50 - "scripts_test.go"
Cohesion: 0.19
Nodes (11): additionalUsersScript(), disableDefaultUsersScript(), domainCertsScript(), parseVPNNames(), sortedKeys(), TestAdditionalUsersScript(), TestDisableDefaultUsersScriptQuoting(), TestDomainCertsScriptSorted() (+3 more)

### Community 51 - "newRootCmd"
Cohesion: 0.09
Nodes (24): applyAliases(), TestAliasesDoNotCollide(), TestAliasesResolveToTheCanonicalCommand(), TestDangerousVerbsHaveNoBareAlias(), TestEveryAliasEntryIsLive(), TestGroupsRejectAnUnknownNoun(), TestStartStopHaveNoAlias(), TestAllowCommandIsRegisteredWhereItExecutes() (+16 more)

### Community 52 - "bytes.Buffer"
Cohesion: 0.18
Nodes (13): bytes.Buffer, maskedKeys(), TestComposeSecretEnvIsTheOnlyChildEnvironment(), TestChildEnvNamesAreNotSystemVariables(), TestExecEchoesOnEveryMethod(), TestExecVerboseAnnouncesEveryCommand(), TestResolveMissingBinaryIsActionable(), TestResolveRefusesCurrentDirectory() (+5 more)

### Community 53 - "eqArgs"
Cohesion: 0.25
Nodes (13): TestManagerLogsCLIShell(), TestPreflightRunsBeforeAnything(), Manager, mgrOver(), TestCtrExecutorRefusesUnapprovedRuntime(), TestCtrRuntimeDefaultArgvUnchanged(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime() (+5 more)

### Community 54 - "rootlessNoFileMgr"
Cohesion: 0.22
Nodes (9): Manager, rootlessNoFileMgr(), setNoFile(), TestPrepHostRootlessNoFileDryRun(), TestPrepHostRootlessNoFileSufficient(), TestPrepHostRootlessNoFileTooLow(), TestPrepHostRootlessNoFileUnlimited(), TestPrepHostRootlessNoFileUnreadable() (+1 more)

### Community 55 - "Test catalogue"
Cohesion: 0.18
Nodes (10): convert_test.go, Coverage, internal/convert, internal/render, internal/tools/vulnjudge, main_test.go, render_test.go, Running the tests (+2 more)

### Community 56 - ".LeaderLocal"
Cohesion: 0.17
Nodes (10): TestLastLines(), TestLastLinesEqualCount(), assertLeaderScript(), TestAssertLeaderScript(), Ops, hostMatches(), roleName(), shortHost() (+2 more)

### Community 57 - "internal/cli"
Cohesion: 0.29
Nodes (7): aliases_test.go, allowcommand_test.go, cli_test.go, commanddoc_test.go, completion_test.go, internal/cli, platform_test.go

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

### Community 62 - "scripts.go"
Cohesion: 0.21
Nodes (13): showCmd, disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), revertActivityConfigureScript(), revertActivityScript(), showRedundancyDetailScript(), showRedundancyScript() (+5 more)

### Community 63 - "internal/engine"
Cohesion: 0.67
Nodes (3): internal/engine, resolve_test.go, runner_test.go

### Community 65 - "RenderOperator"
Cohesion: 0.18
Nodes (13): orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), joinYAMLDocs(), operatorImage(), RenderOperator() (+5 more)

### Community 66 - "container/preflight_test.go"
Cohesion: 0.33
Nodes (5): TestComposeSecretEnvNamesCannotBeSystemVars(), TestPreflightFailureStopsLifecycle(), TestPreflightFailureStopsTheDeploy(), TestPreflightHintIsPlatformShaped(), TestPreflightIsPreviewableUnderDryRun()

### Community 67 - "ResolveEnvPath"
Cohesion: 0.50
Nodes (5): envTree(), TestResolveEnvPath(), TestResolveEnvPathDefaultInBaseDir(), TestResolveEnvPathEmptyBaseDir(), ResolveEnvPath()

### Community 68 - ".AdditionalUsers"
Cohesion: 0.16
Nodes (12): containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), TestValidName(), validCLILine(), validCLIPassword(), validName() (+4 more)

### Community 69 - "capture"
Cohesion: 0.50
Nodes (4): os.File, capture(), TestIsTTYClosedFile(), isTTY()

### Community 70 - "scaling_test.go"
Cohesion: 0.12
Nodes (18): scalingTier, containerMem(), Config, Config, setContainerMem(), TestApplyScalingTierDefaultsContainerBlocks(), TestApplyScalingTierDefaultsK8s(), TestApplyScalingTierDefaultsMemOverride() (+10 more)

### Community 71 - "TestManagerDeployDockerPassesSecretsAsEnv"
Cohesion: 0.40
Nodes (5): assertMode(), containsStr(), TestManagerDeployDockerPassesSecretsAsEnv(), TestManagerDeployPodmanCreatesSecrets(), TestManagerRedeployUnchangedHintsRotation()

### Community 72 - "assertContainerBlockDefaults"
Cohesion: 0.67
Nodes (4): assertContainerBlockDefaults(), assertContainerScaling(), TestApplyDefaultsDocker(), TestApplyDefaultsPodmanRootful()

### Community 74 - ".redundancyLocalPrimary"
Cohesion: 0.26
Nodes (9): field(), TestField(), TestHTTPStatusHelpers(), TestPrimaryRedundancyUp(), TestFieldLabelWithoutColon(), httpStatusLines(), isHTTP2xx(), primaryRedundancyUp() (+1 more)

### Community 75 - ".gatherNode"
Cohesion: 0.33
Nodes (4): gatherConfigsScript(), TestGatherConfigsScript(), TestZipConfigsScript(), zipConfigsScript()

## Knowledge Gaps
- **125 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+120 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `cli_test.go`, `render.go`, `context.Context`, `captureStdout`, `Manager`, `newTestOps`, `capRunner`, `NewCluster`, `convert_test.go`, `manager_test.go`, `secrets_test.go`, `prep_test.go`, `verify_local_test.go`, `Role`, `newEchoMgr`, `strings.Builder`, `io.Writer`, `haCfg`, `kubectlTransport`, `Compose`, `k8s/runtime_test.go`, `Cluster`, `eqArgs`, `rootlessNoFileMgr`, `RenderOperator`?**
  _High betweenness centrality (0.067) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `bg`, `render.go`, `captureStdout`, `config_test.go`, `Manager`, `Config`, `load.go`, `capRunner`, `convert_test.go`, `manager_test.go`, `Role`, `newEchoMgr`, `Command`, `Compose`, `Load`, `manager.go`, `eqArgs`, `rootlessNoFileMgr`, `scaling_test.go`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `bg`, `context.Context`, `.AdditionalUsers`, `kubectlTransport`, `Manager`, `.releaseToBackup`, `Compose`, `.redundancyLocalPrimary`, `.gatherNode`, `Cluster`, `scripts_test.go`, `prep_test.go`, `.LeaderLocal`, `verify_local_test.go`, `Platform`, `scripts.go`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **Are the 10 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **Are the 10 inferred relationships involving `newCapMgr()` (e.g. with `NewManager()` and `TestComposeSecretEnvIsTheOnlyChildEnvironment()`) actually correct?**
  _`newCapMgr()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _125 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.056598016781083144 - nodes in this community are weakly interconnected._