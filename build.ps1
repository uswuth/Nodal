<#
.SYNOPSIS
    Production build for Nodal: version metadata, executable and Windows installer.

.DESCRIPTION
    1. Generates cmd\nodal\resource.syso from cmd\nodal\versioninfo.json so nodal.exe
       carries the publisher name, product description and copyright in its file properties.
    2. Builds bin\nodal.exe as a GUI-subsystem binary (no console window).
    3. Compiles installer\setup.iss into dist\ with Inno Setup 6.3+ and writes SHA256SUMS.txt.

.PARAMETER Version
    Product version in the form major.minor.patch (default 1.0.0).

.PARAMETER SkipInstaller
    Build only the executable, skip the installer.

.PARAMETER SkipChecksums
    Do not write dist\SHA256SUMS.txt.

.EXAMPLE
    .\build.ps1

.EXAMPLE
    .\build.ps1 -Version 1.1.0
#>
[CmdletBinding()]
param(
    [string]$Version = "1.0.0",
    [switch]$SkipInstaller,
    [switch]$SkipChecksums
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
Set-Location $root

$versionFull = if ($Version -match '^\d+\.\d+\.\d+\.\d+$') { $Version } else { "$Version.0" }
$binDir = Join-Path $root "bin"
$distDir = Join-Path $root "dist"
$exePath = Join-Path $binDir "nodal.exe"
$issPath = Join-Path $root "installer\setup.iss"

Write-Host "Nodal $Version production build" -ForegroundColor Cyan

if (-not (Get-Command goversioninfo -ErrorAction SilentlyContinue)) {
    Write-Host "Installing goversioninfo build tool (one time)..."
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
    $goBin = Join-Path (go env GOPATH) "bin"
    if (Test-Path $goBin) { $env:PATH = "$goBin;$env:PATH" }
}

Write-Host "Generating publisher metadata (cmd\nodal\resource.syso)..."
go generate ./cmd/nodal

Write-Host "Building $exePath ..."
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
go build -trimpath -ldflags "-H windowsgui -s -w" -o $exePath ./cmd/nodal

if ($SkipInstaller) {
    Write-Host "Installer skipped." -ForegroundColor Yellow
} else {
    $isccCandidates = @()
    if (${env:ProgramFiles(x86)}) { $isccCandidates += (Join-Path ${env:ProgramFiles(x86)} "Inno Setup 6\ISCC.exe") }
    if ($env:ProgramFiles) { $isccCandidates += (Join-Path $env:ProgramFiles "Inno Setup 6\ISCC.exe") }
    $iscc = $isccCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1

    if (-not $iscc) {
        throw "ISCC.exe not found. Install Inno Setup 6.3 or newer from https://jrsoftware.org/isdl.php"
    }

    Write-Host "Compiling installer with $iscc ..."
    New-Item -ItemType Directory -Force -Path $distDir | Out-Null
    & $iscc "/DMyAppVersion=$Version" "/DMyAppVersionFull=$versionFull" $issPath
    if ($LASTEXITCODE -ne 0) {
        throw "Inno Setup compilation failed with exit code $LASTEXITCODE"
    }
}

if (-not $SkipChecksums -and (Test-Path $distDir)) {
    $hashes = @()
    foreach ($file in @($exePath) + (Get-ChildItem $distDir -File | Where-Object { $_.Name -ne "SHA256SUMS.txt" }).FullName) {
        if (Test-Path $file) {
            $hash = (Get-FileHash -Algorithm SHA256 $file).Hash.ToLower()
            $hashes += "$hash  $(Split-Path $file -Leaf)"
        }
    }
    if ($hashes.Count -gt 0) {
        $hashes | Set-Content (Join-Path $distDir "SHA256SUMS.txt") -Encoding ascii
        Write-Host "Checksums written to dist\SHA256SUMS.txt"
    }
}

Write-Host "Build complete." -ForegroundColor Green
Get-ChildItem $binDir, $distDir -File -ErrorAction SilentlyContinue | Select-Object Name, Length
