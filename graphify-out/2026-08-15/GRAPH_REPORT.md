# Graph Report - solace-k8-scripts  (2026-08-14)

## Corpus Check
- 80 files · ~165,868 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1661 nodes · 5007 edges · 64 communities (55 shown, 9 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 762 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `723b173b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- scripts.go
- ctrCfg
- context.Context
- cli_test.go
- config_test.go
- Commands
- Manager
- config.go
- testing.T
- newTestOps
- render.go
- NewCluster
- eqArgs
- convert_test.go
- Test catalogue
- captureStdout
- NewTransport
- Cluster
- dev.sh
- prep_test.go
- dev.ps1
- CLAUDE.md
- judge
- verify_local_test.go
- Role
- runRoot
- github.com/spf13/cobra.Command
- strings.Builder
- Go Module Definition
- CheckCommand
- Command reference
- manager.go
- Config
- .AdditionalUsers
- .resolveSecretRefs
- recRunner
- newRootCmd
- Compose
- Platform
- haCfg
- kubectlTransport
- load.go
- secrets_test.go
- Cluster
- coverage_test.go
- .Run
- cliScriptPath
- execguard_test.go
- Cluster
- scripts_test.go
- .releaseToBackup
- .LeaderLocal
- containerTransport
- opRunner
- Command
- verify_ops.go
- filterLines
- TestK8sLoginOutcomes
- .ServerCert
- ranContains
- .Preflight

## God Nodes (most connected - your core abstractions)
1. `Commands` - 115 edges
2. `Role` - 81 edges
3. `ctrCfg()` - 79 edges
4. `Config` - 75 edges
5. `newTestOps()` - 73 edges
6. `newCapMgr()` - 72 edges
7. `bg()` - 70 edges
8. `Manager` - 53 edges
9. `matchCLI()` - 52 edges
10. `NewCluster()` - 50 edges

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

## Communities (64 total, 9 thin omitted)

### Community 0 - "bg"
Cohesion: 0.06
Nodes (97): time.Duration, Ops, New(), TestNewDefaults(), Transport, TestCtrManagerConfirmWiring(), containerWhat(), ctrLogin() (+89 more)

### Community 1 - "scripts.go"
Cohesion: 0.16
Nodes (15): showCmd, disableDefaultUsersScript(), disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), revertActivityConfigureScript(), revertActivityScript(), showRedundancyDetailScript() (+7 more)

### Community 2 - "ctrCfg"
Cohesion: 0.07
Nodes (88): fileExists(), NewManager(), assertMode(), containsStr(), ctrCfg(), Manager, hasCall(), maskedKeys() (+80 more)

### Community 3 - "context.Context"
Cohesion: 0.14
Nodes (11): context.Context, HARoles(), lbServiceName(), podName(), pvcName(), RestartOrder(), stsName(), TestResourceNames() (+3 more)

### Community 4 - "cli_test.go"
Cohesion: 0.17
Nodes (26): capture(), captureStderr(), runCtr(), TestBashEnvGivenToEnvFlag(), TestConvertErrorPaths(), TestConvertRoundTrip(), TestConvertToFile(), TestConvertToStdout() (+18 more)

### Community 5 - "config_test.go"
Cohesion: 0.06
Nodes (64): assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, haNodesConfig(), minimalK8s(), TestApplyDefaultsDocker(), TestApplyDefaultsK8s() (+56 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (115): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+107 more)

### Community 8 - "config.go"
Cohesion: 0.11
Nodes (22): Container, ContainerSecurity, DockerConfig, DomainCerts, Image, Network, Node, Nodes (+14 more)

### Community 9 - "testing.T"
Cohesion: 0.10
Nodes (41): bytes.Buffer, testing.T, decodeRuntime(), TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors() (+33 more)

### Community 10 - "newTestOps"
Cohesion: 0.12
Nodes (33): appUsers(), Ops, newTestOps(), TestAdditionalUsers(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDisableDefaultUsers() (+25 more)

### Community 11 - "render.go"
Cohesion: 0.14
Nodes (21): Config, NodeIdentity, containerSecretSpecs(), cut(), EnvFile(), EnvPairs(), escapePercent(), groupKey() (+13 more)

### Community 12 - "NewCluster"
Cohesion: 0.13
Nodes (33): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster() (+25 more)

### Community 13 - "eqArgs"
Cohesion: 0.17
Nodes (24): Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived(), TestOperatorNSExplicit() (+16 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (52): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+44 more)

### Community 15 - "Test catalogue"
Cohesion: 0.04
Nodes (47): allowcommand_test.go, broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go (+39 more)

### Community 16 - "captureStdout"
Cohesion: 0.20
Nodes (21): opCall, captureStdout(), failDisableDefaultUsersUpload(), k8sConfigAllRunner(), k8sUpOutputHook(), loadDirect(), opArgvMatch(), opFailOn() (+13 more)

### Community 17 - "NewTransport"
Cohesion: 0.15
Nodes (20): capCall, capRunner, Manager, mgrOver(), TestCtrExecutorRefusesUnapprovedRuntime(), TestCtrTransportHonoursRuntime(), TestManagerHonoursRuntime(), TestManagerReachableProbesRuntimeThenCompose() (+12 more)

### Community 18 - "Cluster"
Cohesion: 0.13
Nodes (13): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels() (+5 more)

### Community 19 - "dev.sh"
Cohesion: 0.19
Nodes (20): finish(), log_init(), main(), NO_COLOR, dev.sh script, build_one(), cap(), die() (+12 more)

### Community 20 - "prep_test.go"
Cohesion: 0.17
Nodes (23): saCfg(), TestRestartRolling(), adminCfg(), Cluster, labelCluster(), TestCreateSecretsAdminOnly(), TestCreateSecretsAllThree(), TestCreateSecretsPreflight() (+15 more)

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

### Community 26 - "runRoot"
Cohesion: 0.14
Nodes (24): TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), firstLine(), App, runRoot(), runRootWith(), TestConvertParseError(), TestCtrRoleHelp() (+16 more)

### Community 27 - "github.com/spf13/cobra.Command"
Cohesion: 0.07
Nodes (84): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, os.File, TestConfirmFlagShortcuts() (+76 more)

### Community 28 - "strings.Builder"
Cohesion: 0.17
Nodes (18): strings.Builder, PodAffinityTerm, Cluster, boolStr(), BrokerCR(), parseToleration(), sortedKeys(), TestParseToleration() (+10 more)

### Community 31 - "CheckCommand"
Cohesion: 0.19
Nodes (15): commandRules, checkBinary(), CheckCommand(), checkFlagShape(), checkToken(), clusterRules(), composeRules(), execBase() (+7 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "manager.go"
Cohesion: 0.24
Nodes (8): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), Manager

### Community 34 - "Config"
Cohesion: 0.25
Nodes (9): Admin, Replication, Scaling, TLS, Config, GenOperator(), RenderOperator(), watchNamespace() (+1 more)

### Community 35 - ".AdditionalUsers"
Cohesion: 0.18
Nodes (11): containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), validCLILine(), validCLIPassword(), Ops, additionalUsersScript() (+3 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.18
Nodes (9): NewTransport(), isCanI(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest(), recRunner (+1 more)

### Community 38 - "newRootCmd"
Cohesion: 0.13
Nodes (20): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandIsRegisteredOnPlatformTrees(), TestAllowCommandIsRepeatable(), TestEscalationIsRefusedEndToEnd(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb(), TestPathRuntimeIsRefused(), TestSmuggledSubcommandIsRefused() (+12 more)

### Community 39 - "Compose"
Cohesion: 0.19
Nodes (18): Compose(), ContainerSecrets(), ContainerSecret, SecretPreflight(), SecretScript(), shQuote(), splitPair(), envLines() (+10 more)

### Community 40 - "Platform"
Cohesion: 0.20
Nodes (13): keyValueEntries, Platform, TestValidateUnknownPlatform(), foldToEnvVar(), Config, missingErr(), platformKey(), requireAll() (+5 more)

### Community 41 - "haCfg"
Cohesion: 0.20
Nodes (14): haCfg(), TestHARoles(), TestProductKeyRoles(), TestShowAll(), TestShowAllWrapsGetError(), TestCreateSecretsFailsWithoutAdminFields(), TestDeleteSecretsSkipsUnconfiguredAdminSecret(), TestClusterHonoursRuntime() (+6 more)

### Community 43 - "load.go"
Cohesion: 0.19
Nodes (12): TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config, parseError() (+4 more)

### Community 44 - "secrets_test.go"
Cohesion: 0.17
Nodes (17): TestUpdateServerCertSecret(), AdminSecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden(), decodeDataValue(), TestAdminSecretDecodes(), TestAdminSecretErrors() (+9 more)

### Community 46 - "coverage_test.go"
Cohesion: 0.07
Nodes (43): matchCLI(), writeFile(), TestDiagnosticsMkdirError(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultVPNDisableError(), TestDomainCertsBadFilename() (+35 more)

### Community 47 - ".Run"
Cohesion: 0.18
Nodes (11): Echo, Exec, os/exec.Cmd, TestChildEnvNamesAreNotSystemVariables(), command(), MaskEnv(), Quote(), quoteTok() (+3 more)

### Community 48 - "cliScriptPath"
Cohesion: 0.20
Nodes (12): TestDiagnostics(), TestDomainCerts(), TestPathHelpers(), TestServerCert(), TestValidName(), validName(), TestDisableDefaultUsersShowVPNError(), TestReleaseToBackupReleasedTimeout() (+4 more)

### Community 49 - "execguard_test.go"
Cohesion: 0.19
Nodes (15): decodeStrict(), Config, guardCommandOf(), guardConfig(), setGuardCommand(), TestAllowCommandIsNotASchemaKey(), TestAllowCommandsAccepts(), TestAllowCommandsRejects() (+7 more)

### Community 50 - "Cluster"
Cohesion: 0.27
Nodes (5): Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

### Community 51 - "scripts_test.go"
Cohesion: 0.18
Nodes (12): domainCertsScript(), parseVPNNames(), productKeysScript(), sortedKeys(), TestDomainCertsScriptSorted(), TestGatherConfigsScript(), TestParseVPNNames(), TestParseVPNNamesNoSeparator() (+4 more)

### Community 52 - ".releaseToBackup"
Cohesion: 0.28
Nodes (8): field(), TestField(), TestPrimaryRedundancyUp(), TestFieldLabelWithoutColon(), activity(), Ops, primaryRedundancyUp(), rdEnabledUp()

### Community 53 - ".LeaderLocal"
Cohesion: 0.23
Nodes (7): assertLeaderScript(), TestAssertLeaderScript(), Ops, hostMatches(), roleName(), shortHost(), TestRoleName()

### Community 57 - "verify_ops.go"
Cohesion: 0.32
Nodes (6): TestHTTPStatusHelpers(), TestLastLines(), TestLastLinesEqualCount(), httpStatusLines(), isHTTP2xx(), lastLines()

### Community 59 - "TestK8sLoginOutcomes"
Cohesion: 0.32
Nodes (8): runStandalone(), runStatusStderr(), TestEnvFileLookup(), TestK8sErrorPaths(), TestK8sLoginOutcomes(), TestK8sStandaloneDryRun(), TestSecretsNeverEchoed(), writeStandaloneEnv()

### Community 61 - ".ServerCert"
Cohesion: 0.47
Nodes (4): concatFiles(), serverCertFile(), serverCertScript(), TestServerCertScript()

### Community 62 - "ranContains"
Cohesion: 0.33
Nodes (6): ranContains(), TestExecCLI(), TestDisableDefaultVPNShowError(), TestExecCLIRunError(), TestRemoveCLIWarnsOnFailure(), TestRemoveDomainCertsRunCLIError()

## Knowledge Gaps
- **169 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+164 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **9 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `ctrCfg`, `context.Context`, `Manager`, `config.go`, `newTestOps`, `render.go`, `NewCluster`, `convert_test.go`, `captureStdout`, `NewTransport`, `Cluster`, `prep_test.go`, `verify_local_test.go`, `github.com/spf13/cobra.Command`, `strings.Builder`, `recRunner`, `Compose`, `haCfg`, `kubectlTransport`, `secrets_test.go`, `Cluster`, `containerTransport`?**
  _High betweenness centrality (0.074) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `manager.go`, `ctrCfg`, `config_test.go`, `Manager`, `config.go`, `Compose`, `load.go`, `convert_test.go`, `captureStdout`, `execguard_test.go`, `NewTransport`, `containerTransport`, `github.com/spf13/cobra.Command`, `CheckCommand`?**
  _High betweenness centrality (0.062) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `bg`, `scripts.go`, `.AdditionalUsers`, `context.Context`, `Manager`, `kubectlTransport`, `render.go`, `cliScriptPath`, `Cluster`, `scripts_test.go`, `.releaseToBackup`, `.LeaderLocal`, `containerTransport`, `verify_local_test.go`, `verify_ops.go`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.036) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _169 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06365720331511197 - nodes in this community are weakly interconnected._
- **Should `ctrCfg` be split into smaller, more focused modules?**
  _Cohesion score 0.07340823970037454 - nodes in this community are weakly interconnected._