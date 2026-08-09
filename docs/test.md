# Test catalogue

Every Go test in this repository, grouped by package and file, with a one-line statement of
what each one proves. Use it to find existing coverage before adding a test, and to spot
what is *not* covered.

This file is maintained by hand. When you add, rename, or delete a test, update the matching
row in the same change (CLAUDE.md S6).

## Running the tests

Tests are run through the mirrored dev scripts, never with a bare `go test` in CI:

| Want | Windows | Linux/macOS |
| --- | --- | --- |
| Run everything | `scripts\dev.ps1 test` | `./scripts/dev.sh test` |
| Coverage profile + total | `scripts\dev.ps1 cov` | `./scripts/dev.sh cov` |
| Build + vet + test (CI's gate) | `scripts\dev.ps1 all` | `./scripts/dev.sh all` |

- The `test` task runs `go test -count=1 ./...`. Race detection is on by default in `dev.sh`;
  on `dev.ps1` it is opt-in with `SOLACE_RACE=1`.
- `cov` writes `coverage/coverage.out` and `coverage/coverage.html` and prints the total. The
  previous total in `scripts/logs/cov.log` is the local floor -- an unexplained drop is a
  failed gate. CI is a fresh checkout with no prior log, so it cannot catch a coverage
  regression; that check is local only.
- Per-task logs land in `scripts/logs/<task>.log`.

Narrowing a run during development (not a substitute for the gate):

```
go test ./internal/config -run TestResolveEnvPath -v
```

Three packages carry golden files and accept `-update` to regenerate them. Only run it after
eyeballing the diff -- the committed goldens are the reviewed expected output:

```
go test ./internal/render -update
go test ./internal/k8s -update
go test ./internal/cli -update      # rewrites docs/commands.md
```

## Summary

22 test files, 326 test functions. `TestHelperProcess` in `internal/engine` is not a real
test -- it is the os/exec helper-process shim, a no-op unless `GO_WANT_HELPER_PROCESS=1`.

| Package | Files | Tests |
| --- | --- | --- |
| internal/broker | 4 | 85 |
| internal/k8s | 9 | 65 |
| internal/container | 2 | 55 |
| internal/config | 1 | 38 |
| internal/cli | 2 | 39 |
| internal/convert | 1 | 22 |
| internal/engine | 1 | 13 |
| internal/tools/vulnjudge | 1 | 5 |
| internal/render | 1 | 4 |
| **Total** | **22** | **326** |

## Coverage

Last recorded run, from `scripts/logs/cov.log` (2026-08-09). Re-run `cov` after any change;
these figures go stale the moment tests move.

| Package | Coverage |
| --- | --- |
| internal/config | 98.4% |
| internal/container | 96.8% |
| internal/convert | 96.3% |
| internal/cli | 90.0% |
| internal/k8s | 89.5% |
| internal/render | 89.0% |
| internal/broker | 84.9% |
| internal/tools/vulnjudge | 83.1% |
| internal/engine | 79.7% |
| **total** | **90.4%** |

---

## internal/config

Config loading, defaults, validation, and env-file resolution. 38 tests, all in one file.

### config_test.go

| Test | What it covers |
| --- | --- |
| `TestPlatformIsContainer` | `Platform.IsContainer` is true for docker/podman, false for k8s |
| `TestRedundancyEnabled` | Only the literal `yes` enables HA; `no`, empty, and junk do not |
| `TestImageRef` | `Image.Ref` joins repo:tag, prefixing the registry only when set |
| `TestParseRole` | Long and short role spellings parse, empty defaults to primary, junk errors |
| `TestRoleLetter` | Role -> `p`/`b`/`m` letter used in resource names |
| `TestResolveNodeStandalone` | Standalone ignores the role and always resolves the primary as a message-routing node |
| `TestResolveNodeHA` | HA resolves each role to its host name, with the monitor typed `monitoring` |
| `TestContainerRuntime` | Runtime binary comes from the platform's block; k8s has none |
| `TestContainerBlock` | Podman reads its own container block; everything else falls through to docker's |
| `TestNetworkBlock` | Network block is selected per platform |
| `TestApplyDefaultsK8s` | Every k8s default lands: redundancy, update strategy, admin secret, diag dir, CLI folder, storage, resources, operator image/resources, scaling, ports, anti-affinity |
| `TestApplyDefaultsK8sTLS` | TLS cert/key default only when `tls.serverSecret` is set |
| `TestApplyDefaultsDocker` | Docker defaults (runtime, compose mode, host network, admin user, container name) plus the shared `k8s.*` fields containers reuse |
| `TestApplyDefaultsPodmanRootful` | Rootful podman gets the system quadlet dir, no `--user`, `multi-user.target` |
| `TestApplyDefaultsPodmanRootlessXDG` | Rootless quadlet dir derives from `XDG_CONFIG_HOME`, with `--user` and `default.target` |
| `TestApplyDefaultsPodmanRootlessHomeDir` | Empty `XDG_CONFIG_HOME` falls back to the user home dir branch |
| `TestValidateK8sValid` | A fully populated k8s config validates clean |
| `TestValidateK8sMissingMandatory` | Every missing mandatory k8s field is named in one message, exact wording pinned |
| `TestValidateK8sBadUpdateStrategy` | `k8s.updateStrategy` enum is rejected loud |
| `TestValidateContainerHA` | A valid HA container config validates for both docker and podman |
| `TestValidateContainerStandalone` | Standalone only requires `nodes.primary.name` among the node fields |
| `TestValidateContainerMissingMandatory` | Missing container fields (image, admin, all three node name/ip pairs) are all named |
| `TestValidateContainerBridge` | `network.mode=bridge` without ports errors; with ports it passes |
| `TestValidateContainerBadNetworkMode` | Unknown network mode is rejected loud |
| `TestValidateDockerBadMode` | `docker.mode` enum (compose/run) is rejected loud |
| `TestValidateUnknownPlatform` | An unrecognised platform fails rather than validating nothing |
| `TestValidateBadRedundancy` | `redundancy` enum is rejected loud |
| `TestResolveEnvPath` | Env-file lookup over a real temp tree: base dir first, `env/` fallback, base dir shadows `env/`, default name, no extension inference, a path used verbatim with no `env/` retry, a directory is not a match, both candidates named in the not-found error, control characters rejected |
| `TestResolveEnvPathEmptyBaseDir` | An empty base dir means the current directory, for both candidates |
| `TestResolveEnvPathDefaultInBaseDir` | The default name resolves in the base dir before the `env/` fallback is tried |
| `TestLoadSuccess` | A valid file loads, and defaults are applied during `Load` |
| `TestLoadReadError` | A missing file errors with `read env file` |
| `TestLoadParseError` | Malformed YAML errors with `parse env file` |
| `TestLoadUnknownField` | Strict decoding turns a typo'd key into a hard error |
| `TestLoadBashEnvFileHint` | A legacy bash env file is reported as not-YAML and points at `solace convert` |
| `TestLoadNotYAMLHint` | Any other non-YAML file says the env file must be YAML and names the schema and the converter |
| `TestLoadUnknownFieldHasNoConvertHint` | A valid-YAML file with an unknown key stays a schema error, without the convert hint |
| `TestLoadValidationError` | A file that parses but fails validation surfaces the missing-fields message |

---

## internal/cli

Command-tree wiring, global flags, confirm prompts, and end-to-end `--dry-run` /
`--gen` passes over the sample env, plus the generated command reference. 39 tests across
two files.

### cli_test.go

| Test | What it covers |
| --- | --- |
| `TestEnvFileLookup` | `-e`/`--env` as the CLI wires it: `env/` fallback, base dir shadowing, no extension inference, the `==> env file:` echo, and long/short flag parity |
| `TestFirstArg` | `firstArg` on nil and populated slices |
| `TestFirstArgOr` | `firstArgOr` falls back on a missing or empty first argument |
| `TestNotImplemented` | The placeholder error names the command and says "not implemented yet" |
| `TestEmit` | `emit` writes bytes to stdout unchanged |
| `TestWarnAndStep` | `warn` and `step` write `[WARN]` / `==>` lines to stderr |
| `TestTreeStructure` | Every platform and a representative set of leaf command paths exist in the tree |
| `TestFlagsRegistered` | Per-command flags (`purge`/`clear-data`/`keep-data`, `keep-yaml`, `days`, `pod`, `dir`) are registered where expected |
| `TestHelpNoConfig` | `--help` at root and per platform short-circuits before config load, so no env is needed |
| `TestGenWired` | Every render-only path emits the right artifact: k8s CR (`apiVersion:`), compose (`services:`), quadlet (`[Unit]`) |
| `TestCtrWiredDryRun` | Every container command safe under `--dry-run` reaches its handler and echoes the expected runtime/systemctl/mkdir command |
| `TestCtrRoleGuards` | Node-local HA guards: leader must run on the primary, redundancy rejects the monitor, bad roles error, and standalone self-skips |
| `TestCtrConfigDryRun` | Container config steps run clean on a standalone env; cert/product-key-gated steps self-skip |
| `TestCtrErrorPaths` | Container config/verify failures are actionable: no TLS, no product keys, no exec-cli file, failed SEMP login |
| `TestCtrVerifyDiagnosticsDryRun` | Container diagnostics echoes its gather/download sequence (isolated because it creates a diag dir) |
| `TestCtrRoleArgCount` | Role-taking commands reject a second positional argument |
| `TestCtrRoleHelp` | Role-taking commands expose `--help` without loading an env |
| `TestK8sWiredDryRun` | Every k8s command safe under `--dry-run` runs clean, with `+ kubectl` echoed on the acting paths and absent on the skip paths |
| `TestK8sStandaloneDryRun` | Redundancy-branching commands on a standalone env: HA-only steps self-skip, config/prep/up run clean |
| `TestK8sVerifyDiagnosticsDryRun` | k8s diagnostics echoes the gather/download sequence for all three nodes (isolated for its dir side-effect) |
| `TestK8sGenOperatorWired` | `gen operator` and `operator deploy --gen` emit the bundle with every template marker resolved |
| `TestSecretsNeverEchoed` | A secret-bearing command under `--dry-run` shows stdin as a byte count and never the password |
| `TestK8sErrorPaths` | k8s handler boundaries: missing cert/key, no product keys, failed login, missing exec-cli file, bad `--pod`, `--gen` rejected on non-artifact commands, mutually exclusive data flags |
| `TestConfirmFlagShortcuts` | `--yes` confirms a delete; `--purge`/`--keep-data` drive retention without reading stdin |
| `TestConfirmNonTTY` | Without a TTY and without `--yes`, delete refuses and purge keeps data |
| `TestPromptYesNo` | Lenient delete prompt: `y`/`yes` in any case accept, everything else declines |
| `TestPromptYes` | Strict purge prompt: only an exact trimmed `yes` accepts; a bare `y` does not |
| `TestErrorPaths` | Global rejections: unresolvable env file, invalid node roles across container and k8s leaves, unknown `gen` target |
| `TestGenDockerRunMode` | With `docker.mode=run` the generated artifact is the `docker run ...` command line, not a compose file |
| `TestCtrVerifyAll` | `verify` role arms: unknown host fails loud, this-host-is-monitor skips redundancy, standalone skips redundancy |
| `TestCtrConfigAllArms` | `config` with every optional step configured runs all three gated arms, and the private key never reaches stdout |
| `TestCtrExecCLIPathSeparator` | An exec-cli argument containing a separator is used as-is, not joined under the CLI scripts folder |
| `TestConvertToStdout` | `convert` writes YAML to stdout and its warnings to stderr, so the artifact stays clean |
| `TestConvertToFile` | `-o` writes the file, a second run refuses to clobber it, and `--force` overrides |
| `TestConvertRoundTrip` | A converted file loads: `-e` against it drives a real command |
| `TestConvertErrorPaths` | Bad `--platform`, a missing source file, and a missing argument all fail loud |
| `TestBashEnvGivenToEnvFlag` | Pointing `-e` at a legacy bash file reports not-valid-YAML and names `solace convert` |
| `TestExecute` | `Execute()` builds the tree and runs `--help` without error |

### commanddoc_test.go

| Test | What it covers |
| --- | --- |
| `TestCommandDocs` | Renders the command reference from the live tree and fails while `docs/commands.md` is stale -- the drift gate for every command path, positional, flag, and `Short` string. This file is also the generator: `-update` rewrites the doc |

---

## internal/convert

The legacy bash env -> YAML converter: a shell-assignment parser, the variable
mapping, and the YAML emitter. 22 tests.

### convert_test.go

| Test | What it covers |
| --- | --- |
| `TestConvertBashSample` | The committed `bash/env/sample` converts end to end: platform detected as k8s, `true` -> `yes`, every scalar/array/associative value mapped, `${SOLBK_NS}` expanded, a trailing comment stripped, an explicit `0` kept, an empty PSK omitted, and no warnings |
| `TestConvertContainer` | A container env file maps the node table, container block, ulimits, network, and spool scaling |
| `TestConvertPlatformDetection` | Podman markers, docker markers, and both-present all resolve to the expected section |
| `TestConvertPodmanSection` | Podman rootless and quadlet dir land in the podman block, and no docker block is written |
| `TestConvertExplicitPlatformWins` | `--platform` overrides detection and suppresses the detection warning |
| `TestConvertUnmappedVariablesWarn` | Variables with no YAML equivalent are named in the warnings, not dropped silently |
| `TestConvertBashPlumbingIsSilent` | Bootstrap-only variables (`KUBE`, `EXDIR`, `GENONLY`) are dropped without noise |
| `TestConvertRedundancySpellings` | `true`/`yes` and `false`/`no` normalise (any case); anything else copies through with a warning |
| `TestConvertBadNumberWarns` | A non-numeric value for a numeric field warns and is not written |
| `TestConvertBadBooleanWarns` | An unparseable boolean (`SOLOP_WATCH_SOLBK_NS`, which the bootstrap never enum-checked) warns and is not written |
| `TestGeneratedHeaderSanitisesSource` | A control character in the source name cannot end the header comment and inject document structure |
| `TestConvertIncompleteEnvWarns` | A source env missing mandatory fields converts, but says the result is incomplete |
| `TestConvertUnterminatedArray` | An array assignment with no closing paren is a hard error |
| `TestConvertInvalidPlatformSection` | An unrecognised platform still writes the shared sections and surfaces the validation warning |
| `TestParseAssignmentForms` | Every assignment form: bare, double/single quoted, empty, trailing comment, `export`/`declare`, inline and multi-line arrays, `declare -A` and bare `[k]=v` maps, `${VAR}`/`$VAR`/unset references, and a function definition that must not parse |
| `TestParseScalarListFallback` | A single-entry list written as a scalar reads as a one-element list; an absent one is nil |
| `TestParseCRLF` | CRLF line endings parse the same as LF, including multi-line arrays |
| `TestParseEscapedQuote` | `\"` inside a double-quoted value survives |
| `TestUnmappedTracksFileOrder` | Unmapped variables are reported in file order, not map order |
| `TestScalarQuoting` | Values a YAML reader could misread (bools, null, numbers, paths, `:`, `#`, quotes, backslashes) are quoted; plain identifiers are not |
| `TestEmptyBlocksOmitted` | Blocks with no content are left out entirely |
| `TestGeneratedHeader` | The output carries the provenance header naming the source file |

---

## internal/broker

Broker CLI operations over an injected transport: script generation, config steps, verify
state machines, and the node-local HA variants. 85 tests across 4 files.

### broker_test.go

| Test | What it covers |
| --- | --- |
| `TestField` | `field` extracts a labelled value from CLI output, empty for an absent label |
| `TestCountContains` | `countContains` counts labelled lines carrying a substring |
| `TestContainsAnyFold` | Case-insensitive substring match used for error detection |
| `TestValidName` | The boundary validator accepts safe names and rejects empty, `..`, separators, and shell metacharacters |
| `TestPathHelpers` | `cliScriptPath`, `cliArg`, and `certPath` build the in-jail paths |
| `TestLastLines` | `lastLines` returns the tail, and the whole input when it is shorter than n |
| `TestHTTPStatusHelpers` | `isHTTP2xx` accepts only 2xx; `httpStatusLines` extracts every status line |
| `TestPrimaryRedundancyUp` | The primary health predicate requires redundancy Up |
| `TestRunCLIUploadsThenExecs` | `RunCLI` uploads the script body, then execs it, and returns its output |
| `TestRunCLIRejectsBadName` | An invalid script name is rejected before any upload happens |
| `TestSkipIfStandalone` | The HA-only guard is false in HA and true in standalone |
| `TestServerCert` | The uploaded bundle is key+cert+CA concatenated, plus the apply script |
| `TestServerCertRequiresCert` | Missing `tls.cert`/`certKey` errors |
| `TestDomainCerts` | Each CA file is uploaded and the load script matches the generated one |
| `TestDomainCertsRejectsBadName` | A CA name with a space is rejected before any upload |
| `TestDomainCertsEmptySkips` | No configured certs means no calls at all |
| `TestDisableDefaultVPN` | The hardening script is uploaded and its scripts are cleaned up with one `rm -f` |
| `TestDisableDefaultUsers` | Every parsed VPN gets a `client-username default` line |
| `TestDisableDefaultUsersNoVPNs` | Unparseable VPN output means the step does not run |
| `TestProductKeys` | The generated product-key script is uploaded verbatim |
| `TestProductKeysDetectsError` | A broker-reported failure in the output fails loud |
| `TestProductKeysEmpty` | No keys configured is an error |
| `TestExecCLI` | A local script is uploaded under its base name and cleaned up afterwards |
| `TestExecCLIRejectsBadName` | A base name of `..` is rejected before upload |
| `TestLogin` | A 2xx SEMP response succeeds, and the password rides stdin, never the argv |
| `TestLoginFailure` | A 401 reports failure without erroring |
| `TestLoginNoResponse` | Empty output is reported as "no HTTP response" |
| `TestLeaderStandaloneSkips` | Leader makes no calls in standalone mode |
| `TestLeaderSuccess` | Activity is reverted on the backup, leadership asserted on the primary, sync reported |
| `TestLeaderTimeout` | Redundancy that never recovers times out and dumps `show redundancy detail` |
| `TestRedundancySuccess` | The full failover handshake walks its scripted primary and backup sequences to completion |
| `TestRedundancyStandaloneSkips` | Redundancy makes no calls in standalone mode |
| `TestDiagnostics` | The dest dir is created, the gather script matches, and both the configs zip and the diagnostics bundle are downloaded under their expected names |

### coverage_test.go

Branch coverage for the paths the happy-path tests in `broker_test.go` cannot reach.

| Test | What it covers |
| --- | --- |
| `TestNewDefaults` | `New` sets the transport, config, 2s poll interval, 60 attempts, and stdout |
| `TestOutDefaultsToStdout` | `out()` falls back to stdout when `Out` is nil |
| `TestShowWritesToStdout` | `show` writes through the resolved default sink |
| `TestSleepElapses` | A tiny poll interval sleeps and returns nil |
| `TestSleepZeroIntervalReturnsCtxErr` | A zero interval with a live context returns nil |
| `TestSleepCancelled` | A cancelled context surfaces `context.Canceled` |
| `TestPollCondError` | A condition error is propagated rather than retried away |
| `TestPollContextCancelled` | A cancelled context ends the poll with `context.Canceled` |
| `TestRemoveCLIWarnsOnFailure` | Failed cleanup warns instead of erroring, and still issues `rm -f` for every path |
| `TestFieldLabelWithoutColon` | A label line with no colon-space separator yields empty |
| `TestLastLinesEqualCount` | `lastLines` when n equals the line count |
| `TestExecCLIWarnsOnErrorOutput` | Error-looking CLI output produces a warning |
| `TestExecCLIRunError` | A failed run errors, and cleanup is still attempted |
| `TestServerCertBundleReadError` | Unreadable cert/key files error |
| `TestServerCertCAReadError` | An unreadable CA file errors |
| `TestDomainCertsBadFilename` | A certificate *filename* with a space is rejected (the CA name being valid) |
| `TestDiagnosticsTwoRolesNoBundle` | Output with no "Diagnostics saved" line pulls only the configs zip, once per role |
| `TestDiagnosticsRunError` | A transport `Run` failure surfaces |
| `TestLeaderPollCondError` | A failing `show redundancy` propagates, and the detail dump still runs |
| `TestRedundancyShowRDError` | The initial `show redundancy` error surfaces |
| `TestRedundancyUnhealthyPrimary` | An unhealthy primary fails and its output is shown |
| `TestRedundancyNeitherActive` | Neither node locally active is a loud failure |

### scripts_test.go

Pins the generated broker CLI script text -- these strings are what the broker executes.

| Test | What it covers |
| --- | --- |
| `TestFixedScripts` | The seven fixed scripts (show redundancy, detail, revert, release, no-release, show vpn, show vpn bare) match byte for byte |
| `TestRevertActivityTrailingSpace` | The redundancy-test revert keeps its trailing space; the leader-path revert does not |
| `TestAssertLeaderScript` | Asserts leader for router and all VPNs, ending with the config-sync database show |
| `TestServerCertScript` | The cert filename format plus the script's prefix, load line, and closing show |
| `TestDomainCertsScriptSorted` | Map input emits CAs in sorted order for deterministic output, with the right prefix/suffix and per-CA block |
| `TestDisableDefaultUsersScriptQuoting` | A VPN name containing a space stays one quoted token |
| `TestProductKeysScript` | Exact script text for a list of keys |
| `TestDisableDefaultVPNScript` | The hardening script disables the VPN, its default user, plain-text downgrade, and plain-text SMF |
| `TestParseVPNNames` | Column-based VPN parsing skips the legend, header, separator, comments, and blank lines |
| `TestParseVPNNamesNoSeparator` | Output without a separator row yields no names |
| `TestGatherConfigsScript` | Prefix, first show command, `gather-diagnostics` with days substituted, and one line per configured show |
| `TestZipConfigsScript` | The zip command is present |
| `TestSortedKeys` | `sortedKeys` returns map keys in sorted order |

### verify_local_test.go

The node-local HA state machines used by the container platforms, where each host drives
only its own half.

| Test | What it covers |
| --- | --- |
| `TestLocalRole` | Role detection: an explicit arg wins, host name matches the node table (including FQDN vs short name), no match and bad args fail loud |
| `TestLeaderLocalStandaloneSkips` | No calls in standalone mode |
| `TestLeaderLocalRejectsNonPrimary` | Backup and monitor hosts, and an explicit backup arg, are rejected before any upload |
| `TestLeaderLocalSuccess` | On the primary, leadership is asserted and sync reported |
| `TestLeaderLocalTimeoutDumpsDetail` | Unrecovered redundancy times out and dumps the detail |
| `TestRedundancyLocalStandaloneSkips` | No calls in standalone mode |
| `TestRedundancyLocalRejectsMonitor` | The monitor is rejected before any `show redundancy` |
| `TestRedundancyLocalPrimaryActive` | Active primary releases, un-releases, and waits for fail-back, consuming the whole scripted sequence |
| `TestRedundancyLocalPrimaryStandby` | A standby primary only waits for fail-back and never releases activity |
| `TestRedundancyLocalBackupActive` | An already-active backup reverts activity to the primary |
| `TestRedundancyLocalBackupInactive` | An inactive backup waits to become active, dwells, then reverts |
| `TestRedundancyLocalInitialShowError` | An initial `show redundancy` error propagates |
| `TestRedundancyLocalBadRoleArg` | A bad explicit role argument propagates |
| `TestRedundancyLocalPrimaryUnhealthy` | An unhealthy primary fails loud without releasing activity |
| `TestRedundancyLocalPrimaryFailBackTimeout` | A fail-back that never arrives times out, still without releasing |
| `TestRoleName` | Role -> display name |
| `TestLocalRoleDefaultHostname` | With the seam unset, the real `os.Hostname` is used and an off-table host fails loud |

---

## internal/k8s

Everything driven through `kubectl`: preflight, prep, deploy, operator, day-2 ops, secrets,
and the pod transport. 65 tests across 9 files.

### names_test.go

| Test | What it covers |
| --- | --- |
| `TestResourceNames` | Pod, PVC, StatefulSet, and load-balancer service names for every role |
| `TestHARoles` | HA yields all three roles; standalone yields only the primary |
| `TestProductKeyRoles` | Product keys target primary+backup in HA, primary only in standalone |

### cluster_test.go

| Test | What it covers |
| --- | --- |
| `TestOperatorNSExplicit` | A configured operator namespace is used without probing the cluster |
| `TestOperatorNSDerived` | Discovery reads the namespace from the operator deployment row |
| `TestOperatorNSDefaultWhenAbsent` | No operator row falls back to the fixed default |
| `TestOperatorNSDefaultOnError` | An unreachable cluster falls back to the default rather than failing |
| `TestApplyOnStdin` | `apply` pipes the manifest on stdin via `kubectl apply -f -` |
| `TestDeleteStdin` | `deleteStdin` pipes the manifest with `--ignore-not-found` |

### check_test.go

| Test | What it covers |
| --- | --- |
| `TestCheckEnvNoSecretLeak` | The config report shows secrets as set/MISSING and never prints their values |
| `TestReachable` | The API-server probe argv, and failure when it errors |
| `TestCheckStorageClass` | A suitable configured class passes with no default lookup; Immediate binding or no expansion is rejected; missing attributes reject; dry-run skips the assertions |
| `TestResolveStorageClass` | A configured class short-circuits; a single default resolves; multiple defaults error; no default returns empty |
| `TestCheckDryRun` | The whole preflight runs clean under Echo and emits the report plus the skip note |

### prep_test.go

| Test | What it covers |
| --- | --- |
| `TestCreateNamespace` | The namespace manifest is applied on stdin |
| `TestDeleteNamespace` | Namespace teardown argv with `--ignore-not-found` |
| `TestCreateSecretsAdminOnly` | With no TLS or pull secret only the admin secret is applied, as a single document |
| `TestCreateSecretsAllThree` | Admin + TLS + pull secret join into one multi-doc apply, and the registry password reaches neither argv nor plaintext stdin |
| `TestCreateSecretsPreflight` | Missing TLS inputs fail before any apply runs |
| `TestDeleteSecrets` | All configured secrets are deleted; admin-only config deletes one |
| `TestUpdateServerCertSecret` | The TLS secret is applied on stdin; an unset secret name errors |
| `TestSplitLabel` | Label parsing across `=` and `:` forms, with whitespace, and rejection of malformed entries |
| `TestIsBuiltinLabel` | Kubernetes-owned label keys are recognised and custom ones are not |
| `TestLabelNodesNoCustomLabels` | No labels means no cluster calls and an early-exit message |
| `TestLabelNodesBuiltinOnly` | Built-in labels are never applied |
| `TestLabelNodesMalformedAndUnsafe` | Malformed and unsafe-character labels are dropped with warnings and never reach the cluster |
| `TestLabelNodesHappyPath` | RBAC precheck, node list, then the label call for the selected node |
| `TestLabelNodesReprompt` | Out-of-range and non-numeric selections re-prompt before the correct node is labelled |
| `TestLabelNodesRBACDenied` | A failed RBAC precheck aborts and labels nothing |
| `TestLabelNodesEOFNoSelection` | EOF with no selection errors |

### deploy_test.go

| Test | What it covers |
| --- | --- |
| `TestDeployBrokerApply` | One apply on stdin carrying the rendered CR |
| `TestDeployBrokerKeepYAML` | `--keep-yaml` writes `.broker.yaml` byte-identical to what was applied |
| `TestDeleteBrokerNoPurge` | Without purge only the CR is deleted, no PVCs |
| `TestDeleteBrokerPurgeHA` | Purge deletes the CR plus all three role PVCs |
| `TestDeleteBrokerPurgeStandalone` | Purge on standalone deletes the CR plus the single PVC |
| `TestDeleteBrokerPurgeSwallowsPVCError` | A failing PVC delete is best-effort: teardown continues and every PVC is still attempted |

### operator_test.go

| Test | What it covers |
| --- | --- |
| `TestWatchNamespace` | `WATCH_NAMESPACE` joins: broker namespace appended by default, onto a configured list, or omitted when disabled |
| `TestRenderOperatorSubstitutions` | Every substitution point lands (namespace, watch list, image with/without registry prefix, resources, pull secret) and no template marker survives |
| `TestGenOperator` | Render-only uses the configured operator namespace, or the fixed default when unset |
| `TestOperatorApply` | regcred is applied into the operator namespace first, then the bundle, both on stdin |
| `TestOperatorApplyNoPullSecret` | With no pull secret only the bundle is applied |
| `TestOperatorDelete` | Teardown deletes the rendered bundle with `--ignore-not-found` |
| `TestOperatorLogsArgs` | Log passthrough targets the operator deployment |

### ops_test.go

| Test | What it covers |
| --- | --- |
| `TestStatus` | Status queries pods, services, and statefulsets in the broker namespace |
| `TestShowAll` | Cluster-wide listing keeps broker resources and filters out the operator pod and unrelated resources |
| `TestShowAllWrapsGetError` | A failing get aborts loudly with per-resource context and preserves the cause |
| `TestFilterLines` | Empty input, header retention, matching, and the "(none matched)" note |
| `TestDescribeBroker` | Describe targets the pod for the requested role |
| `TestDescribeLB` | Describe targets the load-balancer service |
| `TestLogsPassthrough` | Extra log flags pass through to the role's pod |
| `TestCLIAndShellAreInteractive` | `cli` and `shell` run interactively with the right in-pod command |
| `TestCopyFrom` | Each file downloads under its base name; an empty list errors; failures are aggregated after every file is attempted |
| `TestCopyInto` | Uploads into the target dir, defaults it to `.`, errors on an empty list, aggregates failures |
| `TestReplicasStart` | HA scales and waits for all three roles, standalone only the primary, and a stuck rollout fails loud at the first role |
| `TestReplicasStop` | HA scales all three to zero, standalone only the primary |

### secrets_test.go

| Test | What it covers |
| --- | --- |
| `TestSecretGoldens` | Rendered admin, TLS, docker-registry, and operator-regcred secrets match their committed goldens |
| `TestAdminSecretDecodes` | The base64 data round-trips to the expected plaintext passwords |
| `TestAdminSecretErrors` | Empty password, empty secret name, and malformed per-user entries all error |
| `TestTLSSecretErrors` | Unset cert, unset secret name, and missing cert/CA/key files all error |
| `TestDockerRegistrySecretEmptyName` | An empty pull-secret name errors |

### transport_test.go

| Test | What it covers |
| --- | --- |
| `TestTransportExecArgs` | Exec argv for Run/Output/OutputInput, `-i` only where stdin is used, and `-c` never present (broker pods are single-container) |
| `TestTransportUpload` | The body rides stdin through `sh -c 'cat > <dest>'` and never appears in the argv |
| `TestTransportUploadQuotesDest` | Single-quote escaping stops a metacharacter in a path breaking out of the redirect |
| `TestTransportCopy` | `kubectl cp` argv in both directions with the namespace flag |
| `TestTransportEchoHidesUploadBody` | End to end over Echo: the uploaded body shows as a byte count, and the CLI exec line is still echoed |

---

## internal/container

The host-local Docker/Podman manager and its node-local transport. 55 tests across 2 files.

### manager_test.go

| Test | What it covers |
| --- | --- |
| `TestManagerCheckDryRun` | Preflight report for docker/podman x HA/standalone: title, mode line, runtime version probe, dry-run skip note |
| `TestManagerCheckDNSFailsLoudInHA` | An unresolvable redundancy host fails the check and is named |
| `TestManagerCheckStandaloneDNSWarnsOnly` | Standalone tolerates an unresolved name |
| `TestManagerPrepHostDryRunDoesNotWritePSK` | Dry-run leaves the env file untouched, never generates a PSK, and still echoes mkdir/chown |
| `TestManagerPrepHostWritesPSK` | The generated PSK is written into `nodes.psk`, the replication PSK is untouched, and the data dir is created and chowned |
| `TestManagerPrepHostRootlessUsesUnshareChown` | Rootless podman chowns via `podman unshare` |
| `TestManagerDeployDockerComposeWritesFile` | Compose mode writes the file and runs `compose up -d` |
| `TestManagerDeployDockerRun` | Run mode issues `docker run` |
| `TestManagerDeployPodmanWritesUnit` | The quadlet unit is written, then daemon-reload and service start |
| `TestManagerDeployPodmanDryRunSkipsWrite` | Dry-run echoes the systemctl steps without writing the unit |
| `TestManagerPodmanEUIDGuardSkippedOnDryRun` | The rootless/rootful euid guard does not run under dry-run |
| `TestManagerDeletePodmanRemovesUnit` | Delete stops the service, removes the unit, and daemon-reloads |
| `TestManagerDeletePodmanPurgeRootless` | Rootless purge removes the data dir via `podman unshare` |
| `TestManagerDeleteDockerComposeDownWhenFileExists` | With a compose file present, delete runs `compose down` |
| `TestManagerDeleteDockerRunStopsAndRemoves` | Run mode stops and removes the container, and purge removes the data dir |
| `TestManagerDeleteDockerComposeNoFileFallsBackToStopRm` | A missing compose file falls back to stop+rm |
| `TestManagerStatusPodman` | Status shows the systemd unit and lists the container |
| `TestManagerStatusDockerCompose` | Compose-mode status runs `compose ps` and a filtered `docker ps` |
| `TestManagerLogsCLIShell` | `logs` follows, `cli` and `shell` exec interactively into the container |
| `TestReplacePSKLine` | Only the `nodes.psk` line is replaced, never the replication one; absence of the line is reported |
| `TestDefaultGenPSK` | The default generator produces 60 base64-encoded random bytes |
| `TestManagerCheckReachableError` | A failing runtime version probe fails the check |
| `TestManagerPrepHostMkdirError` | A mkdir failure propagates |
| `TestManagerPrepHostChownError` | A chown failure propagates |
| `TestManagerPrepHostRootlessUnshareChownError` | A rootless `unshare chown` failure propagates |
| `TestManagerPrepHostGenPSKError` | A PSK generation failure propagates |
| `TestManagerPrepHostWritePSKReadError` | An unreadable env file fails the PSK write-back |
| `TestManagerPrepHostWritePSKWriteError` | A read-only env file fails the PSK write-back (skipped when running as root) |
| `TestManagerPrepHostPSKAlreadySet` | An existing `nodes.psk` skips generation and says so |
| `TestManagerPrepHostNoPSKLinePrintsValue` | With no psk line to replace, the value is printed for the user and the file is not modified |
| `TestManagerDeployPodmanMkdirError` | An uncreatable quadlet dir fails deploy |
| `TestManagerDeployPodmanWriteUnitError` | An unwritable unit path fails deploy |
| `TestManagerDeployPodmanDaemonReloadError` | A daemon-reload failure propagates |
| `TestManagerDeployPodmanStartError` | A service-start failure propagates |
| `TestManagerDeployPodmanEUIDGuardFails` | Rootless-as-root is rejected by the euid guard |
| `TestManagerDeployDockerComposeWriteError` | An unwritable compose path fails deploy |
| `TestManagerDeployDockerComposeUpError` | A `compose up` failure propagates |
| `TestManagerDeployDockerRunError` | A `docker run` failure propagates |
| `TestManagerDeletePodmanStopTolerated` | A failed stop is tolerated with a warning |
| `TestManagerDeletePodmanDaemonReloadError` | A daemon-reload failure during delete propagates |
| `TestManagerDeletePodmanRemoveUnitError` | An unremovable unit path fails delete |
| `TestManagerDeleteDockerComposeDownError` | A `compose down` failure propagates |
| `TestManagerDeleteDockerRunStopTolerated` | A failed run-mode stop is tolerated with a warning |
| `TestManagerDeletePurgeError` | A failing data-dir removal under `--purge` propagates |
| `TestManagerStatusDockerRun` | Run-mode status lists the container and never calls compose |
| `TestManagerStatusPodmanUnitInactiveTolerated` | An inactive unit warns but status still lists the container |
| `TestManagerStatusDockerComposePsTolerated` | A failing `compose ps` warns but the plain `ps` still runs |
| `TestManagerCheckPodmanEUID` | The euid guard across rootless/rootful x root/non-root, and skipped on a non-POSIX euid |
| `TestManagerPrepHostRootlessAsRootWarns` | Rootless config running as root warns |
| `TestManagerNilSinks` | Nil log and output sinks fall back to discard and stdout without erroring |

### transport_test.go

| Test | What it covers |
| --- | --- |
| `TestTransportExecArgs` | Exec argv is `<runtime> exec [-i] <name> ...` with no `--` (docker rejects it), and the role argument is ignored because the transport is node-local |
| `TestTransportUpload` | The body rides stdin through `sh -c 'cat > <dest>'` and never appears in the argv |
| `TestTransportUploadQuotesDest` | Single-quote escaping stops a metacharacter in a path breaking out of the redirect |
| `TestTransportCopy` | `<runtime> cp` argv in both directions, container-name prefixed |
| `TestTransportEchoHidesUploadBody` | End to end over Echo: the body shows as a byte count and the CLI exec line carries no `--` |

---

## internal/engine

The command runner seam: `Echo` (dry-run) and `Exec` (real subprocess), plus display quoting.
13 tests.

### runner_test.go

| Test | What it covers |
| --- | --- |
| `TestHelperProcess` | Not a test -- the os/exec helper-process shim used as a fake external command by the `Exec` tests below |
| `TestQuoteTok` | Display quoting per token: empty, plain, and every shell-significant character |
| `TestQuote` | Whole command lines, including quoted and empty arguments |
| `TestEchoRun` | `Run` echoes `+ <cmd>` |
| `TestEchoRunInteractive` | `RunInteractive` echoes with quoting applied |
| `TestEchoRunInput` | `RunInput` shows stdin as a byte count, never its contents |
| `TestEchoOutput` | `Output` echoes and returns nil bytes |
| `TestEchoDefaultWriter` | A zero-value `Echo` writes to stdout |
| `TestExecOutput` | `Output` captures a child process's stdout |
| `TestExecOutputFail` | A non-zero exit errors and the message names the binary |
| `TestExecRun` | `Run` streams on success and errors (naming the binary) on failure |
| `TestExecRunInput` | `RunInput` feeds stdin to the child and streams its output |
| `TestExecRunInteractive` | `RunInteractive` runs a child to a clean exit |

---

## internal/render

Manifest and unit-file rendering, guarded by committed goldens. 4 tests.

### render_test.go

| Test | What it covers |
| --- | --- |
| `TestGolden` | Seven renderings from the sample env match their goldens: k8s broker CR (the sample omits `k8s.ports`, so this covers the 16 defaults), the same CR with an explicit port list (the specified branch, including a container port differing from the service port and an explicit protocol), podman quadlet, docker compose, docker run args, and container env pairs for HA and standalone |
| `TestParsePort` | Port entries across the `name=container`, `container:service`, and `/PROTO` forms |
| `TestParseToleration` | Toleration Equal (`key=value:effect`) and Exists (`key:effect`) forms |
| `TestQuadletEscape` | systemd `Environment=` escaping of `%`, `"`, and `\` |

---

## internal/tools/vulnjudge

The dev-only judge the `scan` task pipes govulncheck JSON through. 5 tests.

### main_test.go

| Test | What it covers |
| --- | --- |
| `TestJudge` | The core policy: a called vulnerability with a fix exits 1, one with no released fix warns and exits 0, and uncalled findings are ignored |
| `TestJudgeModuleFixHint` | A fixable module vulnerability advises `go get <module>@<version>` rather than a toolchain bump |
| `TestJudgeNoFindings` | A clean scan exits 0 with the no-vulnerabilities message |
| `TestJudgeTolerantOfBOM` | A UTF-8 BOM (what PowerShell's encoders emit) does not break decoding |
| `TestJudgeMalformedInput` | Truncated or non-JSON input exits 2 with a malformed-JSON message rather than reporting success |

---

## Fixtures and doubles

There is no shared `testutil` package by design -- each package keeps its own small doubles
next to the tests that use them. Reuse the one in your package rather than hand-rolling a
new fake.

### Shared env fixtures

- **`env/sample.yaml`** is the one fixture for goldens and CLI end-to-end runs. It is a valid
  config for all three platforms, so `internal/render`, `internal/k8s`, and `internal/cli`
  all load it (as `../../env/sample.yaml`) instead of maintaining separate fixtures. It is HA
  with TLS configured.
- Tests needing a clean single-broker pass write their own minimal env to a temp file:
  `writeStandaloneEnv` (k8s-shaped) and `writeCtrStandaloneEnv` (container-shaped, needs a
  `nodes:` block) in `internal/cli/cli_test.go`.
- **`bash/env/sample`** is the legacy-format fixture. `internal/convert` converts the real
  committed file rather than an inlined copy, so a change to the old format is caught. The
  container flavour uses an inline fixture (`ctrEnv`) instead: the committed
  `bash/docker-podman/env/sample` is written as a bash script rather than a plain env file
  (several assignments per line), which is not a format the converter targets.

### Per-package doubles

| Package | Double | Purpose |
| --- | --- | --- |
| internal/broker | `fakeTransport` (broker_test.go) | Records every upload/run/output and answers via a `responder` func |
| internal/broker | `newTestOps` (broker_test.go) | Builds an `Ops` with a buffer sink and zero poll interval so nothing sleeps |
| internal/broker | `runErrTransport` (coverage_test.go) | Embeds `fakeTransport` but fails `Run`, reaching the best-effort cleanup branches |
| internal/broker | `seqTransport`, `newLocalOps`, `rd` (verify_local_test.go) | Scripts a sequence of `show redundancy` readings for the local HA state machines |
| internal/k8s | `recRunner` / `rrCall` (transport_test.go) | Capturing `engine.Runner` with `outQueue` and `runErrQueue` for scripting multi-step ops |
| internal/k8s | `haCfg`, `saCfg` (names_test.go), `adminCfg` (prep_test.go), `loadK8s` (secrets_test.go) | Config builders |
| internal/k8s | `checkGolden`, `-update` flag (secrets_test.go) | Golden comparison for the whole package |
| internal/cli | `renderCommandDocs`, `-update` flag (commanddoc_test.go) | Renders `docs/commands.md` from the cobra tree; the doc is the golden |
| internal/container | `capRunner` / `failOn` (transport_test.go) | Capturing runner whose `fail` hook errors on a targeted command, driving each error-wrap branch |
| internal/container | `newEchoMgr`, `newCapMgr`, `ctrCfg` (manager_test.go) | Manager over dry-run Echo, or over the capturing runner for real file writes |
| internal/cli | `runRoot`, `capture`/`captureStdout`/`captureStderr` (cli_test.go) | Builds a fresh command tree per call and captures a standard stream through a pipe |
| internal/cli | `bashEnv`, `writeBashEnv` (cli_test.go) | Minimal legacy env file for the convert command tests |
| internal/convert | `strictDecode` (convert_test.go) | Re-reads generated YAML with `KnownFields(true)`, so an emitted key that is not in the schema fails the test |
| internal/convert | `ctrEnv`, `convertOK`, `hasWarning` (convert_test.go) | Container-flavoured legacy fixture and warning assertions |
| internal/config | `envTree`, `writeTempYAML` (config_test.go) | Real temp-dir fixture trees for path resolution and loading |
| internal/engine | `helperCommand` + `TestHelperProcess` (runner_test.go) | Re-invokes the test binary as a fake external command |

### Injectable seams

Small external effects are seamed as function fields so they are testable off a Linux host.
Override them on the struct after construction:

| Seam | Default | Where |
| --- | --- | --- |
| `Manager.Resolve` | `net.LookupHost` | internal/container -- DNS probes in `Check`/`PrepHost` |
| `Manager.GenPSK` | crypto/rand + base64 | internal/container -- redundancy PSK generation |
| `Manager.Geteuid` | `os.Geteuid` | internal/container -- the rootless/rootful guard (returns -1 on Windows, which skips it) |
| `Ops.Hostname` | `os.Hostname` | internal/broker -- node-role detection in `LocalRole` |
| `engine.Runner` | `engine.Exec` | Everywhere -- swapped for `engine.Echo` (dry-run) or a capturing fake |

Filesystem access is *not* seamed: tests use real `t.TempDir()` trees.
