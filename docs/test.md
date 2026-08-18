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

Four packages carry golden files and accept `-update` to regenerate them. Only run it after
eyeballing the diff -- the committed goldens are the reviewed expected output:

```
go test ./internal/render -update
go test ./internal/k8s -update
go test ./internal/convert -update
go test ./internal/cli -update      # rewrites docs/commands.md
```

Every fixture a test reads must be committed. `bash/` is gitignored in its entirety, so no
test may point at it -- a fresh CI checkout has no such files.

## Summary

32 test files, 602 test functions. `TestHelperProcess` in `internal/engine` is not a real
test -- it is the os/exec helper-process shim, a no-op unless `GO_WANT_HELPER_PROCESS=1`.

| Package | Files | Tests |
| --- | --- | --- |
| internal/broker | 4 | 135 |
| internal/k8s | 11 | 98 |
| internal/config | 4 | 97 |
| internal/container | 4 | 93 |
| internal/cli | 4 | 95 |
| internal/convert | 1 | 31 |
| internal/engine | 2 | 25 |
| internal/render | 1 | 17 |
| internal/tools/vulnjudge | 1 | 11 |
| **Total** | **32** | **602** |

## Coverage

Last recorded run, from `scripts/logs/cov.log` (2026-08-18 13:36), total **96.9%**. Re-run
`cov` after any change; these figures go stale the moment tests move, and the previous
total is the floor the next run has to hold.

| Package | Coverage |
| --- | --- |
| internal/tools/vulnjudge | 98.8% |
| internal/config | 98.4% |
| internal/cli | 97.7% |
| internal/broker | 97.3% |
| internal/render | 96.9% |
| internal/convert | 96.9% |
| internal/k8s | 96.2% |
| internal/container | 95.2% |
| internal/engine | see below |

**`internal/engine` is not currently measurable.** It reports `0.0%` in this run and
`100.0%` in the one before it, with every test passing both times -- so the figure is a
measurement artifact, not coverage that vanished. The package's `Exec` tests re-execute the
test binary as a child process (`TestHelperProcess`/`helperCommand`), and a coverage-
instrumented child can clobber the parent's profile; the run also slows to 17s when it
happens. Until that is fixed, treat the total above as understated by roughly engine's
share, and do not read `96.9%` as a drop from `97.2%` -- the difference is almost entirely
this artifact.

---

## internal/config

Config loading, defaults, validation, and env-file resolution, plus the `Command`
type behind the platform CLI overrides and the execution guard that decides what a
`Command` may be, and the scaling block that sizes the broker on every platform. 97 tests across 4 files.

### command_test.go

