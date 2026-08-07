<#
scripts/dev.ps1 - build/vet/test/cov/scan/dist tasks for the `solace` Go CLI.

Mirror of scripts/dev.sh (behaviourally identical: same task names, same
gating, same footer format). The USER runs this; CI calls task names only.
Works from any cwd. Accepts multiple tasks per invocation.

  ./scripts/dev.ps1 build vet test
  ./scripts/dev.ps1 all          # build vet test           (CI runs: all scan)
  ./scripts/dev.ps1 full         # all + cov scan graphify  (pre-tag sweep)

Windows PowerShell 5.1 compatible: ASCII only, no &&, no ternary, and no
Tee-Object on native commands.
#>
#requires -Version 5.1
[CmdletBinding()]
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Tasks)

$ErrorActionPreference = 'Continue'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot  = Split-Path -Parent $ScriptDir
$LogDir    = Join-Path $ScriptDir 'logs'
$DistDir   = Join-Path $RepoRoot  'dist'
$CovDir    = Join-Path $RepoRoot  'coverage'
$BinName   = 'solace'

# govulncheck is a go.mod `tool` dependency, so its version is pinned by go.sum
# rather than by a variable here. Bump it with:
#   go get -tool golang.org/x/vuln/cmd/govulncheck@vX.Y.Z

# Local convenience cross-compile set (`dist`). CI does not use it: tag.yml
# matrixes over `build` with TARGET_OS/TARGET_ARCH from BUILD_TARGETS instead.
$DistTargets = @(
  @{ os = 'linux';   arch = 'amd64' },
  @{ os = 'linux';   arch = 'arm64' },
  @{ os = 'darwin';  arch = 'arm64' },
  @{ os = 'windows'; arch = 'amd64' }
)

# -race needs cgo + a C compiler; OFF by default on Windows, enable with SOLACE_RACE=1.
$RaceFlag = @(); $CoverMode = 'count'
if ($env:SOLACE_RACE -eq '1') { $RaceFlag = @('-race'); $CoverMode = 'atomic' }

# Toolchain parity: go.mod's `toolchain` pin is what local and CI must agree on,
# but an exported GOTOOLCHAIN (`local` especially) silently overrides it and
# builds against whatever Go is on PATH. Apply the pin only when unset, so an
# explicit value still wins -- same when-unset rule as the env defaults.
if (-not $env:GOTOOLCHAIN) {
  # @() forces an array: -match on a single hit returns a bare string, and
  # indexing that would yield its first character instead of the line.
  $tcLines = @((Get-Content (Join-Path $RepoRoot 'go.mod')) -match '^toolchain\s+\S+')
  if ($tcLines.Count -gt 0) { $env:GOTOOLCHAIN = ($tcLines[0] -split '\s+')[1] }
}

# Keep captured logs clean: no color / progress from tools.
$env:NO_COLOR = '1'

# --- pretty helpers -----------------------------------------------------------
function Step($m) { Write-Host "==> $m"    -ForegroundColor Cyan }
function Ok($m)   { Write-Host "[ ok ] $m" -ForegroundColor Green }
function Warn($m) { Write-Host "[warn] $m" -ForegroundColor Yellow }
function Die($m)  { Write-Host "[fail] $m" -ForegroundColor Red; exit 1 }

# Matches dev.sh's `date +%Y-%m-%dT%H:%M:%S%z`: .NET 'zzz' renders the offset as
# +08:00, so strip the colon or the two scripts' footers would not match.
function Get-Now {
  $d = Get-Date
  return $d.ToString('yyyy-MM-ddTHH:mm:ss') + $d.ToString('zzz').Replace(':', '')
}
function Get-Log($task) { Join-Path $LogDir "$task.log" }

$script:LogFile = $null
function Start-TaskLog($task) {
  if (-not (Test-Path $LogDir)) { New-Item -ItemType Directory -Path $LogDir | Out-Null }
  $script:LogFile = Get-Log $task
  "=== $(Get-Now) | $task ===" | Set-Content -Path $script:LogFile -Encoding utf8
}

# Contract footer, appended to the task log and echoed. Add-Content, never
# Tee-Object: Tee doubles lines and writes UTF-16 into a UTF-8 file.
function Write-Finish {
  param([string]$Task, [int]$Code, [int]$Seconds)
  $status = 'OK'
  if ($Code -ne 0) { $status = "FAILED (exit $Code)" }
  $line = '{0} | {1} | {2}s | {3}' -f (Get-Now), $Task, $Seconds, $status
  Add-Content -Path (Get-Log $Task) -Value $line -Encoding utf8
  Write-Host $line
}

