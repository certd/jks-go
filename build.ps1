<#
.SYNOPSIS
    jks-go build script for Windows.
.DESCRIPTION
    Compiles jks-go.exe with version info from git.
.EXAMPLE
    .\build.ps1
.EXAMPLE
    .\build.ps1 -Version "1.0.0"
#>

param(
    [string]$Version = "",

    [switch]$Release
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Push-Location $ScriptDir

try {
    $commit = try { git rev-parse --short HEAD 2>&1 } catch { "unknown" }
    if ($LASTEXITCODE -ne 0) { $commit = "unknown" }

    $buildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd_HH:mm:ss")

    if (-not $Version) {
        $tag = try { git describe --tags --abbrev=0 2>&1 } catch { "" }
        if ($LASTEXITCODE -eq 0 -and $tag) {
            $Version = $tag.Trim() -replace '^v', ''
        }
        else {
            $Version = "dev"
        }
    }

    $ldflags = "-X main.Version=$Version -X main.Commit=$commit -X main.BuildDate=$buildDate"
    if ($Release) {
        $ldflags = "-s -w $ldflags"
    }

    $distDir = Join-Path $ScriptDir "dist"
    if (-not (Test-Path $distDir)) {
        New-Item -ItemType Directory -Path $distDir | Out-Null
    }
    $outputPath = Join-Path $distDir "jks-go.exe"

    Write-Host "Building jks-go.exe ..." -ForegroundColor Cyan
    Write-Host "  Version:   $Version"    -ForegroundColor Gray
    Write-Host "  Commit:    $commit"     -ForegroundColor Gray
    Write-Host "  BuildDate: $buildDate"  -ForegroundColor Gray

    go build -ldflags="$ldflags" -o $outputPath ./src/

    if ($LASTEXITCODE -ne 0) {
        throw "Build failed."
    }

    $fileInfo = Get-Item $outputPath
    $sizeKB = [math]::Round($fileInfo.Length / 1KB, 1)

    Write-Host ""
    Write-Host "Build successful!" -ForegroundColor Green
    Write-Host "  Output: $($fileInfo.FullName)" -ForegroundColor White
    Write-Host "  Size:   ${sizeKB} KB" -ForegroundColor White
}
catch {
    Write-Host "`nERROR: $_" -ForegroundColor Red
    exit 1
}
finally {
    Pop-Location
}
