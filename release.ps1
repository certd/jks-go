<#
.SYNOPSIS
    jks-go Windows release script.
.DESCRIPTION
    Runs tests, generates changelog, creates a git tag, and pushes to remote.
.PARAMETER Version
    Version number to release, e.g. "1.0.0". The "v" prefix will be added automatically.
.PARAMETER SkipTest
    Skip running tests.
.PARAMETER DryRun
    Preview actions without actually pushing.
.EXAMPLE
    .\release.ps1 1.0.0
.EXAMPLE
    .\release.ps1 1.0.0 -DryRun
.EXAMPLE
    .\release.ps1 1.0.0 -SkipTest
#>

param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$Version,

    [switch]$SkipTest,

    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Push-Location $ScriptDir

try {
    $TagName = "v$Version"

    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host "  jks-go Release: $TagName"              -ForegroundColor Cyan
    Write-Host "========================================`n" -ForegroundColor Cyan

    # ── 1. Check prerequisites ──────────────────────────────────────────
    Write-Host "[1/6] Checking prerequisites..." -ForegroundColor Yellow

    $goVersion = go version 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "Go is not installed or not in PATH"
    }
    Write-Host "  Go: $goVersion" -ForegroundColor Green

    $gitVersion = git --version 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "Git is not installed or not in PATH"
    }
    Write-Host "  Git: $gitVersion" -ForegroundColor Green

    $branch = git rev-parse --abbrev-ref HEAD 2>&1
    Write-Host "  Branch: $branch" -ForegroundColor Green

    $status = git status --porcelain 2>&1
    if ($status) {
        throw "Working directory is not clean. Please commit or stash changes before releasing."
    }
    Write-Host "  Working directory: clean" -ForegroundColor Green

    # ── 2. Run tests ────────────────────────────────────────────────────
    if (-not $SkipTest) {
        Write-Host "`n[2/6] Running tests..." -ForegroundColor Yellow
        go test -v ./src/...
        if ($LASTEXITCODE -ne 0) {
            throw "Tests failed. Aborting release."
        }
        Write-Host "  All tests passed" -ForegroundColor Green
    }
    else {
        Write-Host "`n[2/6] Tests skipped (--SkipTest)" -ForegroundColor Yellow
    }

    # ── 3. Build with version ───────────────────────────────────────────
    Write-Host "`n[3/6] Building with version info..." -ForegroundColor Yellow
    $commit = git rev-parse --short HEAD
    $buildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd_HH:mm:ss")

    $ldflags = "-s -w -X main.Version=$Version -X main.Commit=$commit -X main.BuildDate=$buildDate"
    go build -ldflags="$ldflags" -o "dist/jks-go.exe" ./src/
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed."
    }

    $output = & .\dist\jks-go.exe -version 2>&1
    Write-Host "  $output" -ForegroundColor Green

    # ── 4. Generate changelog ───────────────────────────────────────────
    Write-Host "`n[4/6] Generating changelog..." -ForegroundColor Yellow

    $prevTag = try { git describe --tags --abbrev=0 2>&1 } catch { "" }
    if ($LASTEXITCODE -eq 0 -and $prevTag) {
        $prevTag = $prevTag.Trim()
        $changelog = git log --pretty=format:"- %s (%h)" "$prevTag..HEAD"
        Write-Host "  Changes since $prevTag :" -ForegroundColor Cyan
    }
    else {
        $changelog = git log --pretty=format:"- %s (%h)"
        Write-Host "  All commits (first release):" -ForegroundColor Cyan
    }

    if ($changelog) {
        $changelog -split "`n" | ForEach-Object { Write-Host "    $_" }
    }
    else {
        Write-Host "    (no new commits)" -ForegroundColor Gray
    }

    # ── 5. Create tag ───────────────────────────────────────────────────
    Write-Host "`n[5/6] Creating tag $TagName ..." -ForegroundColor Yellow

    $changelogFile = Join-Path $ScriptDir ".changelog.tmp"
    if ($changelog) {
        $changelog | Out-File -FilePath $changelogFile -Encoding utf8
        $changelogMsg = Get-Content $changelogFile -Raw
        Remove-Item $changelogFile
    }
    else {
        $changelogMsg = "Release $TagName"
    }

    if ($DryRun) {
        Write-Host "  [DRY RUN] Would create annotated tag: $TagName" -ForegroundColor Magenta
        Write-Host "  [DRY RUN] Message: $changelogMsg" -ForegroundColor Magenta
        Write-Host "`n[6/6] Push skipped (--DryRun)" -ForegroundColor Yellow
        Write-Host "`nDry run completed. No changes were made." -ForegroundColor Cyan
        exit 0
    }

    git tag -a $TagName -m $changelogMsg
    Write-Host "  Tag $TagName created" -ForegroundColor Green

    # ── 6. Push ─────────────────────────────────────────────────────────
    Write-Host "`n[6/6] Pushing tag to remote..." -ForegroundColor Yellow
    git push origin $TagName
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to push tag. You may need to delete the local tag: git tag -d $TagName"
    }

    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host "  Release $TagName pushed successfully!"  -ForegroundColor Cyan
    Write-Host "  GitHub Actions will build and publish."  -ForegroundColor Cyan
    Write-Host "========================================`n" -ForegroundColor Cyan
}
catch {
    Write-Host "`nERROR: $_" -ForegroundColor Red
    exit 1
}
finally {
    Pop-Location
}
