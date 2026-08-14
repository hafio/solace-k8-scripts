# Graph Report - solace-k8-scripts  (2026-08-14)

## Corpus Check
- 80 files · ~164,369 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1658 nodes · 4996 edges · 65 communities (58 shown, 7 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 758 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c417e533`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bg
- scripts.go
- ctrCfg
- context.Context
- cli_test.go
- testing.T
- Commands
- Manager
- Config
- runner_test.go
- newTestOps
- render.go
- NewCluster
- eqArgs
- convert_test.go
- Test catalogue
- captureStdout
- Ops
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
- Platform
- Command reference
- manager.go
- RenderOperator
- .AdditionalUsers
- .resolveSecretRefs
- recRunner
- newRootCmd
- Quadlet
- newEchoMgr
- haCfg
- kubectlTransport
- NewManager
- secrets_test.go
- Cluster
- coverage_test.go
- .Run
- cliScriptPath
- command_test.go
- Cluster
- scripts_test.go
- .releaseToBackup
- .LeaderLocal
- containerTransport
- opRunner
- Ops
- verify_ops.go
- .redundancyLocalPrimary
- writeStandaloneEnv
- BrokerCR
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
- `matchCLI()` --calls--> `cliArg()`  [INFERRED]
  internal/broker/broker_test.go → internal/broker/transport.go
- `TestAdditionalUsersRunCLITransportError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestDisableDefaultUsersShowVPNError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestDisableDefaultVPNShowError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestReleaseToBackupReleasedTimeout()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (65 total, 7 thin omitted)

### Community 0 - "bg"
Cohesion: 0.08
Nodes (92): TestCtrManagerConfirmWiring(), containerWhat(), ctrLogin(), ctrManager(), ctrOps(), App, opCtrCheck(), opCtrCLI() (+84 more)

### Community 1 - "scripts.go"
Cohesion: 0.21
Nodes (13): showCmd, disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), revertActivityConfigureScript(), revertActivityScript(), showRedundancyDetailScript(), showRedundancyScript() (+5 more)

### Community 2 - "ctrCfg"
Cohesion: 0.10
Nodes (68): fileExists(), assertMode(), containsStr(), ctrCfg(), hasCall(), maskedKeys(), newCapMgr(), TestContainerRunningMatchesNameExactly() (+60 more)

### Community 3 - "context.Context"
Cohesion: 0.14
Nodes (9): context.Context, lbServiceName(), podName(), stsName(), TestResourceNames(), Cluster, filterLines(), Cluster (+1 more)

### Community 4 - "cli_test.go"
Cohesion: 0.17
Nodes (25): capture(), captureStderr(), runCtr(), TestBashEnvGivenToEnvFlag(), TestConvertErrorPaths(), TestConvertRoundTrip(), TestConvertToFile(), TestConvertToStdout() (+17 more)

### Community 5 - "testing.T"
Cohesion: 0.08
Nodes (65): testing.T, assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, haNodesConfig(), minimalK8s(), TestApplyDefaultsDocker() (+57 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (115): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+107 more)

### Community 8 - "Config"
Cohesion: 0.10
Nodes (25): Admin, Container, ContainerSecurity, DockerConfig, DomainCerts, Image, Network, Node (+17 more)

### Community 9 - "runner_test.go"
Cohesion: 0.12
Nodes (26): bytes.Buffer, captureResolve(), TestExecEchoesOnEveryMethod(), TestExecEchoesResolvedPath(), TestExecMissingBinaryIsActionable(), TestExecRefusesCurrentDirectoryResolution(), captureStdout(), helperCommand() (+18 more)

### Community 10 - "newTestOps"
Cohesion: 0.12
Nodes (33): appUsers(), Ops, newTestOps(), TestAdditionalUsers(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestAdditionalUsersReportsExistingUser(), TestDisableDefaultUsers() (+25 more)

### Community 11 - "render.go"
Cohesion: 0.13
Nodes (20): Config, NodeIdentity, Compose(), containerSecretSpecs(), cut(), EnvFile(), EnvPairs(), ContainerSecret (+12 more)

### Community 12 - "NewCluster"
Cohesion: 0.13
Nodes (32): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster() (+24 more)

### Community 13 - "eqArgs"
Cohesion: 0.15
Nodes (26): Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived(), TestOperatorNSExplicit() (+18 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (52): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+44 more)

### Community 15 - "Test catalogue"
Cohesion: 0.04
Nodes (47): allowcommand_test.go, broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go (+39 more)

### Community 16 - "captureStdout"
Cohesion: 0.18
Nodes (23): opCall, captureStdout(), failDisableDefaultUsersUpload(), k8sConfigAllRunner(), k8sUpOutputHook(), loadDirect(), opArgvMatch(), opFailOn() (+15 more)

### Community 17 - "Ops"
Cohesion: 0.12
Nodes (15): capCall, capRunner, time.Duration, Ops, New(), TestNewDefaults(), Transport, NewTransport() (+7 more)

### Community 18 - "Cluster"
Cohesion: 0.13
Nodes (13): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels() (+5 more)

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
Cohesion: 0.14
Nodes (11): downloadErrTransport, fakeTransport, recDownload, recOutput, recRun, recUpload, recUploadFile, runErrMatchTransport (+3 more)

### Community 26 - "runRoot"
Cohesion: 0.14
Nodes (24): TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), firstLine(), App, runRoot(), runRootWith(), TestConvertParseError(), TestCtrRoleHelp() (+16 more)