| Test | What it covers |
| --- | --- |
| `TestCommandUnmarshal` | Both accepted forms of a `Command`: a scalar split on whitespace (reproducing the bash bootstraps' unquoted expansion, so a quoted scalar still splits and whitespace runs collapse), and a sequence kept token-for-token -- the only way to express a token containing a space. Empty scalar, empty list, and an omitted key all decode to nothing, so the default applies |
| `TestCommandUnmarshalRejectsOtherKinds` | A mapping is neither a command line nor an argv, so it fails loud at decode naming the accepted forms |
| `TestCommandUnmarshalPropagatesDecodeErrors` | A node of an accepted kind whose contents still will not decode (`!!binary` with invalid base64; a sequence element that is not a scalar) surfaces yaml's error instead of falling through to an empty command |
| `TestCommandNameAndArgs` | `Name`/`Args` split a command into argv[0] and the leading arguments that precede each call's own, including the unset and bare-binary cases |
| `TestCommandArgsDoesNotAliasCommand` | `Args` allocates: with spare capacity in the backing array, a naive `append(cmd[1:], ...)` would corrupt the previous call's argv. Two successive calls must stay independent and the `Command` itself unchanged |
| `TestCommandString` | Display rendering, used by the check reports and error messages |
| `TestValidateProbeCommandAccepts` | The container health-check probe keeps the loose rules, because it runs inside the broker rather than here: a path, and a shell pipeline with metacharacters, both pass, as does an unset command. The field-by-field opposite of `TestCheckCommandRejects` |
| `TestValidateProbeCommandRejects` | Even the probe rejects empty arguments and control characters (newline, NUL), naming the field and the offending index |
| `TestValidateRejectsBadRuntime` | A malformed runtime fails `Validate` for all three platforms, ahead of the mandatory-field checks, so the message names the runtime rather than the fields also missing |
| `TestRuntimeDefaults` | Defaults resolve to exactly one token with no leading args (`kubectl`/`docker`/`podman`), so existing argv is byte-identical; `k8s.runtime` is defaulted on every platform |
| `TestRuntimeExplicitValueSurvivesDefaults` | A configured override is never overwritten by defaulting |

### execguard_test.go

The execution guard: what a config-declared command may be, and the proof that the
validator and every executor enforce it from one definition.

| Test | What it covers |
| --- | --- |
| `TestGuardConfigIsValid` | Guards the fixture the rest of the file rests on: the untouched `guardConfig` validates cleanly on all three platforms, so a `Validate` failure below can only have come from the guard |
| `TestCheckCommandAccepts` | The shapes an operator legitimately writes: every allowlisted binary bare, a flags-and-values profile, `--flag=value` followed by another flag, a lone `-`, a stripped `.exe`/`.EXE` suffix, a chained runner WITH the escape hatch (`lima podman`, `microk8s kubectl`, `lima --tty=false nerdctl`), and both compose forms including one derived behind a wrapper. Every wrapper here is non-escalating on purpose -- `--allow-command` can never approve sudo and its relatives |
| `TestCheckCommandRejects` | The full reject matrix, each case also asserting the message names the offending token and a way out: an empty or nil command, an unlisted binary (`curl`, `bash`), the right binary on the wrong platform, every path form, a bare word in subcommand position, a bare word after `--flag=value`, `compose` outside the one field and position that permits it, a token after `compose`, a chained runner WITHOUT the hatch, the wrong name in the hatch, an escalation wrapper even when one is forced into the allow-set, the literal `--`, and one case per charset class -- all 21 metacharacters, quotes, backslash, backtick, `$`, control characters, NUL, DEL, seven Unicode space characters beyond ASCII, and four invisible formatting characters (zero-width space and joiner, RTL override, soft hyphen) whose message names the code point since the character cannot be seen -- in argv[0] and in a later token |
| `TestFlagValuePositionIsNotGuaranteed` | Documents the one acknowledged limit as a property rather than a surprise: `docker --tls rm` passes, because flag arity is unknowable, so a token after a value-shaped flag is trusted as that flag's value. The whole charset -- metacharacters, Unicode whitespace, invisible characters -- still applies there, which is the half an adversarial review found incomplete once. Fails deliberately if a future change narrows the limit, so it gets rewritten as a rejection |
| `TestCharsetAgreesAcrossBothYAMLForms` | The regression guard for how that gap arose: a `Command` may be a scalar (split with the Unicode-aware `strings.Fields`) or an explicit sequence (preserved token for token). For ten Unicode space characters, the scalar form must split and the sequence form must refuse -- otherwise two spellings of one config would get two verdicts |
| `TestGuardErrorsAreActionable` | Every guard message is one line, names the field, and names a remedy (the allowlist, the escape hatch, or the specific fix) -- a message that only said "invalid" would leave the operator guessing which of four fields to edit |
| `TestValidatorAndExecutorAgree` | The shared-definition test. Every accept and reject case is driven through BOTH enforcement points -- `Validate`, and the accessor each executor calls before building argv -- and the two must return the same verdict. Fails if anyone ever forks the check |
| `TestExecutorRejectsWithoutValidate` | The reason the check runs twice: a `Config` built in code, which never went through `config.Load`, is still refused by `ClusterCommand`, `RuntimeCommand` and `ComposeCommand` |
| `TestAllowCommandsAccepts` | `--allow-command` is repeatable, each value extends the same set, and `.exe` folds to one entry |
| `TestAllowCommandsRejects` | A bad hatch value is a usage error naming the flag: paths (so the hatch cannot reintroduce the path form layer 2 refuses), metacharacters, whitespace, control characters, empty |
| `TestAllowCommandsRejectsEscalation` | The escape hatch has a floor: `sudo`, `doas`, `su`, `pkexec`, `run0`, `runas`, `gsudo` and their `.exe` spellings can be approved by nobody. Granting one elevates every command the tool issues for the life of an env file, where `sudo solace-util ...` elevates one invocation the operator chose -- so the message must name that alternative, and the name must not be recorded despite the failure |
| `TestEscalationCannotBeAllowedByAnyRoute` | The structural backstop: even with an escalation wrapper forced into the allow-set (a future edit to `execBinaries`, or a caller populating `extraAllowed` directly), `allowed()` strips the category back out -- while a legitimate wrapper in the same forced set still works, proving it is a deny-list and not a broken allow-set |
| `TestAllowedBinaryIsNotGloballyAllowed` | An approval is per-`Config`, so it cannot leak into a second config in the same process |
| `TestComposeCommandDerivation` | `docker.compose` defaults to the runtime's own `compose` subcommand, and `ApplyDefaults` stores exactly what `ComposeCommand` derives -- the two definitions cannot drift |
| `TestComposeDerivationInheritsRejection` | An unlisted runtime cannot become an approved compose command by way of the derivation |
| `TestAllowCommandIsNotASchemaKey` | The structural half of "the config author has no say": every plausible spelling of an allowlist key fails strict decoding, and none reaches the unexported field backing it |

### config_test.go

| Test | What it covers |
| --- | --- |
| `TestPlatformIsContainer` | `Platform.IsContainer` is true for docker/podman, false for k8s |
| `TestRedundancyEnabled` | Only the literal `yes` enables HA; `no`, empty, and junk do not |
| `TestImageRef` | `Image.Ref` joins repo:tag, prefixing the registry only when set |
| `TestParseRole` | Long and short role spellings parse, empty defaults to primary, junk errors |
| `TestRoleNames` | Pins the shell-completion suggestion list to the parser: every name `RoleNames` offers parses, no two name the same role, and all three roles are covered -- a hand-written slice beside a hand-written switch has to fail here, not at a user's TAB press |
| `TestRoleLetter` | Role -> `p`/`b`/`m` letter used in resource names |
| `TestResolveNodeStandalone` | Standalone ignores the role and always resolves the primary as a message-routing node |
| `TestResolveNodeHA` | HA resolves each role to its host name, with the monitor typed `monitoring` |
| `TestContainerRuntime` | Runtime command comes from the platform's block, leading args included; k8s has none |
| `TestContainerBlock` | Podman reads its own container block; everything else falls through to docker's |
| `TestNetworkBlock` | Network block is selected per platform |
| `TestApplyDefaultsK8s` | Every k8s default lands: redundancy, update strategy, admin secret, diag dir, CLI folder, storage, operator image/resources, scaling, ports, anti-affinity. Broker resources now come from the scaling tier instead: `msgNode.cpu` stays empty (it is the removal sentinel, not a value), `msgNode.mem` is the tier-100 default and `Scaling.CPU` its cores |
| `TestApplyDefaultsK8sTLS` | TLS cert/key default only when `tls.serverSecret` is set |
| `TestApplyDefaultsDocker` | Docker defaults (runtime, compose mode, the compose command derived from the runtime, host network, admin user, container name) plus the shared `k8s.*` fields containers reuse |
| `TestApplyDefaultsPodmanRootful` | Rootful podman gets the system quadlet dir, no `--user`, `multi-user.target` |
| `TestApplyDefaultsPodmanRootlessXDG` | Rootless quadlet dir derives from `XDG_CONFIG_HOME`, with `--user` and `default.target` |
| `TestApplyDefaultsPodmanRootlessHomeDir` | Empty `XDG_CONFIG_HOME` falls back to the user home dir branch |
| `TestValidateK8sValid` | A fully populated k8s config validates clean |
| `TestValidateK8sMissingMandatory` | Every missing mandatory k8s field is named in one message, exact wording pinned |
| `TestValidateK8sBadUpdateStrategy` | `k8s.updateStrategy` enum is rejected loud |
| `TestValidateK8sAdminUserFixed` | Mirrors `TestValidateK8sMsgNodeCPURemoved` for a credential: the operator reads the fixed `username_admin_password` key, so a non-`admin` `admin.user` was silently ignored on Kubernetes and is now refused naming that key. `"admin"` and unset both stay legal (unset means `ApplyDefaults` fills it), and docker still accepts any name -- there the username drives the access-level setting, the password file and the SEMP login |
| `TestValidateContainerHA` | A valid HA container config validates for both docker and podman |
| `TestValidateContainerStandalone` | Standalone only requires `nodes.primary.name` among the node fields |
| `TestValidateContainerMissingMandatory` | Missing container fields (image, admin, all three node name/ip pairs) are all named |
| `TestValidateContainerBridge` | `network.mode=bridge` without ports errors; with ports it passes |
| `TestValidateContainerIdentifiers` | Container and node names that reach the compose/quadlet artifact in structural positions are format-checked, so a colon, '=', space or newline is an error instead of a broken artifact. Empty backup/monitor names stay legal in standalone |
| `TestValidateContainerRunUser` | runUser keeps its own `uid[:gid]` pattern -- the default "0:0" carries a colon the identifier check would reject |
| `TestValidateK8sKeyValueEntries` | The "key: value" fragments (loadBalancer.annotations, placement.labels*) must carry a key; a value holding a colon is fine because the renderer quotes both halves |
| `TestValidatePullPolicy` | `image.pullPolicy` enum, including the empty case that keeps the renderer's own IfNotPresent |
| `TestValidatePlacementAffinity` | The additive affinity blocks: unknown operator, missing key, In without values, and a pod term with no topologyKey each fail naming the field; a full valid set passes |
| `TestDefaultK8sPortsMatchesOperator` | The built-in port list is the operator's own 17 entries, led by the tcp-ssh entry this tool used to omit, with no duplicate names |
| `TestApplyBridgePortDefaults` | Bridge mode with no ports defaults to the k8s set as host:container pairs on both platforms; host mode and an explicit list are untouched |
| `TestImageTagVersion` | Tag parsing behind the health-check gate: dotted versions, a `-rc1` suffix, and a two-part tag parse; `latest`, empty, a bare major and a non-numeric tag report *unknown* rather than guessing; `AtLeast` compares major before minor |
| `TestValidateHealthCheck` | The opt-in probe: with no cmd it uses the built-in readiness endpoint, so 10.26+ is accepted while an older tag and an unidentifiable one are both refused (naming the explicit-cmd escape hatch); an explicit cmd skips the version gate but keeps the exec-boundary check; disabled stays legal on any tag |
| `TestValidateContainerBadNetworkMode` | Unknown network mode is rejected loud |
| `TestValidateDockerBadMode` | An unknown `docker.mode` is rejected loud |
| `TestValidateDockerRunModeRemoved` | The removed `run` value gets its own error naming why it went and what to set, not a bare enum message |
| `TestValidateDockerComposeCommand` | `docker.compose` gets the same exec-boundary check as the runtimes: an empty argument is rejected |
| `TestValidateUnknownPlatform` | An unrecognised platform fails rather than validating nothing |
| `TestValidateBadRedundancy` | `redundancy` enum is rejected loud |
| `TestResolveEnvPath` | Env-file lookup over a real temp tree: base dir first, `env/` fallback, base dir shadows `env/`, default name, no extension inference, a path used verbatim with no `env/` retry, a directory is not a match, both candidates named in the not-found error, control characters rejected |
| `TestResolveEnvPathEmptyBaseDir` | An empty base dir means the current directory, for both candidates |
| `TestResolveEnvPathDefaultInBaseDir` | The default name resolves in the base dir before the `env/` fallback is tried |
| `TestLoadSuccess` | A valid file loads, and defaults are applied during `Load` |
| `TestLoadReadError` | A missing file errors with `read env file` |
| `TestLoadParseError` | Malformed YAML errors with `parse env file` |
| `TestLoadUnknownField` | Strict decoding turns a typo'd key into a hard error |
| `TestLoadBashEnvFileHint` | A legacy bash env file is reported as not-YAML and points at `solace-util convert` |
| `TestLoadNotYAMLHint` | Any other non-YAML file says the env file must be YAML and names the schema and the converter |
| `TestLoadUnknownFieldHasNoConvertHint` | A valid-YAML file with an unknown key stays a schema error, without the convert hint |
| `TestLoadValidationError` | A file that parses but fails validation surfaces the missing-fields message |
| `TestLoadResolvesSecretRefs` | An env file carrying no secret at all -- `passEnv` plus a per-user `passwordEnv` -- loads into a fully populated config |
| `TestLoadSecretRefErrors` | Every way a reference fails: unset variable, exported-but-empty variable, both keys set, a `${...}` value where a NAME belongs, and the same on a per-user entry. No message echoes a value |
| `TestSecretRefsLeaveLiteralsAlone` | A literal `${VAR}` password stays exactly that: only the dedicated `*Env` key resolves anything |
| `TestValidateAdditionalUserPasswordCharsetIsK8sOnly` | The one platform-specific rule: k8s puts the password on a CLI line, so the characters the broker rejects there fail validation (naming the character, never the password), while the same env file stays valid for docker and podman, which mount the value as a file |
| `TestValidateAdditionalUsers` | On k8s and docker alike: a valid entry passes, and missing/invalid/duplicate usernames, the built-in `admin`/`monitor` names, a missing or invalid access level, and an empty password all fail. Two usernames differing only in `.`/`_`/`-` are refused too: they fold to one docker host variable name, which would feed one user's password to both |
| `TestValidateAdditionalUserClashesWithAdminUser` | The container-only clash: a listed user matching a configured `admin.user` is refused (two secrets would feed one broker setting) |

### scaling_test.go

The scaling-tier table: `scaling.maxConnections` fixes the broker's CPU on every
platform and defaults its memory, so these cover the table itself, the derivation,
and the two keys the change removed or added.

| Test | What it covers |
| --- | --- |
| `TestScalingTiers` | All five tiers resolve to the published cores and memory, and the case count is asserted against the table so a tier added to one and not the other fails |
| `TestTierForRejectsOffTierValues` | The deliberate absence of rounding: values between, below and above the tiers (including 0 and a negative) resolve to nothing rather than to a neighbour |
| `TestScalingTierListMatchesTable` | The error message's tier list cannot drift from the table -- every listed value is a key, the list is ascending, and its rendering is exact. The package avoids `sort`, so the order is a literal that needs pinning |
| `TestContainerMem` | The one rewrite between the schema's two memory spellings: Kubernetes' `Mi`/`Gi` to the bare `m`/`g` docker and podman accept, leaving an already-container value untouched. Every tier's rewritten default is checked against the validator it would face from an env file, so a default cannot be one the loader rejects |
| `TestApplyScalingTierDefaultsK8s` | A non-default tier derives its cores into `Scaling.CPU` and its memory into `k8s.msgNode.mem`, while `msgNode.cpu` stays empty so `validateK8s` can read any value there as user-set |
| `TestApplyScalingTierDefaultsMemOverride` | The asymmetry the change rests on: an explicit memory survives defaulting on both k8s and container, while CPU is the tier's regardless |
| `TestApplyScalingTierDefaultsContainerBlocks` | Both container blocks are filled whichever container platform is active, matching `applyContainerDefaults`' existing parity |
| `TestApplyScalingTierDefaultsOffTier` | The fail-safe: an unresolvable tier derives nothing rather than inventing a footprint, and `Validate` is what the operator hears from |
| `TestValidateScalingTierRejectsOffTier` | An off-tier value fails on all three platforms with a message listing the five tiers -- the check sits ahead of the platform switch because every platform now renders a CPU limit from it |
| `TestValidateScalingTierAcceptsEveryTier` | Every tier validates cleanly on every platform, so the enum cannot be narrower than the table |
| `TestValidateK8sMsgNodeCPURemoved` | Mirrors `TestValidateDockerRunModeRemoved`: the removed `k8s.msgNode.cpu` still decodes, so the operator gets a reason naming `scaling.maxConnections` and noting `mem` is unaffected, rather than a bare unknown-field error |
| `TestValidateMaxPoolRemoved` | The second folded-away key: `maxPool` named the same broker setting as `maxSpoolUsageMB` under a platform-specific name. It is rejected on all three platforms naming the replacement, and an unset (zero) value does not trip the sentinel |
| `TestValidateContainerMem` | `container.mem` takes docker's and podman's own `b\|k\|m\|g` suffix: the likely mistake (a `Mi` quantity copied from `k8s.msgNode.mem`) is refused naming that trap, alongside bare numbers, decimals and unknown suffixes, while every legal form and the unset case pass |

---

## internal/cli

Command-tree wiring, global flags, confirm prompts, and end-to-end `--dry-run` /
gen-flag passes over the sample env, plus the generated command reference, shell
completion, and the end-to-end behaviour of the execution guard. 95 tests across
four files.

### cli_test.go

| Test | What it covers |
| --- | --- |
| `TestEnvFileLookup` | `-e`/`--env` as the CLI wires it: `env/` fallback, base dir shadowing, no extension inference, the `==> env file:` echo, and long/short flag parity |
| `TestFirstArg` | `firstArg` on nil and populated slices |
| `TestFirstArgOr` | `firstArgOr` falls back on a missing or empty first argument |
| `TestNotImplemented` | The placeholder error names the command and says "not implemented yet" |
| `TestEmit` | `emit` writes bytes to stdout unchanged |
| `TestWarnAndStep` | `warn` and `step` write `[WARN]` / `==>` lines to stderr |
| `TestAnnounceCommandsNamesResolvedBinaries` | The preamble that replaced the per-call `exec:` line: each binary the env file names is resolved and printed once as `==> using <name>: <path>`. k8s announces the cluster CLI; docker announces one line when `compose` is the runtime's own subcommand and two when a standalone `docker-compose` is configured; a name that resolves nowhere is skipped in silence, since a report must not invent a failure the first real execution already reports. Hermetic -- a stub binary is written into a temp dir put at the front of `PATH`, so the expected path is exact and no test host needs kubectl or docker |
| `TestBinaryAnnouncementWiring` | The same through the real command tree, with the stub named `kubectl` so it is the schema default and needs no `--allow-command` (which is itself refused where nothing executes). A real run announces before it works; `--dry-run`, `--gen-only` and `gen` announce nothing -- `--dry-run` is documented to need no runtime binary installed at all |
| `TestVerboseFlagTracesEveryCommand` | `-v` prints `==> exec: <path> <args>` per call, a run without it prints none, and `--dry-run -v` still works (Echo already echoes every command) |
| `TestTreeStructure` | Every platform and a representative set of leaf command paths exist in the tree |
| `TestFlagsRegistered` | Per-command flags (`purge`/`clear-data`/`keep-data`, `keep-yaml`, `days`, `pod`, `dir`) are registered where expected |
| `TestHelpNoConfig` | `--help` at root and per platform short-circuits before config load, so no env is needed |
| `TestGenWired` | Every render-only path emits the right artifact: k8s CR (`apiVersion:`), Secret manifests, compose (`services:`), quadlet (`[Unit]`), the container secret script, and the container env file -- via both the `gen` command and the `--gen-*-only` flags |
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
| `TestK8sErrorPaths` | k8s handler boundaries: missing cert/key, no product keys, failed login, missing exec-cli file, bad `--pod`, `--gen-only` rejected on non-artifact commands, `--gen-env-only` rejected as having no k8s equivalent, mutually exclusive data flags |
| `TestConfirmFlagShortcuts` | `--yes` confirms a delete; `--purge`/`--keep-data` drive retention without reading stdin |
| `TestConfirmNonTTY` | Without a TTY and without `--yes`, delete refuses and purge keeps data |
| `TestPromptYesNo` | Lenient delete prompt: `y`/`yes` in any case accept, everything else declines |
| `TestPromptYes` | Strict purge prompt: only an exact trimmed `yes` accepts; a bare `y` does not |
| `TestErrorPaths` | Global rejections: unresolvable env file, invalid node roles across container and k8s leaves, unknown `gen` target |
| `TestDockerRunModeRejected` | An env file still carrying `docker.mode: run` fails with the removal reason and points at `docker.compose` |
| `TestK8sGenSecretsWired` | The Secret manifests render through both the `gen secrets` target and `--gen-secrets-only`, on the standalone env rather than the HA sample whose tls.serverSecret points at cert files absent from a checkout |
| `TestGenSecretsRefusesEmptyValue` | The printed script invites execution, so it is refused when running it would create an empty secret -- while `--gen-only` stays renderable, since the deploy artifact only references secrets by name |
| `TestGenFlagsAreExclusive` | Two gen flags at once is a loud error, not a silent precedence rule |
| `TestCtrGenFlagsRejectedOnNonArtifactCommands` | Every gen flag on every non-artifact container command (`delete`, `down`, `status`, `check`) fails loud -- being ignored on `delete` would run the real delete while the user believed they asked for a render |
| `TestGenNeverLeaksSecrets` | End-to-end: `--gen-only` and `--gen-env-only` output on both container platforms omits the admin password, while `--gen-secrets-only` carries it (it is what creates the secret) |
| `TestCtrVerifyAll` | `verify` role arms: unknown host fails loud, this-host-is-monitor skips redundancy, standalone skips redundancy |
| `TestCtrConfigAllArms` | `config` with every optional step configured runs all three gated arms, and the private key never reaches stdout |
| `TestCtrExecCLIPathSeparator` | An exec-cli argument containing a separator is used as-is, not joined under the CLI scripts folder |
| `TestConvertToStdout` | `convert` writes YAML to stdout and its warnings to stderr, so the artifact stays clean |
| `TestConvertToFile` | `-o` writes the file, a second run refuses to clobber it, and `--force` overrides |
| `TestConvertRoundTrip` | A converted file loads: `-e` against it drives a real command |
| `TestConvertErrorPaths` | Bad `--platform`, a missing source file, and a missing argument all fail loud |
| `TestVersionPrintsStampedValue` | `version` reports whatever the dev scripts' `-X` flag (or a test) set the package var to, verbatim -- the contract that aligns a release binary with its git tag |
| `TestVersionDefaultsToDev` | An unstamped build (plain `go build .` or `go test`) reports "dev" |
| `TestVersionIncludesToolchainAndPlatform` | Output carries `runtime.Version()` and GOOS/GOARCH, for support triage |
| `TestVersionRejectsArgs` | `version` takes no arguments |
| `TestBashEnvGivenToEnvFlag` | Pointing `-e` at a legacy bash file reports not-valid-YAML and names `solace-util convert` |
| `TestExecute` | `Execute()` builds the tree and runs `--help` without error |
| `TestK8sConfirmDeclined` | a non-interactive `k8s delete`/`k8s down` (no --yes) makes zero cluster calls, using the new App.Interactive seam instead of ambient stdin |
| `TestK8sRestartConfirmGate` | a non-interactive `k8s restart` (all or one role) bounces nothing, and a bad role is rejected before any prompt |
| `TestCtrConfirmDeclined` | a non-interactive `docker delete` (no --yes) issues zero runtime calls |
| `TestIsTTYClosedFile` | isTTY treats a stream it cannot Stat (a closed file) as non-interactive rather than risking a blocked prompt |
| `TestCtrManagerConfirmWiring` | ctrManager wires Manager.Confirm to confirmRestart, and a non-interactive session (via App.Interactive) declines without reading a prompt |
| `TestK8sLoginOutcomes` | a transport failure propagates as an error and a canned 200 OK response returns nil, the two real SEMP outcomes engine.Echo's fixed (nil,nil) can never produce |
| `TestCtrLoginOutcomes` | same as TestK8sLoginOutcomes for the container login path |
| `TestOpK8sConfigAllArms` | with every optional step configured on an HA env, opK8sConfigAll actually runs all five previously-unreached arms (not just the always-run hardening steps) and no secret value reaches an argv |
| `TestOpK8sConfigAllAborts` | failing any one step's underlying command aborts before the next step's command is ever issued, pinning the function's documented harden-then-provision ordering |
| `TestOpK8sUpAssertsLeaderOnHA` | on an HA config, `up` asserts the config-sync leader as its last step rather than stopping after deploy |
| `TestOpK8sUpAborts` | failing any one step aborts before the next step's command runs, closing the 'every step aborts loud' claim in the docstring |
| `TestOpK8sPrepAllAborts` | same abort-ordering property as opK8sUp, on the four-step prep sequence |
| `TestOpK8sDownAborts` | a failed broker- or secrets-deletion stops before the namespace is removed out from under it |
| `TestOpCtrConfigAllAborts` | same abort-ordering property as the k8s config-all cluster, on the container-side sequence |
| `TestOpCtrVerifyAllRunsRedundancyLocal` | when this host is the primary/backup (not the monitor), opCtrVerifyAll actually calls RedundancyLocal instead of only ever hitting the skip/reject arms |
| `TestK8sVerifyAllRedundancyUnhealthy` | verify stops before the SEMP login when the redundancy check itself reports unhealthy, on the existing HA sample under --dry-run |
| `TestK8sTeardownDomainCertsConfigured` | with a CA actually configured, teardown domain-certs issues a kubectl exec instead of self-skipping (every other test's map is empty, so the conversion was correct only by vacuity) |
| `TestConvertParseError` | a malformed legacy env file (unterminated array assignment) surfaces the parser's own error through the CLI |
| `TestConvertWriteError` | an -o path whose parent directory is absent fails with a wrapped write error naming the path |
| `TestK8sGenSecretsMissingCertFile` | tls.serverSecret configured with an unreadable tls.cert fails loud naming the read failure, instead of only being caught at real deploy time |

### allowcommand_test.go

The execution guard end to end -- flag parsing, `config.Load`, `Validate`, and the
executors -- driven through the real command tree.

| Test | What it covers |
| --- | --- |
| `TestAllowCommandIsRegisteredOnPlatformTrees` | `--allow-command` is declared on `k8s`/`docker`/`podman` and inherited by their leaves, is NOT a root flag, and is a usage error on `solace-util convert`, which loads no config and runs no platform CLI |
| `TestAllowCommandIsRepeatable` | Both values of a repeated flag are collected in order, so a chain needing two approvals does not force a choice |
| `TestAllowCommandApprovesAWrappedRuntime` | The accept case end to end: `microk8s kubectl` is refused with a message naming the hatch, and runs -- reaching the echoed command -- once the operator passes `--allow-command microk8s` |
| `TestAllowCommandRejectsBadValues` | A path, a metacharacter, an empty value, or any privilege-escalation wrapper is a usage error, so the hatch cannot reintroduce what the guard refuses |
| `TestEscalationIsRefusedEndToEnd` | An env file naming `sudo kubectl` is refused with and without the flag, and the refusal names the supported alternative -- elevate the tool itself, at the moment you run it |
| `TestAllowCommandRejectedWhereNothingExecutes` | The flag is refused on `gen` and under `--gen-*-only`, so it is never learned as harmless boilerplate that later gets pasted into a run that does execute |
| `TestHostileRuntimeIsRefusedByEveryVerb` | An unlisted binary stops `check`, `status`, `deploy`, `delete`, `show-all` and `logs` alike -- `status` running `curl` is the same arbitrary execution `deploy` running it would be |
| `TestSmuggledSubcommandIsRefused` | `kubectl delete`, `kubectl delete ns prod` and a literal `--` in the config are all refused: this tool appends its own subcommand, and a word there would run ahead of it |
| `TestPathRuntimeIsRefused` | The bare-name rule end to end for relative, absolute, and parent-directory forms -- the `./kubectl` shipped beside the env file |
| `TestGenPathNeverExecutes` | Backs the trust-model promise that rendering an untrusted env file is safe: the `gen` path issues no external command at all |

### commanddoc_test.go

| Test | What it covers |
| --- | --- |
| `TestCommandDocs` | Renders the command reference from the live tree and fails while `docs/commands.md` is stale -- the drift gate for every command path, positional, flag, and `Short` string. This file is also the generator: `-update` rewrites the doc |

### completion_test.go

Shell completion end to end. The value tests drive cobra's hidden `__complete`
endpoint through the real tree -- the same request a loaded completion script makes
on every TAB press -- via the `runComplete` helper, which cannot reuse `runRoot`
because that discards cobra's own writer. Nothing here loads an env file: the
platform `PersistentPreRunE` never runs for `__complete`, which is what keeps a TAB
press from parsing config or executing anything.

| Test | What it covers |
| --- | --- |
| `TestCompletionScriptsGenerate` | Each of bash/zsh/fish/powershell emits its own script, matched on the line that actually binds the completer to `solace-util`, so a script that generated but wired up nothing still fails |
| `TestCompletionNoDescriptions` | `--no-descriptions` is honoured on every shell: the generated script requests `__completeNoDesc` instead of `__complete`, and does not without the flag |
| `TestCompletionNeedsAShell` | An unsupported shell, or none at all, fails loud with nothing on stdout -- the reason the parent carries a `RunE`, since cobra answers a non-runnable command by printing help to stdout and exiting 0, which would put help text into `solace-util completion tcsh > solace-util.ps1` and call it a success |
| `TestCompletionHelpStillWorks` | `--help` short-circuits ahead of that `RunE`, so asking how to use the command is not itself an error |
| `TestEnvFlagCompletesEnvFiles` | `-e` is completed from the two directories `config.ResolveEnvPath` searches, by bare name: base dir first, the shadowed `env/` copy of the same name offered once, and a non-YAML file not suggested |
| `TestEnvFlagPrefixFilters` | A partial name narrows the suggestions instead of returning every env file |
| `TestEnvFlagWithPathDefersToShell` | A value carrying a directory resolves verbatim, so completion returns the default directive and hands back to the shell rather than offering bare names that would resolve elsewhere |
| `TestRoleArgsComplete` | Every `[role]` positional offers `primary`/`backup`/`monitor` -- the ones built through `roleLeaf` and the ones assembled inline on both platform trees |
| `TestPodFlagCompletesRoles` | `--pod` completes to the same role set as the positionals, not to filenames |
| `TestPlatformFlagCompletes` | `convert --platform` offers exactly what `convertPlatform` accepts, minus the empty detect value |
| `TestDirFlagCompletesDirectories` | `--dir` asks the shell to filter to directories, on both platform trees |
| `TestNoArgsLeafOffersNoFiles` | A command built by `leaf` offers nothing, stopping cobra's filename fallback on the majority of commands in the tree. The list covers the no-arg commands that carry a flag or a `--gen` annotation too (`deploy`, `delete`, `down`, `diagnostics`, `check`, `status`, `show-all`, `up`): each used to hand-roll the literal `leaf` already builds, and so shipped without `NoFileCompletions` -- they now go through `leaf` and attach the extra afterwards |
| `TestAllowCommandOffersNoFiles` | `--allow-command` offers no files on any platform: the value is a bare binary name, and paths are what its own help text warns against |
| `TestFlagCompletionsRegistered` | The drift gate: every flag that should have a completion function still has one, since a renamed flag silently reverts to filename completion at a TAB press and no other test would notice |

---

## internal/convert

The legacy bash env -> YAML converter: a shell-assignment parser, the variable
mapping, and the YAML emitter. 31 tests.

### convert_test.go

| Test | What it covers |
| --- | --- |
| `TestConvertLegacyK8sEnv` | `testdata/legacy-k8s.env` converts end to end and matches `testdata/legacy-k8s.yaml.golden`: platform detected as k8s, `true` -> `yes`, every scalar/array/associative value mapped, `${SOLBK_NS}` expanded, a trailing comment stripped, an explicit `0` kept, an empty PSK omitted, a multi-word `KUBE` preserved as `k8s.runtime` argv, and only the two expected advisories. The fixture sets `SOLBK_MSGNODE_CPU`, as every real legacy file does, so the drop is exercised here: it warns, and no `cpu` reaches the YAML |
| `TestConvertUserPasswordsBecomeAdditionalUsers` | The one legacy variable with no like-for-like successor: `SOLBK_USR_PASS` becomes structured `admin.additionalUsers` entries with the least-privileged `accessLevel: none` plus a warning naming that choice, malformed entries are dropped with a warning naming their POSITION and never their text (a malformed entry is most likely a bare password), and `Convert` re-validating its own output proves the emitted level is a legal one |
| `TestConvertAdminUserIsContainerOnly` | The one admin field that is not portable: `SOLBK_ADM_USER` is emitted only for docker/podman, and on a k8s target is dropped with a warning naming why (`validateK8s` refuses any non-`admin` value), stays out of the generic unmapped list because it is still read, and leaves a document that validates -- no "will not load as-is". A source that already said `admin` warns about nothing |
| `TestConvertContainer` | A container env file maps the node table, container block, ulimits, network, and spool scaling |
| `TestConvertPlatformDetection` | Podman markers, docker markers, and both-present all resolve to the expected section |
| `TestConvertPodmanSection` | Podman rootless and quadlet dir land in the podman block, and no docker block is written |
| `TestConvertExplicitPlatformWins` | `--platform` overrides detection and suppresses the detection warning |
| `TestConvertUnmappedVariablesWarn` | Variables with no YAML equivalent are named in the warnings, not dropped silently |
| `TestConvertBashPlumbingIsSilent` | Bootstrap-only variables (`EXDIR`, `GENONLY`) are dropped without noise |
| `TestConvertKubeMapsToK8sRuntime` | `KUBE` becomes `k8s.runtime` in every shape it carried: a drop-in (`oc`), a wrapper (`microk8s kubectl`), a `--kubeconfig` profile, and an absolute path -- no warning |
| `TestConvertKubeEchoIsDropped` | `KUBE="echo"` was the bash dry-run trick, so it warns pointing at `--dry-run` and emits no `runtime`, rather than becoming a runtime that no-ops every command |
| `TestConvertKubeSilentOnContainerPlatform` | `KUBE` belongs to the k8s bootstrap: a container conversion consumes it silently and never emits `k8s.runtime` |
| `TestConvertRedundancySpellings` | `true`/`yes` and `false`/`no` normalise (any case); anything else copies through with a warning |
| `TestConvertRedundancyOmitted` | An unset SOLBK_REDUNDANCY emits no key either way, but a container source is warned that its bootstrap defaulted to HA while this CLI defaults to standalone; a k8s source stays silent because the defaults already agree |
| `TestConvertDockerRunModeWarns` | DOCKER_MODE=run is dropped with the removal reason rather than carried over to fail validation later |
| `TestConvertBadNumberWarns` | A non-numeric value for a numeric field warns and is not written |
| `TestConvertSpoolVariablesUnify` | Two legacy names for one key: the k8s bootstrap's `SOLBK_SCALING_MAXPOOL` and the container one's `SOLBK_SPOOL_MAXUSAGE` both map to `scaling.maxSpoolUsageMB`, each platform's own name wins when both are set, and the warning says which was used rather than picking in silence |
| `TestConvertOffTierMaxConnWarns` | `SOLBK_SCALING_MAXCONN` was any integer and is now one of five tiers. An off-tier value is still written -- rewriting the operator's declared load would be worse than reporting it -- and `Convert` re-validating its own output is what surfaces it, so this needs no mapping code of its own |
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
state machines, and the node-local HA variants. 135 tests across 4 files.

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
| `TestProductKeysRejectsMultilineKey` | A product key with a newline or CR is rejected before anything is uploaded -- it would append commands to a CLI script already running as admin -- while an opaque vendor key with `+`, `/` or `=` is accepted, since a key's alphabet is not this tool's to constrain |
| `TestAdditionalUsers` | The k8s-only user-creation op: the generated script is uploaded verbatim, the password never reaches an argv or the shown output (the CLI transcript repeats it, so it is withheld), and the uploaded script is deleted afterwards because its body carries every password |
| `TestAdditionalUsersReportsExistingUser` | The deliberate non-idempotency: a broker-reported `already exists` fails loud naming that as the likeliest cause, the error carries no transcript, and the script is still removed on the failure path |
| `TestAdditionalUsersEmpty` | No users configured is an error |
| `TestAdditionalUsersRejectsBadValues` | Every injection and CLI-quoting path is refused before anything is uploaded -- a username with a space, a multiline access level, an empty password, and a password containing a newline or any of the characters the broker rejects inside a quoted value -- and no error echoes the password. Punctuation the CLI does accept still works |
| `TestRemoveDomainCerts` | The removal half of the domain-CA pair emits a script naming the CA (previously untested, and now reachable from both platforms) |
| `TestRemoveDomainCertsRejectsBadName` | A CA name with a space is rejected before any upload |
| `TestRemoveDomainCertsEmptySkips` | No configured CAs makes no calls |
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
| `TestRunCLIUploadError` | RunCLI wraps and returns an Upload failure with the script name and never invokes the CLI binary |
| `TestServerCertUploadError` | ServerCert stops before apply-server-certs when the certificate bundle upload fails |
| `TestServerCertRunCLIError` | ServerCert returns the apply-server-certs error and never shows output |
| `TestDomainCertsUploadFileError` | DomainCerts stops before load-domain-certs when a CA file upload fails |
| `TestDomainCertsRunCLIError` | DomainCerts returns the load-domain-certs error and never shows output |
| `TestDisableDefaultVPNDisableError` | DisableDefaultVPN stops before reading back show-vpn when disabling the VPN fails |
| `TestDisableDefaultVPNShowError` | DisableDefaultVPN surfaces a failed show-vpn readback and skips cli-script cleanup |
| `TestDisableDefaultUsersShowVPNError` | DisableDefaultUsers stops before disabling anything when the VPN listing fails |
| `TestDisableDefaultUsersDisableError` | DisableDefaultUsers surfaces a failed disable-default-usernames call rather than reporting success |
| `TestProductKeysRunCLIErrorStopsLoop` | ProductKeys stops its per-role loop on a transport failure and never touches the remaining role |
| `TestAdditionalUsersRunCLITransportError` | a hard RunCLI transport failure in AdditionalUsers still removes the uploaded password-bearing script |
| `TestExecCLIUploadFileError` | ExecCLI reports a failed upload by the script's basename and skips exec/cleanup entirely |
| `TestRemoveDomainCertsRunCLIError` | RemoveDomainCerts fails loud on a rejected removal and skips the cleanup rm |
| `TestLoginTransportError` | Login returns a wrapped transport error instead of logging 'Login failed' as if it got an HTTP response |
| `TestLeaderRevertActivityError` | Leader aborts before polling redundancy when the initial Backup revert-activity fails |
| `TestLeaderAssertLeaderError` | Leader returns the assert-leader error after a healthy poll without showing partial output |
| `TestRedundancyReleaseError` | Redundancy aborts immediately when releasing activity to the Backup fails, never querying the Backup |
| `TestRedundancyBackupShowError` | Redundancy aborts when its own post-release Backup show-rd read fails, after releaseToBackup already succeeded |
| `TestRedundancyRevertToPrimaryError` | Redundancy surfaces a revertToPrimary failure rather than declaring the drill successful |
| `TestReleaseToBackupReleaseError` | releaseToBackup stops before any show-rd poll when the initial release exec fails |
| `TestReleaseToBackupReleasedTimeout` | releaseToBackup times out and never sends no-release when the release never converges |
| `TestReleaseToBackupNoReleaseError` | releaseToBackup surfaces a failed no-release exec instead of proceeding to the un-released poll |
| `TestReleaseToBackupUnreleasedTimeout` | releaseToBackup times out rather than declaring the Backup active when un-release never converges |
| `TestRevertToPrimaryRunCLIError` | revertToPrimary stops before entering its poll when the Backup's revert-activity exec fails |
| `TestShowRDPairPrimaryError` | showRDPair returns empty strings and the Primary's error without ever querying the Backup |
| `TestDiagnosticsMkdirError` | Diagnostics fails loud with the destination path when it cannot create the diagnostics dir |
| `TestGatherNodeDownloadError` | a failed main-archive Download fails the whole node's diagnostics gather with no local file produced |
| `TestGatherNodeBundleDownloadWarnsOnly` | a failed diagnostics-bundle download only WARNs, naming the bundle, and Diagnostics still succeeds |
| `TestGatherNodeBundleCleanupWarnsOnly` | a failed bundle-cleanup rm only WARNs and Diagnostics still succeeds overall |

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
| `TestAdditionalUsersScript` | Exact script text for the k8s user-creation run: both values quoted so a password with a space survives, one `create username ... password ...` plus `global-access-level` per user, and no trailing `show` whose output would be discarded anyway |
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
| `TestLocalRoleHostnameError` | LocalRole wraps and returns a Hostname read failure instead of matching a garbage host against the node table |
| `TestLeaderLocalBadRoleArg` | LeaderLocal propagates an invalid explicit role arg before the primary-only guard runs, making no transport calls |
| `TestLeaderLocalPollCondError` | LeaderLocal aborts on a mid-poll transport error while still dumping show-redundancy-detail, closing an asymmetry with the k8s Leader |
| `TestLeaderLocalAssertLeaderError` | LeaderLocal returns the assert-leader error after a healthy poll without showing output |
| `TestRedundancyLocalPrimaryReleaseError` | redundancyLocalPrimary stops before any show-rd poll when the release exec fails |
| `TestRedundancyLocalPrimaryReleasedTimeout` | redundancyLocalPrimary times out and never sends no-release when release-to-Backup never converges |
| `TestRedundancyLocalPrimaryNoReleaseError` | redundancyLocalPrimary surfaces a failed no-release exec instead of proceeding to the un-released poll |
| `TestRedundancyLocalPrimaryUnreleasedTimeout` | redundancyLocalPrimary times out rather than waiting for the Backup fail-back on stale un-released state |
| `TestRedundancyLocalBackupBecomeActiveTimeout` | redundancyLocalBackup times out and never reverts a handshake that never started |
| `TestRedundancyLocalBackupDwellCancelled` | a cancelled context during the mandatory ActiveDwell hold aborts redundancyLocalBackup with context.Canceled before it reverts |
| `TestRedundancyLocalBackupRevertActivityError` | redundancyLocalBackup surfaces a failed revert-activity exec instead of entering the final poll |
| `TestRedundancyLocalBackupStandbyTimeout` | redundancyLocalBackup times out rather than declaring success when it never reports returning to standby |

---

## internal/k8s

Everything driven through `kubectl`: the read-only permission preflight, prep, deploy,
operator, day-2 ops, secrets, and the pod transport. 98 tests across 11 files.

### names_test.go

| Test | What it covers |
| --- | --- |
| `TestResourceNames` | Pod, PVC, StatefulSet, and load-balancer service names for every role |
| `TestRestartOrder` | The safe manual-bounce order (monitor, backup, primary; standalone just the primary) |
| `TestHARoles` | HA yields all three roles; standalone yields only the primary |
| `TestProductKeyRoles` | Product keys target primary+backup in HA, primary only in standalone |

### runtime_test.go

| Test | What it covers |
| --- | --- |
| `TestClusterHonoursRuntime` | Every `Cluster` helper (`kubectl`, `apply`, `deleteStdin`, `output`, `interactiveExec`) runs argv[0] from `k8s.runtime` and places its leading arguments ahead of the subcommand |
| `TestTransportHonoursRuntime` | The pod transport does the same for `exec`, `exec -i`, the stdin `Upload`, and both `cp` directions |
| `TestExecutorRefusesUnapprovedRuntime` | The executor half of enforce-twice: a `Cluster` and a transport built straight from a `*config.Config` that never went through `config.Load` still refuse an unapproved `microk8s kubectl`, on all eleven paths that could reach exec -- and hand the runner nothing at all, since refusing after the call would mean the binary already ran |
| `TestRuntimeDefaultArgvUnchanged` | With the default runtime the argv is exactly what the old hardcoded `kubectl` constant produced -- the regression guard for every existing `+ kubectl ...` assertion |

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
| `TestCheckEnvNoSecretLeak` | The config report shows secrets as set/MISSING and never prints their values, and the operator image line carries the `image.registry` prefix the apply adds -- the report named a bare `Operator.Image` the deploy would never pull |
| `TestCheckOperatorNS` | The line `CheckEnv` cannot print, since discovery needs a live cluster: a configured `k8s.operator.namespace` reports its origin with no cluster call, a scripted deployment row reports `discovered on the cluster`, and both "no operator row" and a refused lookup fall back to the default with wording that does not claim the operator is uninstalled. Under `--dry-run` the line skips instead of reporting an unresolved default |
| `TestReachable` | The API-server probe argv, and failure when it errors |
| `TestCheckStorageClass` | A suitable configured class passes with no default lookup; Immediate binding or no expansion is rejected; missing attributes reject; dry-run skips the assertions |
| `TestResolveStorageClass` | A configured class short-circuits; a single default resolves; multiple defaults error; no default returns empty |
| `TestCheckDryRun` | The whole preflight runs clean under Echo and emits the report plus the skip note |
| `TestCheckAbortsWhenUnreachable` | Check aborts before CheckStorageClass wastes a round-trip when the API server is unreachable, and wraps the error with "cannot reach" |
| `TestCheckEnvSparseConfig` | CheckEnv prints MISSING/none/not-configured fallbacks correctly when the corresponding config fields are unset, not just the all-set sample fixture -- including the watch fallback, which used to claim "(broker namespace only)" for the one input (no list + `watchBrokerNs: false`) that renders an empty `WATCH_NAMESPACE` and so makes the operator watch every namespace |

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
| `TestLabelNodesHappyPath` | RBAC precheck (now the shared `Preflight`, which tells "not allowed" from "nobody answered"), node list, then the label call for the selected node |
| `TestLabelNodesReprompt` | Out-of-range and non-numeric selections re-prompt before the correct node is labelled |
| `TestLabelNodesRBACDenied` | A failed RBAC precheck aborts and labels nothing |
| `TestLabelNodesEOFNoSelection` | EOF with no selection errors |
| `TestCreateSecretsFailsWithoutAdminFields` | CreateSecrets can pass secretPreflight (TLS-only) and still fail loud inside GenSecrets when admin.pass/k8s.adminSecret are unset, with zero applies made |
| `TestCreateSecretsStopsOnPreflightFailure` | A refused `auth can-i create secrets` stops CreateSecrets before GenSecrets reads the TLS private key off disk -- loading key material for a cluster that will not accept it is work worth not doing |
| `TestGenSecretsTLSError` | GenSecrets itself (not just via CreateSecrets' preflight) fails when tls.serverSecret is set but the cert files are unreadable, guarding `prep secrets --gen-secrets-only` |
| `TestDeleteSecretsSkipsUnconfiguredAdminSecret` | DeleteSecrets never issues `kubectl delete secret ""` when k8s.adminSecret was never configured |
| `TestDeleteSecretsStopsOnError` | A genuine delete failure stops the teardown loop and surfaces instead of silently continuing to the remaining secrets |
| `TestLabelNodesHAOnlyPrimaryConfigured` | In an HA config with only LabelsPrimary set, backup and monitor are read from their own config fields (not Primary's) and are silently skipped when empty, never prompted |
| `TestNodeNamesError` | nodeNames fails loud with its own wrap on a genuine query failure instead of returning a misleadingly empty list |
| `TestLabelNodesNoNodesFound` | LabelNodes fails loud when RBAC passes but the cluster reports zero nodes, instead of misbehaving in promptNode with an empty list |
| `TestLabelNodesLabelFailureIsNonFatal` | A single failed label application is reported and skipped, not fatal, matching the doc comment's stated contract |
| `TestCreateNamespaceApplyFails` | A failing apply (RBAC denial) surfaces from CreateNamespace instead of being silently swallowed -- previously impossible to test since RunInput unconditionally returned nil |

### preflight_test.go

| Test | What it covers |
| --- | --- |
| `TestCanIAnswerReadsTheLastLine` | The verdict is the LAST non-empty line, not the whole output: `kubectl auth can-i` prints advisory lines above it on stdout ("Warning: resource 'x' is not namespace scoped"), and comparing the whole body would turn every such cluster into the unreadable-answer branch -- a preflight failing safe in the wrong direction. Covers plain yes/no, no trailing newline, one and several warnings, blank lines, CRLF, empty, and whitespace-only |
| `TestPreflightAcceptsAWarnedYes` | The end-to-end of the above: a cluster that warns and then permits lets the deploy proceed |
| `TestPreflightRefusesAnUnreadableAnswer` | Exit 0 with neither yes nor no -- a wrapper that swallowed stdout -- is refused rather than assumed permitted, since proceeding would act on a permission nobody confirmed |
| `TestPreflightIsPreviewableUnderDryRun` | `--dry-run` echoes the probe and skips its assertion, so previewing needs no cluster -- which is why there is no skip flag |

### deploy_test.go

| Test | What it covers |
| --- | --- |
| `TestDeployBrokerApply` | One apply on stdin carrying the rendered CR |
| `TestDeployBrokerKeepYAML` | `--keep-yaml` writes `.broker.yaml` byte-identical to what was applied |
| `TestDeleteBrokerNoPurge` | Without purge only the CR is deleted, no PVCs |
| `TestDeleteBrokerPurgeHA` | Purge deletes the CR plus all three role PVCs |
| `TestDeleteBrokerPurgeStandalone` | Purge on standalone deletes the CR plus the single PVC |
| `TestDeleteBrokerPurgeSwallowsPVCError` | A failing PVC delete is best-effort: teardown continues and every PVC is still attempted |
| `TestDeployBrokerKeepYAMLWriteError` | DeployBroker fails loud and never applies the manifest when writing .broker.yaml fails, instead of silently proceeding as if the file were saved |
| `TestDeployBrokerStopsOnPreflightFailure` | The preflight ordering guarantee: when `auth can-i` answers no, no manifest is written and no call follows the probe -- without this the probe would be decoration |
| `TestPreflightUnreachableClusterHints` | An unreachable API server is a different failure from an RBAC refusal and gets the hint that helps (`log in first`, `oc login`), carrying kubectl's own error rather than replacing it; still nothing runs after the probe |

### operator_test.go

| Test | What it covers |
| --- | --- |
| `TestWatchNamespace` | `WATCH_NAMESPACE` joins: broker namespace appended by default, onto a configured list, or omitted when disabled -- plus the dedupe half, since a list that already named the broker namespace (the common case, `watchBrokerNs` defaults on) listed it twice in the report and the applied Deployment. Entries are trimmed, empties and trailing commas dropped, repeats inside the list collapsed, first occurrence winning. controller-runtime's map-keyed cache hid the repeat at runtime, so only these cases can catch a regression |
| `TestOperatorImage` | The registry-prefix rule now shared by `RenderOperator` and `CheckEnv`: prefixed when `image.registry` is set, raw when it is not. Its own test rather than only being reached through the 119 KB bundle render, because the report and the apply drifted for exactly as long as each owned a copy |
| `TestRenderOperatorSubstitutions` | Every substitution point lands (namespace, watch list, image with/without registry prefix, resources, pull secret) and no template marker survives |
| `TestGenOperator` | Render-only uses the configured operator namespace, or the fixed default when unset |
| `TestOperatorApply` | regcred is applied into the operator namespace first, then the bundle, both on stdin |
| `TestOperatorApplyNoPullSecret` | With no pull secret only the bundle is applied |
| `TestOperatorDelete` | Teardown deletes the rendered bundle with `--ignore-not-found` |
| `TestOperatorLogsArgs` | Log passthrough targets the operator deployment |
| `TestOperatorStatus` | OperatorStatus issues the deployment-wide get then the controller-pods get in order, and stops after the first if it fails |
| `TestOperatorDescribe` | OperatorDescribe issues `describe deployment/<name> -n <opNS>` against the resolved operator namespace |

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
| `TestRestartPod` | The manualPodRestart step: delete the role's pod with --ignore-not-found, then wait for the statefulset within the bounded rollout timeout |
| `TestRestartRolling` | HA bounces monitor -> backup -> primary in that order, standalone only the primary, and a pod that does not come back stops the sequence before the next role |
| `TestReplicasStart` | HA scales and waits for all three roles, standalone only the primary, and a stuck rollout fails loud at the first role |
| `TestReplicasStop` | HA scales all three to zero, standalone only the primary |
| `TestStatusFailureStopsEarly` | Status stops at whichever get fails first (pods, or pods+svc) instead of continuing to the remaining queries |
| `TestRestartPodDeleteFails` | A failing pod delete surfaces its own actionable message and never reaches the rollout-status wait |

### secrets_test.go

| Test | What it covers |
| --- | --- |
| `TestSecretGoldens` | Rendered admin, TLS, docker-registry, and operator-regcred secrets match their committed goldens |
| `TestAdminSecretDecodes` | The base64 data round-trips to the expected plaintext passwords |
| `TestAdminSecretExcludesAdditionalUsers` | The finding that shaped the k8s user path: the operator reads only the admin and monitor keys, so an additional user's name and password (plain and base64) must be absent from this Secret entirely |
| `TestAdminSecretErrors` | Empty password, empty `k8s.adminSecret`, and an additional user with no name, a bad name, or no password all error |
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

The host-local Docker/Podman manager, its node-local transport, and the engine
preflight that precedes every mutating operation. 93 tests across 4 files.

### runtime_test.go

| Test | What it covers |
| --- | --- |
| `TestManagerHonoursRuntime` | The manager's shell-outs (`run`, `output`, `CLI`, `Shell`) run argv[0] from `docker.runtime` with its leading arguments ahead of the subcommand -- the bash bootstrap expanded `${CONTAINER_RUNTIME}` unquoted, so a wrapper like `sudo -n docker` has to reach exec as argv |
| `TestManagerReachableProbesRuntimeThenCompose` | Docker `Reachable` probes both the engine and compose, and the derived compose default keeps the runtime wrapper (`sudo -n docker compose`, not a bare `docker compose`) |
| `TestCtrTransportHonoursRuntime` | The node-local transport does the same for `exec`, the stdin `Upload`, and `cp` |
| `TestCtrRuntimeDefaultArgvUnchanged` | A single-token runtime produces exactly the argv it did before, for both docker and podman |
| `TestCtrExecutorRefusesUnapprovedRuntime` | The container half of enforce-twice: a `Manager` and a transport built from a `*config.Config` that never saw `config.Load` still refuse an unapproved `sudo -n docker` on every path that could reach exec, handing the runner nothing |

### preflight_test.go

The read-only engine probe, and the child-environment hygiene it shares with
`internal/engine`.

| Test | What it covers |
| --- | --- |
| `TestPreflightRunsBeforeAnything` | The layer-7 ordering guarantee: `<runtime> info` is the FIRST call `Deploy`, `Delete` and `PrepHost` make, on both platforms -- anything before it would be host state left behind by a deploy that then failed on a stopped daemon |
| `TestPreflightFailureStopsTheDeploy` | An unreachable engine stops `Deploy` nonzero, carries the engine's own error, adds the actionable hint, writes no compose file, and issues no call after the probe |
| `TestPreflightHintIsPlatformShaped` | Docker gets the daemon/group hint; rootful podman gets `sudo systemctl start podman.socket`; rootless podman gets the user-session hint and explicitly NOT a sudo suggestion, which would start the engine its deploy is not using. None of them offers to act on the operator's behalf |
| `TestPreflightIsPreviewableUnderDryRun` | `--dry-run` echoes the probe and skips its assertion, so previewing needs no engine -- which is why there is no skip flag |
| `TestComposeSecretEnvNamesCannotBeSystemVars` | The config-side half of the child-environment rule: even with `container.name` set to `PATH`, `LD_PRELOAD`, `ld.preload` or `IFS`, every variable name keeps its fixed literal suffix, so no config value can produce a name the child's loader reads |
| `TestComposeSecretEnvIsTheOnlyChildEnvironment` | `composeSecretEnv` passes through exactly the secrets `render` declares and invents none, which is the assumption the test above rests on; values stay masked in any display path |

### manager_test.go

| Test | What it covers |
| --- | --- |
| `TestManagerCheckDryRun` | Preflight report for docker/podman x HA/standalone: title, mode line, runtime version probe, dry-run skip note |
| `TestManagerCheckDNSFailsLoudInHA` | An unresolvable redundancy host fails the check and is named |
| `TestManagerCheckStandaloneDNSWarnsOnly` | Standalone tolerates an unresolved name |
| `TestManagerPrepHostDryRunDoesNotWritePSK` | Dry-run leaves the env file untouched, never generates a PSK, and still echoes mkdir/chown |
| `TestManagerPrepHostWritesPSK` | The generated PSK is written into `nodes.psk`, the replication PSK is untouched, and the data dir is created and chowned |
| `TestManagerPrepHostRootlessUsesUnshareChown` | Rootless podman chowns via `podman unshare` |
| `TestPrepHostRootlessNoFileSufficient` | Rootless prep probes this user's hard `nofile` limit with `sh -c 'ulimit -Hn'` and reports the value when it covers `container.ulimits.nofile` |
| `TestPrepHostRootlessNoFileTooLow` | The point of the check: a rootless container cannot raise `nofile` past the user's hard limit, so prep stops rather than deploying a broker that would run under-provisioned. The message carries both numbers and the exact `limits.d` drop-in, including the re-login that re-reads it |
| `TestPrepHostRootlessNoFileUnlimited` | An unlimited hard limit satisfies any configured value |
| `TestPrepHostRootlessNoFileUnreadable` | A limit that will not parse fails loud rather than being assumed adequate |
| `TestPrepHostRootlessNoFileUnsetSkips` | With no configured `nofile` there is nothing to assert against, so the probe never runs -- the hand-built config the executors are handed |
| `TestPrepHostRootfulSkipsNoFile` | Docker and rootful podman never probe: their privileged engine raises the limit itself, so the invoking user's hard limit does not bound the container |
| `TestPrepHostRootlessNoFileDryRun` | `--dry-run` echoes the probe and skips the assertion, the same shape `Preflight` uses, since the Echo runner answers nothing |
| `TestSplitLimit` | The `soft:hard` ulimit parser: a pair, a single value meaning both, surrounding whitespace, and the values that mean "nothing to assert" (`-1`, empty, non-numeric) |
| `TestManagerDeployDockerComposeWritesFile` | Deploy writes the compose file and runs `compose up -d --force-recreate` |
| `TestManagerDockerComposeCommandOverride` | A `docker.compose` override (the standalone `docker-compose` binary) is what every compose call goes through |
| `TestManagerDockerCheckProbesCompose` | Docker `check` probes the compose command, so a missing plugin fails at check time rather than at deploy time |
| `TestManagerDockerCheckFailsWhenComposeMissing` | With only the compose probe failing, the error names the `docker.compose` override |
| `TestManagerDeployDockerPassesSecretsAsEnv` | Deploy writes no secret file at all: `compose up -d` goes through `RunEnv` carrying `SOLACE_ADMIN_PASSWORD`/`SOLACE_REDUNDANCY_PSK`, no value reaches an argv, and the compose file holds the source/target/variable references and the `*filepath` pointer instead of either value |
| `TestManagerDeployPodmanCreatesSecrets` | Deploy loads both secrets into podman's store with `secret create --replace` under container-scoped names (`sol-pod-*`), values on stdin and never in an argv; the quadlet unit is 0600 |
| `TestManagerDeployRejectsEmptySecret` | An empty required secret fails the deploy naming the field and the fix (`prep host` for the PSK) rather than starting a broker without one |
| `TestManagerDeployDockerDryRunMasksSecretEnv` | Dry-run creates nothing, stays previewable before `prep host` has generated the PSK, and echoes the compose environment as `NAME=***` without the password |
| `TestManagerDeployPodmanDryRunHidesSecretBytes` | The dry-run echo shows the secret-create command and a stdin byte count, never the values |
| `TestManagerRedeployUnchangedRestartsForRotation` | Docker: `--restart` against an unchanged compose file forces `up -d --force-recreate`, which is the only way a rotated secret reaches the running broker |
| `TestManagerRedeployPodmanUnchangedRestartsForRotation` | Podman: the same state restarts the service (the store was refreshed, but the running container holds the old values) |
| `TestContainerRunningMatchesNameExactly` | The branch selector matches the container name exactly, so a sibling deployment on the same host (`solace-edge` next to `solace`) is never mistaken for this one: exact, among-others, sibling-only, prefix-only, empty and whitespace-padded listings, plus a failed probe reading as not-running |
| `TestManagerRedeployStoppedContainerRecreates` | The arm with no consent prompt: an unchanged compose file with the container stopped recreates rather than starts, because a start would replay the credentials the container was created with |
| `TestManagerRedeployUnchangedHintsRotation` | Without `--restart` nothing is recreated and the log names `--restart` as the way to apply a rotation |
| `TestManagerDeployPodmanWritesUnit` | The quadlet unit is written, then daemon-reload and service start |
| `TestManagerDeployPodmanDryRunSkipsWrite` | Dry-run echoes the systemctl steps without writing the unit |
| `TestManagerPodmanEUIDGuardSkippedOnDryRun` | The rootless/rootful euid guard does not run under dry-run |
| `TestManagerDeletePodmanRemovesUnit` | Delete stops the service, removes the unit, and daemon-reloads |
| `TestManagerDeletePodmanPurgeRootless` | Rootless purge removes the data dir via `podman unshare` |
| `TestManagerDeleteDockerComposeDownWhenFileExists` | With a compose file present, delete runs `compose down` |
| `TestManagerDeleteDockerPurgeRemovesDataDir` | Delete runs `compose down` and, with purge, removes the data dir |
| `TestManagerDeleteDockerComposeNoFileFallsBackToStopRm` | A missing compose file falls back to stop+rm |
| `TestManagerRedeployUnchangedIsNoOp` | Re-deploying an unchanged artifact against a running broker touches nothing on either platform and says there was nothing to do |
| `TestManagerRedeployChangedNeedsConsent` | A changed artifact against a running broker is written but not applied without consent (warning that the broker is still on the previous one); `--restart` and an accepted prompt both apply it. This is the podman silent no-op fixed: `systemctl start` on an active unit left the old image running while reporting success |
| `TestManagerDescribe` | `<runtime> inspect` on both platforms, plus the installed unit on podman, with a missing unit tolerated |
| `TestManagerCopy` | The copy verbs: cp out of and into the container, per-file reporting, an error with no files, and a non-zero exit when any file fails |
| `TestManagerPrepHostRegistryLogin` | prep logs in to the registry with the password on stdin, never in an argv -- credentials that were previously ignored on containers entirely |
| `TestManagerPrepHostNoLoginWithoutCreds` | No credentials means no login attempt |
| `TestManagerPrepHostRejectsHalfCredentials` | A user with no password (or the reverse) fails loud rather than attempting a broken login |
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
| `TestManagerDeployPodmanSecretCreateError` | A failed `secret create` aborts the deploy, naming the config key behind the secret |
| `TestManagerDeletePodmanStopTolerated` | A failed stop is tolerated with a warning |
| `TestManagerDeletePodmanDaemonReloadError` | A daemon-reload failure during delete propagates |
| `TestManagerDeletePodmanRemoveUnitError` | An unremovable unit path fails delete |
| `TestManagerDeleteDockerComposeDownError` | A `compose down` failure propagates |
| `TestManagerDeleteDockerStopTolerated` | A failed stop in the no-compose-file fallback is tolerated with a warning |
| `TestManagerDeletePurgeError` | A failing data-dir removal under `--purge` propagates |
| `TestManagerStatusDockerNoComposeFile` | With no compose file on disk, status lists the container and never calls compose |
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

The command runner seam: `Echo` (dry-run) and `Exec` (real subprocess), plus display
quoting, PATH resolution, and the pre-exec announcement. 25 tests across 2 files.

### runner_test.go

| Test | What it covers |
| --- | --- |
| `TestHelperProcess` | Not a test -- the os/exec helper-process shim used as a fake external command by the `Exec` tests below |
| `TestQuoteTok` | Display quoting per token: empty, plain, and every shell-significant character |
| `TestQuote` | Whole command lines, including quoted and empty arguments |
| `TestEchoRun` | `Run` echoes `+ <cmd>` |
| `TestEchoRunInteractive` | `RunInteractive` echoes with quoting applied |
| `TestEchoRunInput` | `RunInput` shows stdin as a byte count, never its contents |
| `TestEchoRunEnv` | `RunEnv` echoes the command first and annotates the variables it would set after it (`<<< (env: NAME=***)`), so `+ <cmd>` stays greppable and no value is printed |
| `TestEchoRunEnvNoEnv` | With nothing to annotate the line is exactly what `Run` prints, not a dangling `(env: )` |
| `TestMaskEnv` | The masking helper keeps names (quoting an odd one) and drops values, including a value holding `=` and an empty one |
| `TestEchoOutput` | `Output` echoes and returns nil bytes |
| `TestEchoDefaultWriter` | A zero-value `Echo` writes to stdout |
| `TestExecOutput` | `Output` captures a child process's stdout |
| `TestExecOutputFail` | A non-zero exit errors and the message names the binary |
| `TestExecRun` | `Run` streams on success and errors (naming the binary) on failure |
| `TestExecRunInput` | `RunInput` feeds stdin to the child and streams its output |
| `TestExecRunEnv` | `RunEnv` gives the child the extra variable *and* still inherits this process's environment; a non-zero exit errors naming the binary |
| `TestExecRunInteractive` | `RunInteractive` runs a child to a clean exit |
| `TestExecOutputInput` | OutputInput wires stdin from `in` into the child and captures stdout into the returned buffer rather than leaking it to the real terminal -- the curl -K - path this backs has no other way to get the response body back |
| `TestExecOutputInputFail` | OutputInput wraps a child failure the same way Output does ("name: err") |
| `TestEchoOutputInput` | the dry-run echo for the credential-bearing curl -K - path prints a byte count and never the stdin body, mirroring TestEchoRunInput but with an explicit assertion that the fake credential never appears in the echoed line |

### resolve_test.go

PATH resolution and the command announcement -- the transparency half of the
execution guard, which lives here because this is where a binary is actually run.
The announcement is injected (`Exec.Announce`) rather than written to a package
variable, so what a run prints is decided by the CLI, in one place.

| Test | What it covers |
| --- | --- |
| `TestExecVerboseAnnouncesEveryCommand` | Under `--verbose`, `Exec` writes `==> exec: <absolute resolved path> <args>` before each command -- the path, not the name as typed, with the arguments alongside -- and on *every* call, since a trail with one entry per binary would not answer "what did this run issue?" |
| `TestExecIsSilentWithoutVerbose` | The default runner announces nothing, and neither does a bare `Exec{}`: nil `Announce` is the quiet default rather than a hole that falls back to stderr. The CLI names the binaries once in its preamble instead, which is what stopped the same resolved path landing between report lines on every command |
| `TestExecEchoesOnEveryMethod` | All six `Runner` methods announce, not just `Run` -- and `Output`/`OutputInput`, which read cluster state, are the least visible to begin with |
| `TestResolveMissingBinaryIsActionable` | A name that resolves nowhere fails before any process starts, naming what was not found rather than a path the operator never typed. Asserted on `Resolve` (which the CLI preamble now shares) and again through `Exec.Run` |
| `TestResolveRefusesCurrentDirectory` | The pair to config's bare-name rule: a bare name must never resolve to a file in the working directory -- the binary unpacked beside a shared env file. Go reports it as `exec.ErrDot`; hosts that do not offer the cwd copy at all are logged and still asserted not to run it |
| `TestChildEnvNamesAreNotSystemVariables` | The variable names this tool passes to a child are never `PATH`, `LD_PRELOAD` or their relatives, and their values stay masked in display paths. The upstream half is container's `TestComposeSecretEnvNamesCannotBeSystemVars` |

---

## internal/render

Manifest and unit-file rendering, guarded by committed goldens. 17 tests.

### render_test.go

| Test | What it covers |
| --- | --- |
| `TestGolden` | Fourteen renderings from the sample env match their goldens: k8s broker CR (the sample omits `k8s.ports`, `timezone` and both security blocks, so this covers the default ports and the omitted branches), the same CR with an explicit port list (a container port differing from the service port, and an explicit protocol), the same CR with timezone and both security blocks set, podman quadlet, docker compose in HA and standalone (standalone drops the redundancy block and its PSK secret reference), container env pairs for HA (no `timezone`, so no TZ pair) and standalone (`timezone` set, so the TZ pair is present), the container env file, the podman and docker secret scripts (the docker case uses a password holding a quote, a space and a `$`), the quadlet and compose forms of the opt-in health check, the CR with an explicit pullPolicy plus podAnnotations/podLabels, the CR with node and pod affinity alongside the legacy anti-affinity term, and the CR with loadBalancer annotations, node labels and tolerations (values carrying a colon and a URL, which survive only because both halves are quoted) |
| `TestArtifactsCarryNoSecrets` | The externalization guard: with distinctive values in `admin.pass`, `nodes.psk` and an additional user's password, no deployment artifact on any platform (broker CR, quadlet, compose file, env file) contains any of them, while `SecretScript` -- the renderer that supplies them -- contains all three |
| `TestContainerSecretsRedundancy` | HA lists both secrets in a fixed order with the expected broker settings, `FilePathKey`/`MountPath` derive the file form both engines use (the mount is named after the setting, not the host-side secret), and standalone lists the admin password only (no mate link, so no PSK secret). An encrypted server-certificate key adds a third secret reaching the broker as `tls_servercertificate_passphrasefilepath`, and only when the passphrase is actually set |
| `TestContainerSecretNamesAreHostScoped` | The de-confliction: the host-side name is `<container.name>-<suffix>` (the default name keeps the historical `solace-admin-password`), the in-container target and path never carry that prefix, and `EnvVar` maps `.`/`-` to `_` and prefixes a leading digit so the name stays exportable |
| `TestAdditionalUsersReachBothHalves` | An extra user's password becomes a per-host secret named after it (`ConfigKey` naming the env-file key), while its access level and `*filepath` pointer ride the env pairs and the password does not |
| `TestQuadletHealthCmdEscapesPercent` | systemd expands %-specifiers in every unit assignment, not just the quoted `Environment=` ones, so a percent-encoded character in a probe URL is doubled or the line is dropped and the health check silently disabled. Quotes and backslashes stay untouched there (the value is unquoted and podman splits it itself), and compose keeps the percent literal since it has no specifier expansion |
| `TestHealthCmdDefaultsToReadiness` | An enabled block with no cmd polls `/health-check/readiness` on 5550, and an explicit cmd wins |
| `TestSecretPreflight` | The precondition `deploy` and `--gen-secrets-only` share: an empty secret value is refused up front, naming the field and `prep host` for the PSK. Standalone needs no PSK, so an empty one is fine there |
| `TestShQuote` | Secret-script quoting: a value holding a single quote survives as itself instead of ending the shell string |
| `TestParsePort` | Port entries across the `name=container`, `container:service`, and `/PROTO` forms |
| `TestParseToleration` | Toleration Equal (`key=value:effect`) and Exists (`key:effect`) forms |
| `TestQuadletEscape` | systemd `Environment=` escaping of `%`, `"`, and `\` |
| `TestScalingReachesContainersAsEnv` | Every scaling knob reaches docker and podman as a container environment variable -- five of them used to render on k8s only -- carrying the env file's values, including an explicit `0`, which is a real setting rather than an absent one |
| `TestScalingReachesK8sAsSpecOnly` | The other half of the delivery split: on k8s the same settings are CR fields under `spec.systemScaling` and never pod environment variables, the spool size is spelled `maxSpoolUsage` there, and the container spelling appears nowhere in the CR |
| `TestScalingTierReachesEveryArtifact` | One tier value decides the CPU cap in all three artifacts: the broker CR's `messagingNodeCpu`/`messagingNodeMemory`, compose's `cpus:`/`mem_limit:`, and the quadlet's `PodmanArgs=--cpus=`/`Memory=`. It uses 100000, which is no platform's default, so the value is proven read rather than hardcoded -- the goldens only ever show the default tier |
| `TestContainerMemOverrideReachesArtifact` | The asymmetry survives to the artifact: an overridden `container.mem` reaches compose while the CPU stays the tier's |
| `TestUnresolvedTierOmitsLimits` | The renderers' fail-safe branch. A `Config` built in code -- what the executors are handed -- carries no tier, and all three artifacts must then omit the limits rather than emit an empty `cpus:`/`--cpus=`/`messagingNodeCpu:`, which the engines and the CRD would reject |

---

## internal/tools/vulnjudge

The dev-only judge the `scan` task pipes govulncheck JSON through. 11 tests.

### main_test.go

| Test | What it covers |
| --- | --- |
| `TestJudge` | The core policy: a called vulnerability with a fix exits 1, one with no released fix warns and exits 0, and uncalled findings are ignored |
| `TestJudgeModuleFixHint` | A fixable module vulnerability advises `go get <module>@<version>` rather than a toolchain bump |
| `TestJudgeNoFindings` | A clean scan exits 0 with the no-vulnerabilities message |
| `TestJudgeTolerantOfBOM` | A UTF-8 BOM (what PowerShell's encoders emit) does not break decoding |
| `TestJudgeMalformedInput` | Truncated or non-JSON input exits 2 with a malformed-JSON message rather than reporting success |
| `TestJudgeEmptyTrace` | a finding with zero trace frames is folded into the 'uncalled' bucket instead of panicking on Trace[0] a few lines later |
| `TestJudgeSortsFixableByID` | report ordering across two fixable vulns given out of ID order is deterministic (map iteration order can't leak through) and also reaches plural()'s multi-count branch ("2 vulnerabilities") |
| `TestJudgeUnknownModule` | a called finding whose trace carries a function but no module still prints an actionable placeholder instead of a blank field |
| `TestRunUsageError` | run prints the usage message to errOut (never out) and returns code 2 for any argument count other than exactly one |
| `TestRunUnreadableFile` | an unreadable path returns code 2 and names the file in the errOut message rather than handing a zero-value byte slice to judge |
| `TestRunHappyPath` | run wires judge's report to out and judge's exit code to its return value -- the run/judge integration, not just judge in isolation |

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
  `nodes:` block) in `internal/cli/cli_test.go`. `writeRuntimeEnv`
  (`internal/cli/allowcommand_test.go`) is the same idea parameterized by `k8s.runtime`, for
  driving one hostile or wrapped command through the whole CLI.
- **`guardConfig`** (`internal/config/execguard_test.go`) is a config that validates cleanly
  on every platform with only the command fields left to vary, so a `Validate` failure in the
  execution-guard tests can only have come from the guard.
- **`wrappedCfg`** (`internal/k8s/runtime_test.go`) and **`wrappedCtrCfg`**
  (`internal/container/runtime_test.go`) carry a chained runner (`microk8s kubectl`,
  `sudo -n docker`) WITH the operator approval the guard requires; `unapprovedCfg` /
  `unapprovedCtrCfg` are the same values without it, for the refusal tests.
- **`internal/convert/testdata/legacy-k8s.env`** is the legacy-format fixture, with
  `legacy-k8s.yaml.golden` as its expected output. It deliberately does not point at
  `bash/env/sample`: the whole `bash/` tree is gitignored, so that path is absent on a fresh
  checkout and CI could never run the test. The container flavour uses an inline fixture
  (`ctrEnv`) in the same file, and `k8sEnv` is an inline minimal-but-complete k8s source
  for tests that need a conversion free of "incomplete" warnings.

### Per-package doubles

| Package | Double | Purpose |
| --- | --- | --- |
| internal/broker | `fakeTransport` (broker_test.go) | Records every upload/run/output and answers via a `responder` func |
| internal/broker | `newTestOps` (broker_test.go) | Builds an `Ops` with a buffer sink and zero poll interval so nothing sleeps |
| internal/broker | `runErrTransport` (coverage_test.go) | Embeds `fakeTransport` but fails `Run`, reaching the best-effort cleanup branches |
| internal/broker | `uploadErrTransport`, `downloadErrTransport`, `runErrMatchTransport` (coverage_test.go) | The same embed-and-override shape for the transport methods `fakeTransport` always succeeds at: failing `Upload`/`UploadFile`, failing `Download` (optionally only for one remote path, so a bundle fetch can fail while the archive succeeds), and failing `Run` only when an argv predicate matches (isolating a best-effort cleanup failure from an earlier fail-loud one). Each still records the call |
| internal/broker | `removed` (broker_test.go) | Whether an uploaded script was deleted afterwards -- `removeCLI` issues `rm -f` through `Run`, so removals land in `runs`, not `outputs`. Matters for any script whose body carries a secret |
| internal/broker | `seqTransport`, `newLocalOps`, `rd` (verify_local_test.go) | Scripts a sequence of `show redundancy` readings for the local HA state machines |
| internal/k8s | `recRunner` / `rrCall` (transport_test.go) | Capturing `engine.Runner` with `outQueue` and `runErrQueue` for scripting multi-step ops, plus `outErrQueue` (per-`Output` errors, so one read in an op can fail while an earlier one succeeds) and `runInputErr` (fails `apply -f -` / `delete -f -`). Both queues fall back to the blanket `outErr`/`runErr` once drained, so a test that sets only those behaves as before. `canI`/`canIErr` answer `Cluster.Preflight`'s `auth can-i` probe out of band (default: permitted) so it never consumes a queued read written for a different call, and `afterPreflight` asserts the probe came first and returns the calls after it |
| internal/k8s | `haCfg`, `saCfg` (names_test.go), `adminCfg` (prep_test.go), `loadK8s` (secrets_test.go) | Config builders |
| internal/k8s | `checkGolden`, `-update` flag (secrets_test.go) | Golden comparison for the whole package |
| internal/cli | `renderCommandDocs`, `-update` flag (commanddoc_test.go) | Renders `docs/commands.md` from the cobra tree; the doc is the golden |
| internal/container | `capRunner` / `failOn` (transport_test.go) | Capturing runner whose `fail` (Run family) and `outFail` (Output family) hooks error on a targeted command, driving each error-wrap branch -- `failOn("info")` is how a failed engine preflight is injected. `outFail` exists because the blanket `outErr` cannot single out one of two probes in the same call. `capCall.env` records the extra environment of a `RunEnv` call, which is how the docker secret path is asserted |
| internal/container | `containsStr`, `maskedKeys` (manager_test.go) | Exact-match lookup in a captured environment, and `engine.MaskEnv` for failure messages -- a test diagnostic must not print a secret either |
| internal/container | `newEchoMgr`, `newCapMgr`, `ctrCfg` (manager_test.go) | Manager over dry-run Echo, or over the capturing runner for real file writes. `Manager.Confirm` is the injectable restart prompt (nil declines, which is what a non-interactive run must do) and `Manager.Restart` is the `--restart` pre-approval |
| internal/container | `assertMode` (manager_test.go) | Permission-bit assertion for the artifacts the manager writes; skipped on Windows, which carries no POSIX mode |
| internal/cli | `runRoot`, `capture`/`captureStdout`/`captureStderr` (cli_test.go) | Builds a fresh command tree per call and captures a standard stream through a pipe |
| internal/cli | `runRootWith` (cli_test.go) | `runRoot` with a hook to configure the `App` before `Execute` -- how the confirm-prompt branches are driven deterministically instead of depending on the test process's own stdin. `runRoot` delegates to it with a nil hook, so it is unchanged for every existing test |
| internal/cli | `opRunner` / `opCall` / `opFailOn` / `opFailOnCount` (cli_test.go) | A fake `engine.Runner` whose failure is targeted by argv substring (or by the Nth matching occurrence, for the repeated identical `apply`/`delete` calls in `up`/`down`/`prep`). Ported from internal/container's `capRunner`/`failOn`; this is what makes the orchestration-abort tests possible -- assert step N fails and step N+1 never ran. `Output` answers the k8s `auth can-i` preflight "yes" unless a test supplies its own `output`/`fail` for that call, so the op-level tests stay about the work they were written for |
| internal/cli | `loadDirect`, `healthyShowRD` (cli_test.go) | Loads a config from an inline YAML body for tests that build an `App` directly with a non-Echo runner, and a canned `show redundancy` transcript that satisfies `broker.primaryRedundancyUp` so a poll succeeds on the first read |
| internal/cli | `bashEnv`, `writeBashEnv` (cli_test.go) | Minimal legacy env file for the convert command tests |
| internal/convert | `strictDecode` (convert_test.go) | Re-reads generated YAML with `KnownFields(true)`, so an emitted key that is not in the schema fails the test |
| internal/convert | `ctrEnv`, `convertOK`, `hasWarning` (convert_test.go) | Container-flavoured legacy fixture and warning assertions |
| internal/config | `envTree`, `writeTempYAML`, `minimalK8s` (config_test.go) | Real temp-dir fixture trees for path resolution and loading, plus the smallest valid k8s document the secret-reference tests append a body to |
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
| `App.Interactive` | `isTTY(os.Stdin)` | internal/cli -- whether a run may prompt. Gates `confirmDelete`/`confirmPurge`/`confirmRestart` and the placement-labelling step of `up`, so the prompt branches guarding destructive actions are testable |
| `App.PromptIn` | `os.Stdin` | internal/cli -- where a confirmation answer is read from |
| `engine.Runner` | `engine.Exec` | Everywhere -- swapped for `engine.Echo` (dry-run) or a capturing fake |

Filesystem access is *not* seamed: tests use real `t.TempDir()` trees.