# Cap runs a native command, capturing combined output once (never pipe a native
# command through Tee-Object on 5.1). Appends an ANSI-stripped copy to the log,
# echoes to the console, and returns the command's exit code.
#
# No param block on purpose: every token (incl. -count=1, -o, ./...) lands in the
# automatic $args verbatim instead of being parsed as a parameter. A local
# Continue preference keeps native stderr from raising a terminating error before
# the real exit code is read.
function Cap {
  $ErrorActionPreference = 'Continue'
  $exe  = $args[0]
  $rest = @(); if ($args.Count -gt 1) { $rest = $args[1..($args.Count - 1)] }
  $out  = (& $exe @rest 2>&1 | ForEach-Object { "$_" } | Out-String -Width 4096)
  $code = $LASTEXITCODE
  $clean = $out -replace "\x1b\[[0-9;?]*[a-zA-Z]", ""
  Add-Content -Path $script:LogFile -Value $clean -Encoding utf8
  if ($out.Trim().Length -gt 0) { Write-Host $out.TrimEnd() }
  return $code
}

# --- tasks --------------------------------------------------------------------
# Tasks return non-zero on failure; the dispatcher writes the footer and stops.
# PowerShell returns EVERY uncaptured pipeline value, so route native commands
# through Cap and pipe anything not being returned to Out-Null.

function Task-tidy { return (Cap go mod tidy) }
function Task-vet  { return (Cap go vet ./...) }
function Task-test { return (Cap go test @RaceFlag -count=1 ./...) }

# Build-One compiles the CLI for one target into dist/. The target lands in the
# binary name because the release job merges every leg with merge-multiple:
# identical names would silently overwrite.
function Build-One($os, $arch) {
  $out = Join-Path $DistDir "$BinName-$os-$arch"
  if ($os -eq 'windows') { $out = "$out.exe" }
  if (-not (Test-Path $DistDir)) { New-Item -ItemType Directory -Path $DistDir | Out-Null }
  Step "  $os/$arch"
  $oldGoos = $env:GOOS; $oldGoarch = $env:GOARCH; $oldCgo = $env:CGO_ENABLED
  $env:CGO_ENABLED = '0'; $env:GOOS = $os; $env:GOARCH = $arch
  try {
    return (Cap go build -trimpath -ldflags '-s -w' -o $out .)
  } finally {
    $env:GOOS = $oldGoos; $env:GOARCH = $oldGoarch; $env:CGO_ENABLED = $oldCgo
  }
}

# CI sets TARGET_OS/TARGET_ARCH (from the BUILD_TARGETS repo variable); unset
# means build for the host, so local and CI output are the same shape.
function Task-build {
  $os = $env:TARGET_OS;     if (-not $os)   { $os = (& go env GOOS) }
  $arch = $env:TARGET_ARCH; if (-not $arch) { $arch = (& go env GOARCH) }
  return (Build-One $os $arch)
}

function Task-dist {
  foreach ($t in $DistTargets) {
    $c = Build-One $t.os $t.arch
    if ($c -ne 0) { return $c }
  }
  Ok "binaries in $DistDir"
  return 0
}

function Task-cov {
  if (-not (Test-Path $CovDir)) { New-Item -ItemType Directory -Path $CovDir | Out-Null }
  $prof = Join-Path $CovDir 'coverage.out'
  $html = Join-Path $CovDir 'coverage.html'
  # -count=1 forces a real run so a cached test result can't report a stale
  # coverage total and mask a drop below the floor (the previous total in
  # logs/cov.log; local only -- CI is a fresh checkout with no prior log).
  $c = Cap go test @RaceFlag "-covermode=$CoverMode" "-coverprofile=$prof" -count=1 ./...
  if ($c -ne 0) { return $c }
  $c = Cap go tool cover "-html=$prof" "-o" $html
  if ($c -ne 0) { return $c }
  $total = (& go tool cover "-func=$prof" | Select-Object -Last 1)
  Add-Content -Path $script:LogFile -Value $total -Encoding utf8
  Write-Host $total
  Ok "coverage -> $html"
  return 0
}

