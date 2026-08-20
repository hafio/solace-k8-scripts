# Graph Report - solace-k8-scripts  (2026-08-20)

## Corpus Check
- 90 files · ~206,737 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1792 nodes · 5623 edges · 73 communities (69 shown, 4 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 889 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `02e7a38d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- cli_test.go
- render.go
- Cluster
- captureStdout
- testing.T
- Commands
- Manager
- config.go
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
- BrokerCR
- solace
- Command
- Command reference
- context.Context
- completion_test.go
- runRootWith
- .resolveSecretRefs
- recRunner
- kubectlTransport
- Compose
- .releaseToBackup
- k8s/runtime_test.go
- TestServerCert
- .Run
- Cluster
- Ops
- coverage_test.go
- runner_test.go
- .Run
- manager.go
- scripts_test.go
- newRootCmd
- haCfg
- command_test.go
- HARoles
- Test catalogue
- .LeaderLocal
- internal/cli
- internal/broker
- internal/config
- internal/container
- Fixtures and doubles
- scripts.go
- internal/engine
- Config
- container/preflight_test.go
- containerTransport
- .AdditionalUsers
- .DomainCerts
- .Preflight
- verify_ops.go
- .gatherNode

## God Nodes (most connected - your core abstractions)
1. `ctrCfg()` - 91 edges
2. `newCapMgr()` - 82 edges
3. `Role` - 80 edges
4. `Config` - 77 edges
5. `newTestOps()` - 73 edges
6. `bg()` - 68 edges
7. `Platform` - 67 edges
8. `Commands` - 65 edges
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

## Communities (73 total, 4 thin omitted)

### Community 0 - "bg"
Cohesion: 0.06
Nodes (105): New(), TestNewDefaults(), TestConfirmFlagShortcuts(), TestConfirmNonTTY(), TestCtrManagerConfirmWiring(), TestWarnAndStep(), step(), warn() (+97 more)

### Community 1 - "cli_test.go"
Cohesion: 0.09
Nodes (47): os.File, allowRuntime(), capture(), captureStderr(), fakeBinaryOnPath(), firstLine(), runRoot(), runStandalone() (+39 more)

### Community 2 - "render.go"
Cohesion: 0.15
Nodes (17): containerSecretSpecs(), cut(), EnvPairs(), escapePercent(), ContainerSecret, groupKey(), itoa(), parsePort() (+9 more)

### Community 3 - "Cluster"
Cohesion: 0.17
Nodes (6): podName(), RestartOrder(), filterLines(), Cluster, TestFilterLines(), surveySection

### Community 4 - "captureStdout"
Cohesion: 0.13
Nodes (21): opCall, opRunner, captureStdout(), failDisableDefaultUsersUpload(), k8sDeployAllOutputHook(), loadDirect(), opArgvMatch(), opCanI() (+13 more)

### Community 5 - "testing.T"
Cohesion: 0.06
Nodes (82): testing.T, assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, haNodesConfig(), minimalK8s(), TestApplyDefaultsDocker() (+74 more)

### Community 6 - "Commands"
Cohesion: 0.03
Nodes (65): Commands, solace-util, solace-util auto-complete, solace-util auto-complete bash, solace-util auto-complete fish, solace-util auto-complete powershell, solace-util auto-complete zsh, solace-util check (+57 more)

### Community 7 - "Manager"
Cohesion: 0.15
Nodes (3): os.FileMode, fileExists(), Manager

### Community 8 - "config.go"
Cohesion: 0.09
Nodes (26): Admin, Broker, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, Network (+18 more)

### Community 9 - "load.go"
Cohesion: 0.12
Nodes (19): scalingTier, TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config (+11 more)

### Community 10 - "newTestOps"
Cohesion: 0.11
Nodes (39): appUsers(), Ops, newTestOps(), TestAdditionalUsers(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDiagnostics() (+31 more)

### Community 11 - "capRunner"
Cohesion: 0.15
Nodes (20): capCall, capRunner, Manager, mgrOver(), TestCtrExecutorRefusesUnapprovedRuntime(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), TestManagerReachableProbesRuntimeThenCompose() (+12 more)

### Community 12 - "NewCluster"
Cohesion: 0.12
Nodes (37): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckOperatorNS(), TestCheckStorageClass(), TestResolveStorageClass(), NewCluster() (+29 more)

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
Cohesion: 0.09
Nodes (74): assertMode(), containsStr(), ctrCfg(), hasCall(), maskedKeys(), newCapMgr(), TestContainerRunningMatchesNameExactly(), TestManagerCheckDNSFailsLoudInHA() (+66 more)

### Community 17 - "cli/platform_test.go"
Cohesion: 0.27
Nodes (17): echoRunner(), App, runPlatform(), TestMultiPlatformNonInteractiveIsRefused(), TestMultiPlatformPromptRejectsBadAnswer(), TestMultiPlatformPromptSelects(), TestNoPlatformSectionIsRefused(), TestPlatformFlagAcceptsAbbreviations() (+9 more)

### Community 18 - "secrets_test.go"
Cohesion: 0.17
Nodes (20): GenSecrets(), TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), DockerRegistrySecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden() (+12 more)

### Community 19 - "dev.sh"
Cohesion: 0.18
Nodes (21): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+13 more)

### Community 20 - "prep_test.go"
Cohesion: 0.18
Nodes (22): saCfg(), adminCfg(), Cluster, labelCluster(), TestCreateNamespaceApplyFails(), TestCreateSecretsAdminOnly(), TestCreateSecretsPreflight(), TestCreateSecretsStopsOnPreflightFailure() (+14 more)

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
Cohesion: 0.13
Nodes (12): downloadErrTransport, fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrMatchTransport (+4 more)

### Community 26 - "newEchoMgr"
Cohesion: 0.09
Nodes (23): bytes.Buffer, NewManager(), Manager, newEchoMgr(), rootlessNoFileMgr(), setNoFile(), TestManagerCheckDryRun(), TestManagerDeletePodmanPurgeRootless() (+15 more)

### Community 27 - "Platform"
Cohesion: 0.08
Nodes (89): layer, opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/cobra.ShellCompDirective, github.com/spf13/pflag.FlagSet, availableSubs(), firstDiff() (+81 more)

### Community 28 - "BrokerCR"
Cohesion: 0.21
Nodes (17): strings.Builder, PodAffinityTerm, boolStr(), BrokerCR(), parseToleration(), sortedKeys(), TestParseToleration(), writeKeyValueEntry() (+9 more)

### Community 31 - "Command"
Cohesion: 0.06
Nodes (46): commandRules, keyValueEntries, decodeRuntime(), TestCommandUnmarshal(), Command, checkBinary(), CheckCommand(), checkFlagShape() (+38 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "context.Context"
Cohesion: 0.15
Nodes (4): context.Context, Cluster, Cluster, Cluster

### Community 34 - "completion_test.go"
Cohesion: 0.37
Nodes (12): runComplete(), TestAllowCommandOffersNoFiles(), TestDirFlagCompletesDirectories(), TestEnvFlagCompletesEnvFiles(), TestEnvFlagPrefixFilters(), TestEnvFlagWithPathDefersToShell(), TestNoArgsLeafOffersNoFiles(), TestPlatformFlagCompletes() (+4 more)

### Community 35 - "runRootWith"
Cohesion: 0.18
Nodes (19): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), TestEscalationIsRefusedEndToEnd(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb(), TestPathRuntimeIsRefused(), TestSmuggledSubcommandIsRefused() (+11 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.17
Nodes (10): TestCanIAnswerReadsTheLastLine(), NewTransport(), isCanI(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest() (+2 more)

### Community 39 - "Compose"
Cohesion: 0.17
Nodes (26): HealthCheck, NodeIdentity, Compose(), ContainerSecrets(), healthCmd(), Quadlet(), SecretPreflight(), SecretScript() (+18 more)

### Community 40 - ".releaseToBackup"
Cohesion: 0.30
Nodes (8): field(), TestField(), TestPrimaryRedundancyUp(), TestFieldLabelWithoutColon(), activity(), Ops, primaryRedundancyUp(), rdEnabledUp()

### Community 41 - "k8s/runtime_test.go"
Cohesion: 0.43
Nodes (7): TestClusterHonoursRuntime(), TestExecutorRefusesUnapprovedRuntime(), TestRuntimeDefaultArgvUnchanged(), TestTransportHonoursRuntime(), unapprovedCfg(), withLeading(), wrappedCfg()

### Community 42 - "TestServerCert"
Cohesion: 0.24
Nodes (9): TestPathHelpers(), TestServerCert(), concatFiles(), serverCertFile(), serverCertScript(), TestServerCertScript(), certPath(), cliArg() (+1 more)

### Community 43 - ".Run"
Cohesion: 0.12
Nodes (17): Echo, Exec, os/exec.Cmd, App, TestChildEnvNamesAreNotSystemVariables(), TestExecEchoesOnEveryMethod(), TestExecIsSilentWithoutVerbose(), TestExecVerboseAnnouncesEveryCommand() (+9 more)

### Community 44 - "Cluster"
Cohesion: 0.12
Nodes (12): bufio.Reader, Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels(), splitLabel() (+4 more)

### Community 45 - "Ops"
Cohesion: 0.20
Nodes (9): io.Reader, io.Writer, time.Duration, Ops, TestPromptYes(), TestPromptYesNo(), promptLine(), promptYes() (+1 more)

### Community 46 - "coverage_test.go"
Cohesion: 0.06
Nodes (48): matchCLI(), ranContains(), writeFile(), TestDiagnosticsMkdirError(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultVPNDisableError() (+40 more)

### Community 47 - "runner_test.go"
Cohesion: 0.14
Nodes (22): captureStdout(), helperCommand(), TestEchoDefaultWriter(), TestEchoOutput(), TestEchoOutputInput(), TestEchoRun(), TestEchoRunEnv(), TestEchoRunEnvNoEnv() (+14 more)

### Community 48 - ".Run"
Cohesion: 0.19
Nodes (16): runCtr(), TestConfigStepsDoNotLeakSecrets(), TestCtrConfigDryRun(), TestCtrConfirmDeclined(), TestCtrDiagnosticsDryRun(), TestCtrErrorPaths(), TestCtrExecCLIPathSeparator(), TestCtrRoleArgCount() (+8 more)

### Community 49 - "manager.go"
Cohesion: 0.15
Nodes (10): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), solaceRows() (+2 more)

### Community 50 - "scripts_test.go"
Cohesion: 0.18
Nodes (11): assertLeaderScript(), disableDefaultUsersScript(), parseVPNNames(), productKeysScript(), showVPNBareScript(), TestAssertLeaderScript(), TestDisableDefaultUsersScriptQuoting(), TestDisableDefaultVPNScript() (+3 more)

### Community 51 - "newRootCmd"
Cohesion: 0.09
Nodes (26): applyAliases(), TestAliasesDoNotCollide(), TestAliasesResolveToTheCanonicalCommand(), TestDangerousVerbsHaveNoBareAlias(), TestEveryAliasEntryIsLive(), TestGroupsRejectAnUnknownNoun(), TestStartStopHaveNoAlias(), TestAllowCommandIsRegisteredWhereItExecutes() (+18 more)

### Community 52 - "haCfg"
Cohesion: 0.20
Nodes (13): ProductKeyRoles(), haCfg(), TestHARoles(), TestProductKeyRoles(), TestRestartOrder(), TestOperatorImage(), showAllOutputs(), TestShowAll() (+5 more)

### Community 53 - "command_test.go"
Cohesion: 0.18
Nodes (11): TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults(), TestRuntimeExplicitValueSurvivesDefaults(), TestValidateProbeCommandAccepts() (+3 more)

### Community 54 - "HARoles"
Cohesion: 0.24
Nodes (6): Cluster, HARoles(), lbServiceName(), pvcName(), stsName(), TestResourceNames()

### Community 55 - "Test catalogue"
Cohesion: 0.18
Nodes (10): convert_test.go, Coverage, internal/convert, internal/render, internal/tools/vulnjudge, main_test.go, render_test.go, Running the tests (+2 more)

### Community 56 - ".LeaderLocal"
Cohesion: 0.29
Nodes (5): Ops, hostMatches(), roleName(), shortHost(), TestRoleName()

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
Cohesion: 0.23
Nodes (11): showCmd, disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), revertActivityConfigureScript(), revertActivityScript(), showRedundancyDetailScript(), showRedundancyScript() (+3 more)

### Community 63 - "internal/engine"
Cohesion: 0.67
Nodes (3): internal/engine, resolve_test.go, runner_test.go

### Community 65 - "Config"
Cohesion: 0.17
Nodes (14): tlsConfigured(), Config, orNone(), orValue(), setOrMissing(), setOrNone(), GenOperator(), joinYAMLDocs() (+6 more)

### Community 66 - "container/preflight_test.go"
Cohesion: 0.25
Nodes (7): TestComposeSecretEnvIsTheOnlyChildEnvironment(), TestComposeSecretEnvNamesCannotBeSystemVars(), TestPreflightFailureStopsLifecycle(), TestPreflightFailureStopsTheDeploy(), TestPreflightHintIsPlatformShaped(), TestPreflightIsPreviewableUnderDryRun(), TestPreflightRunsBeforeAnything()

### Community 68 - ".AdditionalUsers"
Cohesion: 0.16
Nodes (13): containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), TestValidName(), validCLILine(), validCLIPassword(), validName() (+5 more)

### Community 69 - ".DomainCerts"
Cohesion: 0.50
Nodes (4): domainCertsScript(), sortedKeys(), TestDomainCertsScriptSorted(), TestSortedKeys()

### Community 74 - "verify_ops.go"
Cohesion: 0.32
Nodes (6): TestHTTPStatusHelpers(), TestLastLines(), TestLastLinesEqualCount(), httpStatusLines(), isHTTP2xx(), lastLines()

### Community 75 - ".gatherNode"
Cohesion: 0.33
Nodes (4): gatherConfigsScript(), TestGatherConfigsScript(), TestZipConfigsScript(), zipConfigsScript()

## Knowledge Gaps
- **124 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+119 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `cli_test.go`, `render.go`, `Cluster`, `captureStdout`, `Manager`, `config.go`, `newTestOps`, `capRunner`, `NewCluster`, `convert_test.go`, `manager_test.go`, `secrets_test.go`, `prep_test.go`, `verify_local_test.go`, `newEchoMgr`, `BrokerCR`, `context.Context`, `recRunner`, `kubectlTransport`, `Compose`, `k8s/runtime_test.go`, `.Run`, `Cluster`, `Ops`, `haCfg`, `HARoles`, `containerTransport`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `containerTransport`, `captureStdout`, `testing.T`, `Manager`, `config.go`, `load.go`, `Compose`, `.Run`, `capRunner`, `convert_test.go`, `manager_test.go`, `manager.go`, `newEchoMgr`, `Command`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Why does `Manager` connect `Manager` to `bg`, `Config`, `.Run`, `Ops`, `manager.go`, `newEchoMgr`, `Platform`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Are the 10 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **Are the 10 inferred relationships involving `newCapMgr()` (e.g. with `NewManager()` and `TestComposeSecretEnvIsTheOnlyChildEnvironment()`) actually correct?**
  _`newCapMgr()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _124 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06201155283724091 - nodes in this community are weakly interconnected._