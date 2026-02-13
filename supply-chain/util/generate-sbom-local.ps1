<#
.SYNOPSIS
    Local SBOM Generation Script for Windows

.DESCRIPTION
    This script mimics the GitHub Actions workflow for local testing on Windows.

.PARAMETER ProjectFilter
    Filter by owner/repo (e.g., "kubernetes/kubernetes"). Leave empty for all.

.PARAMETER Force
    Force regenerate existing SBOMs

.PARAMETER MaxReleases
    Maximum releases to process per repo (default: 3)

.EXAMPLE
    .\generate-sbom-local.ps1
    Process all projects

.EXAMPLE
    .\generate-sbom-local.ps1 -ProjectFilter "kubernetes/kubernetes"
    Process specific repo

.EXAMPLE
    .\generate-sbom-local.ps1 -Force -ProjectFilter "coredns/coredns"
    Force regenerate for coredns

.NOTES
    Prerequisites:
    - Go 1.22+
    - git
    - gh CLI (GitHub CLI) - for API access
    - yq (https://github.com/mikefarah/yq) - install via: choco install yq

    Environment variables:
    - GH_TOKEN or GITHUB_TOKEN - GitHub token for API access
#>

param(
    [Parameter(Position = 0)]
    [string]$ProjectFilter = "",

    [switch]$Force,

    [int]$MaxReleases = 3
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
$DataFile = Join-Path $RootDir "supply-chain\util\data\repositories.yaml"
$SbomBaseDir = Join-Path $RootDir "supply-chain\sbom"

function Write-Header($text) {
    Write-Host ""
    Write-Host ("=" * 50) -ForegroundColor Cyan
    Write-Host $text -ForegroundColor Cyan
    Write-Host ("=" * 50) -ForegroundColor Cyan
}

function Test-Prerequisites {
    $missing = @()

    if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
        $missing += "go"
    }

    if (-not (Get-Command "git" -ErrorAction SilentlyContinue)) {
        $missing += "git"
    }

    if (-not (Get-Command "gh" -ErrorAction SilentlyContinue)) {
        $missing += "gh (GitHub CLI)"
    }

    if (-not (Get-Command "yq" -ErrorAction SilentlyContinue)) {
        $missing += "yq"
    }

    if ($missing.Count -gt 0) {
        Write-Host "Error: Missing required tools: $($missing -join ', ')" -ForegroundColor Red
        Write-Host ""
        Write-Host "Installation:"
        Write-Host "  go:  https://golang.org/dl/"
        Write-Host "  gh:  https://cli.github.com/ or: winget install GitHub.cli"
        Write-Host "  yq:  choco install yq or: winget install MikeFarah.yq"
        exit 1
    }
}

function Install-Bom {
    $bomPath = & go env GOPATH
    $bomExe = Join-Path $bomPath "bin\bom.exe"

    if (-not (Test-Path $bomExe)) {
        Write-Host "Installing bom tool..."
        & go install sigs.k8s.io/bom/cmd/bom@latest
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Error: Failed to install bom tool" -ForegroundColor Red
            exit 1
        }
    }

    # Add to PATH if not already there
    $goBin = Join-Path $bomPath "bin"
    if ($env:PATH -notlike "*$goBin*") {
        $env:PATH = "$goBin;$env:PATH"
    }

    Write-Host "Using bom: $bomExe"
}

function Get-SanitizedProjectName($name) {
    return ($name.ToLower() -replace ' ', '-' -replace '[^a-z0-9-]', '')
}

function New-Sbom($Owner, $Repo, $ProjectName, $Tag) {
    $sanitizedProject = Get-SanitizedProjectName $ProjectName
    $version = $Tag -replace '^v', ''
    $sbomDir = Join-Path $SbomBaseDir "$sanitizedProject\$Repo\$version"
    $sbomFile = Join-Path $sbomDir "$Repo.json"

    # Check if SBOM already exists
    if ((Test-Path $sbomFile) -and -not $Force) {
        Write-Host "  SBOM already exists: $sbomFile, skipping..." -ForegroundColor Yellow
        return $false
    }

    Write-Host "  Generating SBOM for $Owner/$Repo@$Tag..."

    # Clone the repository at specific tag
    $tempDir = Join-Path $env:TEMP "sbom-$(Get-Random)"

    try {
        $cloneOutput = & git clone --depth 1 --branch $Tag "https://github.com/$Owner/$Repo.git" $tempDir 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  Failed to clone $Owner/$Repo@$Tag, skipping..." -ForegroundColor Yellow
            return $false
        }

        # Create output directory
        if (-not (Test-Path $sbomDir)) {
            New-Item -ItemType Directory -Path $sbomDir -Force | Out-Null
        }

        # Generate SBOM with bom tool
        $bomOutput = & bom generate --format json --output $sbomFile $tempDir 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Successfully generated SBOM: $sbomFile" -ForegroundColor Green
            return $true
        } else {
            Write-Host "  Failed to generate SBOM for $Owner/$Repo@$Tag" -ForegroundColor Red
            Write-Host "  Error: $bomOutput" -ForegroundColor Red
            return $false
        }
    }
    finally {
        if (Test-Path $tempDir) {
            Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
        }
    }
}