# One task, every applicable check: govulncheck over source + deps. FATAL on a
# fixable finding, standalone or inside an aggregate; local and CI behave the
# same. (No image half: this project ships binaries only.)
#
# `go tool`, not `go run pkg@version`: the latter resolves in an empty synthetic
# module, so the scanner is compiled by whatever Go is on PATH -- and a checker
# built by an older toolchain refuses to type-check a module declaring a newer
# `go` line ("package requires newer Go version"). A tool dependency builds as
# part of this module, so it gets the same toolchain build/vet/test already use.
#
# `-format json` + vulnjudge, not text mode: text mode exits non-zero for ANY
# finding, so a CVE with no released fix would block a release on someone else's
# patch schedule. The JSON stream carries fixed_version per finding, so the judge
# can fail on what is actionable and warn on what is not -- see
# internal/tools/vulnjudge.
function Task-scan {
  $raw = Join-Path $LogDir 'scan.json'
  # 5.1 decodes native output with the console codepage, which would corrupt a
  # non-ASCII byte in a vulnerability summary and leave the judge's json.Decoder
  # to choke on it. Read as UTF-8, then put the console back as it was.
  $prevEnc = [Console]::OutputEncoding
  try {
    [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
    # `-format json` always exits 0, even with findings, so a non-zero code here
    # is a real tool failure (bad flags, packages that will not load) and stays
    # fatal. stderr is left on the console so those errors reach the CI step log.
    $json = (& go tool govulncheck -format json ./...) -join "`n"
    $code = $LASTEXITCODE
  } finally {
    [Console]::OutputEncoding = $prevEnc
  }
  if ($code -ne 0) {
    Warn 'govulncheck failed to run; see the output above'
    return $code
  }
  # WriteAllText with a BOM-less encoder: 5.1's utf8 writers prepend a BOM and
  # json.Decoder rejects it.
  [System.IO.File]::WriteAllText($raw, $json, (New-Object System.Text.UTF8Encoding($false)))
  return (Cap go run ./internal/tools/vulnjudge $raw)
}

# Local only: the graph is a developer artifact, not a CI output.
function Task-graphify {
  if ($env:CI) { Warn 'graphify is local-only; skipping in CI'; return 0 }
  if (-not (Get-Command graphify -ErrorAction SilentlyContinue)) {
    Warn 'graphify not on PATH; skipping'; return 0
  }
  return (Cap graphify update .)
}

# --- dispatch -----------------------------------------------------------------
$All  = @('build', 'vet', 'test')
$Full = @('build', 'vet', 'test', 'cov', 'scan', 'graphify')

function Show-Usage {
  $raceDesc = ($RaceFlag -join ' ')
  $targetsDesc = (($DistTargets | ForEach-Object { "$($_.os)/$($_.arch)" }) -join ', ')
  Write-Host @"
dev.ps1 - build/test/scan tooling for the solace CLI

Usage: dev.ps1 <task> [task...]

Tasks:
  tidy     go mod tidy
  vet      go vet ./...
  build    compile -> dist\$BinName-<os>-<arch>[.exe]; TARGET_OS/TARGET_ARCH
           pick the target, unset means host
  test     go test $raceDesc -count=1 ./...
  cov      coverage profile -> coverage/coverage.html + printed total
  scan     govulncheck (fatal on a fixable vulnerability this module calls;
           one with no released fix warns and passes)
  dist     cross-compile $targetsDesc
  graphify refresh graphify-out/ (local only; skipped when CI is set)
  all      $($All -join ' ')   (what CI runs, as: all scan)
  full     $($Full -join ' ')   (pre-tag sweep)

Env: SOLACE_RACE=1 enables -race; TARGET_OS/TARGET_ARCH cross-compile a single ``build``.
     GOTOOLCHAIN defaults to go.mod's ``toolchain`` pin; export it to override.
     govulncheck's version lives in go.mod (tool directive), not an env var.
Logs: $LogDir\<task>.log (each run closes with a timestamped footer)
"@
}

Set-Location $RepoRoot
if (-not $Tasks -or $Tasks.Count -eq 0) { Show-Usage; exit 1 }
if ($Tasks[0] -in @('-h', '--help', 'help')) { Show-Usage; exit 0 }

$queue = @()
foreach ($t in $Tasks) {
  switch ($t) {
    'all'      { $queue += $All }
    'full'     { $queue += $Full }
    'binaries' { $queue += 'dist' }
    default    { $queue += $t }
  }
}

foreach ($task in $queue) {
  if (-not (Get-Command "Task-$task" -ErrorAction SilentlyContinue)) {
    Die "unknown task: $task (try: dev.ps1 help)"
  }
  Step $task
  Start-TaskLog $task
  $sw = [Diagnostics.Stopwatch]::StartNew()
  $code = 0
  try { $code = & "Task-$task" } catch { Write-Host $_; $code = 1 }
  # A stray pipeline value inside a task would make this an array; take the
  # command's own code, which is always the last value returned.
  if ($code -is [array]) { $code = $code[-1] }
  if ($null -eq $code) { $code = 0 }
  $sw.Stop()
  Write-Finish -Task $task -Code ([int]$code) -Seconds ([int]$sw.Elapsed.TotalSeconds)
  if ([int]$code -ne 0) { Warn "$task failed; stopping"; exit 1 }
  Ok $task
}
Ok "done: $($queue -join ' ')"
