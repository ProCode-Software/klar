#Requires -Version 5.1
param(
    [Alias('h')]
    [switch]$Help,
    [switch]$Global,
    [switch]$AddToPath,
    [switch]$Prebuild
)

$ErrorActionPreference = 'Stop'

function Write-Red($msg) { Write-Host $msg -ForegroundColor Red }
function Write-Yellow($msg) { Write-Host $msg -ForegroundColor Yellow }
function Write-Status($msg) { Write-Host $msg -ForegroundColor Magenta }

if ($Help) {
    @'
Flags:
    -Global       Install Klar globally (requires administrator)
    -AddToPath    Add Klar to PATH
    -Prebuild     Install a prebuilt binary instead of building from source
    -Help         Show this help message

https://github.com/ProCode-Software/klar
'@ | Write-Host
    return
}

if ($AddToPath -and $Global) {
    Write-Red "Can't enable '-AddToPath' with '-Global'"
    return
}

$globalExplicit = $PSBoundParameters.ContainsKey('Global')
$prebuildExplicit = $PSBoundParameters.ContainsKey('Prebuild')
$addToPathExplicit = $PSBoundParameters.ContainsKey('AddToPath')

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)

function Select-Option {
    param(
        [string]$Prompt,
        [string[]]$Options
    )
    if ($Prompt) { Write-Yellow $Prompt }
    for ($i = 0; $i -lt $Options.Count; $i++) {
        Write-Host "  $($i + 1)) $($Options[$i])"
    }
    while ($true) {
        $choice = Read-Host '#?'
        if ($choice -match '^\d+$' -and [int]$choice -ge 1 -and [int]$choice -le $Options.Count) {
            return $Options[[int]$choice - 1]
        }
        Write-Host 'Invalid option, try again.'
    }
}

if (-not $globalExplicit) {
    $location = Select-Option -Prompt 'Where do you want to install Klar?' `
        -Options @('Local (current user)', 'Global (all users)')
    if ($location -eq 'Local (current user)') {
        $Global = $false
        if (-not $addToPathExplicit) {
            $reply = Read-Host 'Do you want to add Klar to PATH? (Y/n)'
            if ($reply.Trim().ToLower() -ne 'n') { $AddToPath = $true }
        }
    }
    else { $Global = $true }
}

if ($Global -and -not $isAdmin) {
    Write-Red "Global installation requires an elevated (Administrator) PowerShell session.`nRerun this script as Administrator, or drop '-Global' to install for the current user only."
    return
}

if (-not $prebuildExplicit) {
    Write-Yellow 'Do you want to build from source, or use a prebuilt binary?'
    Write-Host 'Building from source makes the latest features and fixes available, but requires Git and the Go toolchain to be installed, and may take longer to install. Downloading a prebuilt binary is faster, but may not include the latest Klar fixes.'
    $buildType = Select-Option -Options @('Build from source', 'Download a prebuilt binary')
    $Prebuild = ($buildType -eq 'Download a prebuilt binary')
}

Write-Host ''

$klarExec = 'klar.exe'
$glasExec = 'glas.exe'

function Get-PrebuildArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($arch -eq 'x86' -and $env:PROCESSOR_ARCHITEW6432) {
        $arch = $env:PROCESSOR_ARCHITEW6432
    }
    switch ($arch) {
        'AMD64' { return 'x86_64' }
        'ARM64' { return 'arm64' }
        default { return $null }
    }
}