function Test-PreRelease($tag) {
    return $tag -match '[-\.](alpha|beta|rc|pre|dev|snapshot|nightly|canary|test|draft|wip)\d*'
}

function Test-SemVer($tag) {
    return $tag -match '^v?\d+\.\d+'
}

function Process-Repository($Owner, $Repo, $ProjectName) {
    $processed = 0

    Write-Header "Processing: $ProjectName ($Owner/$Repo)"

    # Get releases from GitHub API
    try {
        $releasesJson = & gh api "repos/$Owner/$Repo/releases" --paginate 2>&1
        $releases = $releasesJson | ConvertFrom-Json -ErrorAction SilentlyContinue
    }
    catch {
        $releases = @()
    }

    if ($releases.Count -eq 0) {
        Write-Host "No releases found, trying tags..."

        try {
            $tagsJson = & gh api "repos/$Owner/$Repo/tags" --paginate 2>&1
            $tags = ($tagsJson | ConvertFrom-Json).name | Select-Object -First 20
        }
        catch {
            $tags = @()
        }

        if ($tags.Count -eq 0) {
            Write-Host "No tags found, skipping..."
            return
        }

        foreach ($tag in $tags) {
            if (Test-PreRelease $tag) {
                Write-Host "  Skipping pre-release tag: $tag" -ForegroundColor Gray
                continue
            }

            if (-not (Test-SemVer $tag)) {
                Write-Host "  Skipping non-semver tag: $tag" -ForegroundColor Gray
                continue
            }

            if (New-Sbom $Owner $Repo $ProjectName $tag) {
                $processed++
            }

            if ($processed -ge $MaxReleases) {
                Write-Host "  Processed $MaxReleases releases, stopping..."
                break
            }
        }
    }
    else {
        # Filter stable releases
        $stableReleases = $releases | Where-Object {
            -not $_.draft -and -not $_.prerelease
        } | Select-Object -First 50

        foreach ($release in $stableReleases) {
            $tag = $release.tag_name

            if (Test-PreRelease $tag) {
                Write-Host "  Skipping pre-release tag: $tag" -ForegroundColor Gray
                continue
            }

            if (New-Sbom $Owner $Repo $ProjectName $tag) {
                $processed++
            }

            if ($processed -ge $MaxReleases) {
                Write-Host "  Processed $MaxReleases releases, stopping..."
                break
            }
        }
    }

    Write-Host "Processed $processed releases for $Owner/$Repo"
}

function New-SbomIndex {
    Write-Header "Generating SBOM index"

    $indexFile = Join-Path $SbomBaseDir "index.json"

    # Find all SBOM JSON files
    $sbomFiles = Get-ChildItem -Path $SbomBaseDir -Filter "*.json" -Recurse |
        Where-Object { $_.Name -ne "index.json" }

    if ($sbomFiles.Count -eq 0) {
        Write-Host "No SBOMs found, creating empty index..."
        @{
            generated_at = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
            sboms = @()
        } | ConvertTo-Json | Set-Content $indexFile
        return
    }

    $sboms = @()
    foreach ($file in $sbomFiles) {
        $relPath = $file.FullName.Replace("$SbomBaseDir\", "").Replace("\", "/")
        $parts = $relPath.Split("/")

        if ($parts.Count -ge 3) {
            $sboms += @{
                project = $parts[0]
                repo = $parts[1]
                version = $parts[2]
                path = $relPath
            }
        }
    }

    @{
        generated_at = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        sboms = $sboms
    } | ConvertTo-Json -Depth 10 | Set-Content $indexFile

    Write-Host "Index generated: $indexFile"
    Write-Host "Total SBOMs: $($sbomFiles.Count)"
}

# Main execution
function Main {
    Write-Host "SBOM Generator for CNCF Projects" -ForegroundColor Cyan
    Write-Host "=================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Settings:"
    Write-Host "  Force regenerate: $Force"
    Write-Host "  Project filter: $(if ($ProjectFilter) { $ProjectFilter } else { 'all' })"
    Write-Host "  Max releases per repo: $MaxReleases"
    Write-Host "  Output directory: $SbomBaseDir"
    Write-Host ""

    Test-Prerequisites
    Install-Bom

    # Ensure data file exists
    if (-not (Test-Path $DataFile)) {
        Write-Host "Error: Repository data file not found: $DataFile" -ForegroundColor Red
        exit 1
    }

    # Get repositories to process
    if ($ProjectFilter) {
        $filterParts = $ProjectFilter.Split("/")
        $owner = $filterParts[0]
        $repo = $filterParts[1]
        $reposJson = & yq -o=json ".repositories | map(select(.owner == `"$owner`" and .repo == `"$repo`"))" $DataFile
    }
    else {
        $reposJson = & yq -o=json '.repositories' $DataFile
    }

    $repos = $reposJson | ConvertFrom-Json

    if ($repos.Count -eq 0) {
        Write-Host "No repositories found matching filter: $ProjectFilter" -ForegroundColor Red
        exit 1
    }

    Write-Host "Found $($repos.Count) repositories to process"

    foreach ($repoInfo in $repos) {
        Process-Repository $repoInfo.owner $repoInfo.repo $repoInfo.name
    }

    New-SbomIndex

    Write-Header "SBOM generation complete!"
}

Main
