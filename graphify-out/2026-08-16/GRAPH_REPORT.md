# Graph Report - solace-k8-scripts  (2026-08-15)

## Corpus Check
- 82 files · ~170,951 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1688 nodes · 5076 edges · 59 communities (51 shown, 8 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 779 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c27a05aa`
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
- Load
- NewCluster
- eqArgs
- convert_test.go
- Test catalogue
- captureStdout
- command_test.go
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
- render.go
- Go Module Definition
- Platform
- Command reference
- manager.go
- Image
- .AdditionalUsers
- .resolveSecretRefs
- recRunner
- newRootCmd
- Compose
- writeFile
- haCfg
- kubectlTransport
- load.go
- secrets_test.go
- Cluster
- coverage_test.go
- .Run
- cliScriptPath
- Cluster
- scripts_test.go
- .releaseToBackup
- .LeaderLocal
- containerTransport
- verify_ops.go
- filterLines
- .ServerCert
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
10. `Platform` - 50 edges

## Surprising Connections (you probably didn't know these)
- `TestFieldLabelWithoutColon()` --calls--> `field()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker.go
- `activity()` --calls--> `countContains()`  [INFERRED]
  internal/broker/verify_ops.go → internal/broker/broker.go
- `matchCLI()` --calls--> `cliArg()`  [INFERRED]
  internal/broker/broker_test.go → internal/broker/transport.go
- `TestAdditionalUsersRunCLITransportError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go
- `TestServerCertRunCLIError()` --calls--> `matchCLI()`  [INFERRED]
  internal/broker/coverage_test.go → internal/broker/broker_test.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Legacy Bash Script Family** — bash_000_env_sh, bash_010_deploy_operator_sh, bash_020_deploy_broker_sh, bash_059_execute_cli_sh [EXTRACTED 1.00]
- **Solace Go CLI Architecture** — internal_config, internal_engine, internal_render, internal_broker, internal_k8s, internal_cli [EXTRACTED 1.00]

## Communities (59 total, 8 thin omitted)

### Community 0 - "bg"
Cohesion: 0.07
Nodes (95): time.Duration, Ops, New(), TestNewDefaults(), TestCtrManagerConfirmWiring(), containerWhat(), ctrLogin(), ctrManager() (+87 more)

### Community 1 - "scripts.go"
Cohesion: 0.21
Nodes (13): showCmd, disableDefaultVPNScript(), noReleaseActivityScript(), releaseActivityScript(), revertActivityConfigureScript(), revertActivityScript(), showRedundancyDetailScript(), showRedundancyScript() (+5 more)

### Community 2 - "ctrCfg"
Cohesion: 0.05
Nodes (107): capCall, capRunner, bytes.Buffer, fileExists(), NewManager(), assertMode(), containsStr(), ctrCfg() (+99 more)

### Community 3 - "context.Context"
Cohesion: 0.14
Nodes (11): context.Context, HARoles(), lbServiceName(), podName(), pvcName(), RestartOrder(), stsName(), TestResourceNames() (+3 more)

### Community 4 - "cli_test.go"
Cohesion: 0.10
Nodes (57): TestAllowCommandRejectedWhereNothingExecutes(), TestAllowCommandRejectsBadValues(), capture(), captureStderr(), firstLine(), App, runCtr(), runRoot() (+49 more)

### Community 5 - "testing.T"
Cohesion: 0.07
Nodes (65): testing.T, assertContainerBlockDefaults(), assertContainerScaling(), envTree(), Config, haNodesConfig(), TestApplyBridgePortDefaults(), TestApplyDefaultsDocker() (+57 more)

### Community 6 - "Commands"
Cohesion: 0.02
Nodes (115): Commands, solace, solace convert, solace docker, solace docker check, solace docker cli, solace docker config, solace docker config disable-default-users (+107 more)

### Community 8 - "Config"
Cohesion: 0.11
Nodes (28): Container, ContainerSecurity, DockerConfig, DomainCerts, Network, Node, Nodes, Operator (+20 more)

### Community 9 - "runner_test.go"
Cohesion: 0.13
Nodes (25): captureResolve(), TestExecEchoesOnEveryMethod(), TestExecEchoesResolvedPath(), TestExecMissingBinaryIsActionable(), TestExecRefusesCurrentDirectoryResolution(), captureStdout(), helperCommand(), TestEchoDefaultWriter() (+17 more)

### Community 10 - "newTestOps"
Cohesion: 0.15
Nodes (25): Ops, newTestOps(), TestAdditionalUsersEmpty(), TestAdditionalUsersRejectsBadValues(), TestDisableDefaultUsersNoVPNs(), TestDomainCertsEmptySkips(), TestDomainCertsRejectsBadName(), TestExecCLI() (+17 more)

### Community 11 - "Load"
Cohesion: 0.26
Nodes (14): minimalK8s(), TestLoadBashEnvFileHint(), TestLoadNotYAMLHint(), TestLoadParseError(), TestLoadResolvesSecretRefs(), TestLoadSecretRefErrors(), TestLoadSuccess(), TestLoadUnknownField() (+6 more)

### Community 12 - "NewCluster"
Cohesion: 0.13
Nodes (32): TestCheckAbortsWhenUnreachable(), TestCheckDryRun(), TestCheckEnvNoSecretLeak(), TestCheckEnvSparseConfig(), TestCheckStorageClass(), TestReachable(), TestResolveStorageClass(), NewCluster() (+24 more)

### Community 13 - "eqArgs"
Cohesion: 0.15
Nodes (26): Cluster, newCluster(), TestApplyOnStdin(), TestDeleteStdin(), TestOperatorNSDefaultOnError(), TestOperatorNSDefaultWhenAbsent(), TestOperatorNSDerived(), TestOperatorNSExplicit() (+18 more)

### Community 14 - "convert_test.go"
Cohesion: 0.07
Nodes (53): doc, Result, vars, boolOf(), commentSafe(), Convert(), countMarkers(), emitYAML() (+45 more)

### Community 15 - "Test catalogue"
Cohesion: 0.04
Nodes (48): allowcommand_test.go, broker_test.go, check_test.go, cli_test.go, cluster_test.go, command_test.go, commanddoc_test.go, config_test.go (+40 more)

### Community 16 - "captureStdout"
Cohesion: 0.14
Nodes (23): opCall, opRunner, captureStdout(), failDisableDefaultUsersUpload(), k8sConfigAllRunner(), k8sUpOutputHook(), loadDirect(), opArgvMatch() (+15 more)

### Community 17 - "command_test.go"
Cohesion: 0.18
Nodes (11): TestCommandArgsDoesNotAliasCommand(), TestCommandNameAndArgs(), TestCommandString(), TestCommandUnmarshalPropagatesDecodeErrors(), TestCommandUnmarshalRejectsOtherKinds(), TestRuntimeDefaults(), TestRuntimeExplicitValueSurvivesDefaults(), TestValidateProbeCommandAccepts() (+3 more)

### Community 18 - "Cluster"
Cohesion: 0.13
Nodes (14): bufio.Reader, GenSecrets(), Cluster, isBuiltinLabel(), joinManifests(), namespaceManifest(), roleName(), rolePlacementLabels() (+6 more)

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
Nodes (84): opFunc, roleOpFunc, github.com/spf13/cobra.Command, github.com/spf13/pflag.FlagSet, io.Reader, io.Writer, os.File, TestConfirmFlagShortcuts() (+76 more)

### Community 28 - "render.go"
Cohesion: 0.12
Nodes (31): strings.Builder, PodAffinityTerm, Cluster, boolStr(), BrokerCR(), containerSecretSpecs(), cut(), escapePercent() (+23 more)

### Community 31 - "Platform"
Cohesion: 0.06
Nodes (47): commandRules, keyValueEntries, decodeRuntime(), TestCommandUnmarshal(), Command, Platform, checkBinary(), CheckCommand() (+39 more)

### Community 32 - "Command reference"
Cohesion: 0.50
Nodes (3): Command reference, Global flags, Tree

### Community 33 - "manager.go"
Cohesion: 0.20
Nodes (10): defaultGenPSK(), exactName(), orNone(), platformTitle(), replacePSKLine(), secretSummary(), setOrMissing(), TestDefaultGenPSK() (+2 more)

### Community 35 - ".AdditionalUsers"
Cohesion: 0.15
Nodes (14): Admin, containsAnyFold(), countContains(), TestContainsAnyFold(), TestCountContains(), TestValidName(), validCLILine(), validCLIPassword() (+6 more)

### Community 36 - ".resolveSecretRefs"
Cohesion: 0.47
Nodes (3): secretRef, Config, unsetOrEmpty()

### Community 37 - "recRunner"
Cohesion: 0.16
Nodes (11): Runner, TestCanIAnswerReadsTheLastLine(), NewTransport(), isCanI(), TestTransportCopy(), TestTransportEchoHidesUploadBody(), TestTransportExecArgs(), TestTransportUpload() (+3 more)

### Community 38 - "newRootCmd"
Cohesion: 0.13
Nodes (20): TestAllowCommandApprovesAWrappedRuntime(), TestAllowCommandIsRegisteredOnPlatformTrees(), TestAllowCommandIsRepeatable(), TestEscalationIsRefusedEndToEnd(), TestGenPathNeverExecutes(), TestHostileRuntimeIsRefusedByEveryVerb(), TestPathRuntimeIsRefused(), TestSmuggledSubcommandIsRefused() (+12 more)

### Community 39 - "Compose"
Cohesion: 0.13
Nodes (30): emitCtrArtifact(), Config, NodeIdentity, Compose(), ContainerSecrets(), EnvFile(), EnvPairs(), ContainerSecret (+22 more)

### Community 40 - "writeFile"
Cohesion: 0.40
Nodes (5): writeFile(), TestDiagnosticsMkdirError(), TestServerCertCAReadError(), TestServerCertRunCLIError(), TestServerCertUploadError()

### Community 41 - "haCfg"
Cohesion: 0.20
Nodes (14): haCfg(), TestHARoles(), TestProductKeyRoles(), TestShowAll(), TestShowAllWrapsGetError(), TestCreateSecretsFailsWithoutAdminFields(), TestDeleteSecretsSkipsUnconfiguredAdminSecret(), TestClusterHonoursRuntime() (+6 more)

### Community 43 - "load.go"
Cohesion: 0.25
Nodes (10): applyBridgePortDefaults(), applyContainerBlockDefaults(), defaultContainerPorts(), defaultK8sPorts(), Config, parseError(), setDefault(), setDefaultCmd() (+2 more)

### Community 44 - "secrets_test.go"
Cohesion: 0.15
Nodes (18): TestCreateSecretsAllThree(), TestUpdateServerCertSecret(), AdminSecret(), dockerRegistrySecret(), operatorRegcred(), checkGolden(), decodeDataValue(), TestAdminSecretDecodes() (+10 more)

### Community 46 - "coverage_test.go"
Cohesion: 0.07
Nodes (46): matchCLI(), ranContains(), TestDiagnosticsRunError(), TestDiagnosticsTwoRolesNoBundle(), TestDisableDefaultUsersDisableError(), TestDisableDefaultUsersShowVPNError(), TestDisableDefaultVPNDisableError(), TestDisableDefaultVPNShowError() (+38 more)

### Community 47 - ".Run"
Cohesion: 0.18
Nodes (11): Echo, Exec, os/exec.Cmd, TestChildEnvNamesAreNotSystemVariables(), command(), MaskEnv(), Quote(), quoteTok() (+3 more)

### Community 48 - "cliScriptPath"
Cohesion: 0.18
Nodes (17): appUsers(), TestAdditionalUsers(), TestAdditionalUsersReportsExistingUser(), TestDiagnostics(), TestDisableDefaultUsers(), TestDisableDefaultVPN(), TestDomainCerts(), TestPathHelpers() (+9 more)

### Community 50 - "Cluster"
Cohesion: 0.27
Nodes (5): Cluster, orNone(), orValue(), setOrMissing(), setOrNone()

### Community 51 - "scripts_test.go"
Cohesion: 0.14
Nodes (15): disableDefaultUsersScript(), domainCertsScript(), gatherConfigsScript(), parseVPNNames(), productKeysScript(), sortedKeys(), TestDisableDefaultUsersScriptQuoting(), TestDomainCertsScriptSorted() (+7 more)

### Community 52 - ".releaseToBackup"
Cohesion: 0.29
Nodes (7): field(), TestField(), TestPrimaryRedundancyUp(), activity(), Ops, primaryRedundancyUp(), rdEnabledUp()

### Community 53 - ".LeaderLocal"
Cohesion: 0.23
Nodes (7): assertLeaderScript(), TestAssertLeaderScript(), Ops, hostMatches(), roleName(), shortHost(), TestRoleName()

### Community 57 - "verify_ops.go"
Cohesion: 0.32
Nodes (6): TestHTTPStatusHelpers(), TestLastLines(), TestLastLinesEqualCount(), httpStatusLines(), isHTTP2xx(), lastLines()

### Community 61 - ".ServerCert"
Cohesion: 0.47
Nodes (4): concatFiles(), serverCertFile(), serverCertScript(), TestServerCertScript()

## Knowledge Gaps
- **170 isolated node(s):** `solace`, `showCmd`, `Config`, `operatorTmplVars`, `Cluster` (+165 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **8 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Config` connect `Config` to `bg`, `ctrCfg`, `context.Context`, `Manager`, `newTestOps`, `NewCluster`, `convert_test.go`, `captureStdout`, `Cluster`, `prep_test.go`, `verify_local_test.go`, `github.com/spf13/cobra.Command`, `render.go`, `Image`, `.AdditionalUsers`, `recRunner`, `Compose`, `haCfg`, `kubectlTransport`, `secrets_test.go`, `Cluster`, `containerTransport`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Why does `Platform` connect `Platform` to `manager.go`, `ctrCfg`, `testing.T`, `Manager`, `Config`, `Compose`, `load.go`, `Load`, `convert_test.go`, `captureStdout`, `containerTransport`, `tierFor`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.073) - this node is a cross-community bridge._
- **Why does `Role` connect `Role` to `bg`, `scripts.go`, `.AdditionalUsers`, `context.Context`, `Compose`, `Manager`, `kubectlTransport`, `Cluster`, `scripts_test.go`, `.releaseToBackup`, `.LeaderLocal`, `containerTransport`, `verify_local_test.go`, `verify_ops.go`, `github.com/spf13/cobra.Command`?**
  _High betweenness centrality (0.033) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `ctrCfg()` (e.g. with `TestComposeSecretEnvIsTheOnlyChildEnvironment()` and `TestComposeSecretEnvNamesCannotBeSystemVars()`) actually correct?**
  _`ctrCfg()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `solace`, `showCmd`, `Config` to the rest of the system?**
  _170 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bg` be split into smaller, more focused modules?**
  _Cohesion score 0.06501831501831502 - nodes in this community are weakly interconnected._
- **Should `ctrCfg` be split into smaller, more focused modules?**
  _Cohesion score 0.052941176470588235 - nodes in this community are weakly interconnected._