function Invoke-DownloadPrebuild {
    Write-Status '📦 Downloading prebuilt Klar and Glas binaries...'

    $archName = Get-PrebuildArch
    if (-not $archName) {
        Write-Red "Unfortunately, we don't provide prebuilds for the '$($env:PROCESSOR_ARCHITECTURE)' architecture :(`nPlease build from source instead by rerunning without '-Prebuild'."
        throw 'KlarInstallAborted'
    }

    $releases = Invoke-RestMethod -Uri 'https://api.github.com/repos/ProCode-Software/klar/releases?per_page=1' `
        -Headers @{ 'User-Agent' = 'klar-installer' }
    if (-not $releases -or $releases.Count -eq 0) {
        Write-Red "Unfortunately, we couldn't find any Klar releases on GitHub.`nPlease build from source instead by rerunning without '-Prebuild'."
        throw 'KlarInstallAborted'
    }
    $release = $releases[0]
    $tagName = $release.tag_name

    $klarBundle = $release.assets | Where-Object { $_.name -match "^klar-.*windows-$archName" } | Select-Object -First 1
    if (-not $klarBundle) {
        Write-Red "Unfortunately, we couldn't find a prebuilt binary for windows-$archName in release $tagName.`nPlease build from source instead by rerunning without '-Prebuild'."
        throw 'KlarInstallAborted'
    }
    # Download zip with Klar and Glas binaries
    $binariesZip = Join-Path $buildDir 'binaries.zip'
    Invoke-WebRequest -Uri $klarBundle.browser_download_url -OutFile $binariesZip -UseBasicParsing
    Expand-Archive -Path $binariesZip -DestinationPath $buildDir -Force

    # Download the standard library
    Write-Status '📚 Downloading the standard library...'
    $stdlibZip = Join-Path $buildDir 'stdlib.zip'
    Invoke-WebRequest -Uri "https://github.com/ProCode-Software/klar/releases/download/$tagName/stdlib.zip" `
        -OutFile $stdlibZip -UseBasicParsing
    Expand-Archive -Path $stdlibZip -DestinationPath $buildDir -Force
}

function Invoke-BuildFromSource {
    # Ensure we have Git and Go installed
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Red 'git is required to install Klar. Install it at https://git-scm.com.'
        throw 'KlarInstallAborted'
    }
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Red 'Go is required to install Klar. Install it at https://go.dev.'
        throw 'KlarInstallAborted'
    }

    # Clone Klar repository
    Write-Status '📖 Cloning Klar repository...'
    $cloneOutput = & git clone https://github.com/ProCode-Software/klar.git . 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Red 'Failed to clone Klar repository. The error was:'
        $cloneOutput | ForEach-Object { Write-Host $_ }
        throw 'KlarInstallAborted'
    }

    # Get the current GOOS and GOARCH
    $env:GOOS = 'windows'
    switch ($env:PROCESSOR_ARCHITECTURE) {
        'AMD64' { $env:GOARCH = 'amd64' }
        'ARM64' { $env:GOARCH = 'arm64' }
        'x86' { $env:GOARCH = '386' }
        default {
            Write-Red "Unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)"
            throw 'KlarInstallAborted'
        }
    }

    # Build Klar and Glas executables
    $env:GOEXPERIMENT = 'jsonv2'
    $version = '0.0.1'
    $commit = (& git rev-parse --short HEAD).Trim()
    $ldflags = "-X 'github.com/ProCode-Software/klar/internal/cli.KlarVersion=$version' -X 'github.com/ProCode-Software/klar/internal/cli.KlarCommit=$commit'"

    Write-Status '🏗️ Building Klar and Glas binaries...'
    & go generate ./...
    if ($LASTEXITCODE -ne 0) { Write-Red 'go generate failed.'; throw 'KlarInstallAborted' }
    & go build -ldflags $ldflags -o $klarExec ./cmd/klar
    if ($LASTEXITCODE -ne 0) { Write-Red 'Failed to build klar.'; throw 'KlarInstallAborted' }
    & go build -ldflags $ldflags -o $glasExec ./cmd/glas
    if ($LASTEXITCODE -ne 0) { Write-Red 'Failed to build glas.'; throw 'KlarInstallAborted' }
}

$buildDir = Join-Path $env:TEMP ('klar-install-' + [guid]::NewGuid().ToString('N'))
$pushedLocation = $false
try {
    New-Item -ItemType Directory -Path $buildDir -Force | Out-Null
    Push-Location $buildDir
    $pushedLocation = $true

    if ($Prebuild) {
        Invoke-DownloadPrebuild
    }
    else {
        Invoke-BuildFromSource
    }

    # Install Klar and Glas to bin directory
    Write-Status '🚚 Installing Klar and Glas...'
    if ($Global) {
        $binDir = Join-Path $env:ProgramFiles 'Klar\bin'
    }
    else {
        $binDir = Join-Path $env:LocalAppData 'Klar\bin'
    }
    New-Item -ItemType Directory -Path $binDir -Force | Out-Null
    Copy-Item -Path (Join-Path $buildDir $klarExec), (Join-Path $buildDir $glasExec) -Destination $binDir -Force

    # Only add to PATH if it's not already there
    if ($Global) {
        $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
        if ($machinePath -notlike "*$binDir*") {
            [Environment]::SetEnvironmentVariable('Path', "$machinePath;$binDir", 'Machine')
        }
    }
    elseif ($AddToPath) {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        if ([string]::IsNullOrEmpty($userPath)) {
            [Environment]::SetEnvironmentVariable('Path', $binDir, 'User')
        }
        elseif ($userPath -notlike "*$binDir*") {
            [Environment]::SetEnvironmentVariable('Path', "$userPath;$binDir", 'User')
        }
    }
    if ($env:Path -notlike "*$binDir*") {
        $env:Path = "$env:Path;$binDir"
    }

    # Copy the standard library
    Write-Status '📚 Installing the standard library...'
    # Keep paths in sync with ./internal/module/system_path.go (KlarStdDir)
    if ($Global) {
        $stdDir = Join-Path $env:ProgramData 'Klar\std'
    }
    else {
        $stdDir = Join-Path $env:LocalAppData 'Klar\std'
    }
    New-Item -ItemType Directory -Path $stdDir -Force | Out-Null
    Copy-Item -Path (Join-Path $buildDir 'std\*') -Destination $stdDir -Recurse -Force

    Write-Host ''
    Write-Host '🐨 Klar has been successfully installed!' -ForegroundColor Green
    Write-Host "To get started, run 'klar --help'. To use Glas, run 'glas --help'."
    Write-Host ''
    Write-Host 'GitHub: https://github.com/ProCode-Software/klar' -ForegroundColor Blue
}
catch {
    if ($_.Exception.Message -ne 'KlarInstallAborted') {
        Write-Red $_.Exception.Message
    }
}
finally {
    if ($pushedLocation) {
        Pop-Location
    }
    Remove-Item -Path $buildDir -Recurse -Force -ErrorAction SilentlyContinue
}