### Community 27 - "github.com/spf13/cobra.Command"
Cohesion: 0.07
Nodes (84): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, os.File, TestConfirmFlagShortcuts() (+76 more)

### Community 28 - "strings.Builder"
Cohesion: 0.16
Nodes (20): WeightedNodeTerm, strings.Builder, NodeAffinity, NodeMatchExpr, Placement, PodAffinityTerm, validateMatchExprs(), validatePlacementAffinity() (+12 more)

### Community 31 - "Platform"
Cohesion: 0.06
Nodes (51): commandRules, keyValueEntries, Platform, TestApplyBridgePortDefaults(), TestDefaultK8sPortsMatchesOperator(), TestValidateUnknownPlatform(), checkBinary(), CheckCommand() (+43 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "manager.go"
Cohesion: 0.24
Nodes (8): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), Manager

### Community 34 - "RenderOperator"
Cohesion: 0.60
Nodes (4): GenOperator(), RenderOperator(), watchNamespace(), operatorTmplVars

### Community 35 - ".AdditionalUsers"
Cohesion: 0.29
Nodes (7): containsAnyFold(), TestContainsAnyFold(), validCLILine(), validCLIPassword(), additionalUsersScript(), TestAdditionalUsersScript(), AdditionalUser

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.17
Nodes (10): TestCanIAnswerReadsTheLastLine(), NewTransport(), isCanI(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload(), TestTransportUploadQuotesDest() (+2 more)

### Community 38 - "newRootCmd"
Cohesion: 0.13
Nodes (19): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandIsRegisteredOnPlatformTrees(), TestAllowCommandIsRepeatable(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb(), TestPathRuntimeIsRefused(), TestSmuggledSubcommandIsRefused(), writeRuntimeEnv() (+11 more)

### Community 39 - "Quadlet"
Cohesion: 0.21
Nodes (20): emitCtrArtifact(), ContainerSecrets(), escapePercent(), Quadlet(), quadletEscape(), SecretPreflight(), SecretScript(), shQuote() (+12 more)

### Community 40 - "newEchoMgr"
Cohesion: 0.12
Nodes (16): Manager, newEchoMgr(), TestManagerCheckDryRun(), TestManagerDeletePodmanPurgeRootless(), TestManagerDeployDockerDryRunMasksSecretEnv(), TestManagerDeployPodmanDryRunHidesSecretBytes(), TestManagerDeployPodmanDryRunSkipsWrite(), TestManagerDockerCheckProbesCompose() (+8 more)

### Community 41 - "haCfg"
Cohesion: 0.18
Nodes (16): RestartOrder(), haCfg(), TestHARoles(), TestProductKeyRoles(), TestRestartOrder(), TestShowAll(), TestShowAllWrapsGetError(), TestCreateSecretsFailsWithoutAdminFields() (+8 more)

### Community 43 - "NewManager"
Cohesion: 0.21
Nodes (14): NewManager(), TestManagerLogsCLIShell(), TestManagerNilSinks(), TestPreflightRunsBeforeAnything(), Manager, mgrOver(), TestCtrExecutorRefusesUnapprovedRuntime(), TestCtrRuntimeDefaultArgvUnchanged() (+6 more)

### Community 44 - "secrets_test.go"
Cohesion: 0.16
Nodes (18): TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden(), decodeDataValue(), TestAdminSecretDecodes() (+10 more)

### Community 46 - "coverage_test.go"
Cohesion: 0.07
Nodes (43): matchCLI(), writeFile(), TestDiagnosticsMkdirError(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultVPNDisableError(), TestDomainCertsBadFilename() (+35 more)

### Community 47 - ".Run"
Cohesion: 0.18
Nodes (11): Echo, Exec, os/exec.Cmd, TestChildEnvNamesAreNotSystemVariables(), command(), MaskEnv(), Quote(), quoteTok() (+3 more)

### Community 48 - "cliScriptPath"
Cohesion: 0.20
Nodes (12): TestDiagnostics(), TestDomainCerts(), TestPathHelpers(), TestServerCert(), TestValidName(), validName(), TestDisableDefaultUsersShowVPNError(), TestReleaseToBackupReleasedTimeout() (+4 more)

### Community 49 - "command_test.go"
Cohesion: 0.15
Nodes (14): decodeRuntime(), TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshal(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults() (+6 more)

### Community 50 - "Cluster"
Cohesion: 0.27
Nodes (5): Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

### Community 51 - "scripts_test.go"
Cohesion: 0.16
Nodes (12): assertLeaderScript(), disableDefaultUsersScript(), parseVPNNames(), productKeysScript(), TestAssertLeaderScript(), TestDisableDefaultUsersScriptQuoting(), TestGatherConfigsScript(), TestParseVPNNames() (+4 more)

### Community 52 - ".releaseToBackup"
Cohesion: 0.36
Nodes (4): countContains(), TestCountContains(), activity(), Ops

### Community 53 - ".LeaderLocal"
Cohesion: 0.29
Nodes (5): Ops, hostMatches(), roleName(), shortHost(), TestRoleName()

### Community 56 - "Ops"
Cohesion: 0.29
Nodes (6): Ops, domainCertsScript(), removeDomainCertsScript(), sortedKeys(), TestDomainCertsScriptSorted(), TestSortedKeys()

### Community 57 - "verify_ops.go"
Cohesion: 0.32
Nodes (6): TestHTTPStatusHelpers(), TestLastLines(), TestLastLinesEqualCount(), httpStatusLines(), isHTTP2xx(), lastLines()

### Community 58 - ".redundancyLocalPrimary"
Cohesion: 0.43
Nodes (6): field(), TestField(), TestPrimaryRedundancyUp(), TestFieldLabelWithoutColon(), primaryRedundancyUp(), rdEnabledUp()

### Community 59 - "writeStandaloneEnv"
Cohesion: 0.38
Nodes (7): runStandalone(), runStatusStderr(), TestEnvFileLookup(), TestK8sErrorPaths(), TestK8sStandaloneDryRun(), TestSecretsNeverEchoed(), writeStandaloneEnv()

### Community 60 - "BrokerCR"
Cohesion: 0.38
Nodes (5): Cluster, pvcName(), boolStr(), BrokerCR(), writeSecurity()

### Community 61 - ".ServerCert"
Cohesion: 0.47
Nodes (4): concatFiles(), serverCertFile(), serverCertScript(), TestServerCertScript()

### Community 62 - "ranContains"
Cohesion: 0.33
Nodes (6): ranContains(), TestExecCLI(), TestDisableDefaultVPNShowError(), TestExecCLIRunError(), TestRemoveCLIWarnsOnFailure(), TestRemoveDomainCertsRunCLIError()

## Knowledge Gaps
- **169 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+164 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **7 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `ctrCfg`, `context.Context`, `Manager`, `newTestOps`, `render.go`, `NewCluster`, `convert_test.go`, `captureStdout`, `Ops`, `Cluster`, `prep_test.go`, `verify_local_test.go`, `github.com/spf13/cobra.Command`, `strings.Builder`, `RenderOperator`, `recRunner`, `Quadlet`, `newEchoMgr`, `haCfg`, `kubectlTransport`, `NewManager`, `secrets_test.go`, `Cluster`, `containerTransport`, `BrokerCR`?**
  _High betweenness centrality (0.093) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `manager.go`, `ctrCfg`, `testing.T`, `Manager`, `Config`, `newEchoMgr`, `Quadlet`, `NewManager`, `convert_test.go`, `captureStdout`, `Ops`, `containerTransport`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.055) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `bg`, `scripts.go`, `context.Context`, `Manager`, `render.go`, `Ops`, `Cluster`, `verify_local_test.go`, `github.com/spf13/cobra.Command`, `.AdditionalUsers`, `Quadlet`, `haCfg`, `kubectlTransport`, `cliScriptPath`, `scripts_test.go`, `.releaseToBackup`, `.LeaderLocal`, `containerTransport`, `Ops`, `verify_ops.go`, `.redundancyLocalPrimary`, `BrokerCR`?**
  _High betweenness centrality (0.031) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _169 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.07525195968645017 - nodes in this community are weakly interconnected._
- **Should `ctrCfg` be split into smaller, more focused modules?**
  _Cohesion score 0.09974424552429667 - nodes in this community are weakly interconnected._