# harness installer for Windows (PowerShell).
#
#   irm https://raw.githubusercontent.com/devstationtech/harness/main/install.ps1 | iex
#
# Environment overrides:
#   HARNESS_VERSION       release tag to install (default: latest), e.g. v0.1.0
#   HARNESS_INSTALL_DIR   install directory (default: %LOCALAPPDATA%\harness\bin)
#   GITHUB_TOKEN          token for a PRIVATE repository; not needed once public

$ErrorActionPreference = 'Stop'

$Repo    = 'devstationtech/harness'
$Binary  = 'harness'
$Version = if ($env:HARNESS_VERSION) { $env:HARNESS_VERSION } else { 'latest' }
$Dir     = if ($env:HARNESS_INSTALL_DIR) { $env:HARNESS_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'harness\bin' }

# --- detect architecture (must match .goreleaser.yaml archive naming) ---
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw "unsupported architecture '$($env:PROCESSOR_ARCHITECTURE)'" }
}
$asset = "${Binary}_windows_${arch}.zip"

$tmp = Join-Path $env:TEMP ("harness-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
try {
    $zip = Join-Path $tmp $asset

    if ($env:GITHUB_TOKEN) {
        # Private repo: resolve the asset through the API and download with auth.
        $rel = if ($Version -eq 'latest') { 'latest' } else { "tags/$Version" }
        $headers = @{ Authorization = "Bearer $($env:GITHUB_TOKEN)"; Accept = 'application/vnd.github+json' }
        Write-Host "Resolving $Repo release ($Version) via API ..."
        $release = Invoke-RestMethod -Headers $headers -Uri "https://api.github.com/repos/$Repo/releases/$rel"
        $a = $release.assets | Where-Object { $_.name -eq $asset } | Select-Object -First 1
        if (-not $a) { throw "asset '$asset' not found in the release" }
        Write-Host "Downloading $asset ..."
        Invoke-WebRequest -Headers @{ Authorization = "Bearer $($env:GITHUB_TOKEN)"; Accept = 'application/octet-stream' } `
            -Uri $a.url -OutFile $zip
    }
    else {
        # Public repo: download release assets directly.
        $base = if ($Version -eq 'latest') {
            "https://github.com/$Repo/releases/latest/download"
        } else {
            "https://github.com/$Repo/releases/download/$Version"
        }
        Write-Host "Downloading $asset ($Version) ..."
        Invoke-WebRequest -Uri "$base/$asset" -OutFile $zip

        # Best-effort checksum verification.
        try {
            $sumsFile = Join-Path $tmp 'checksums.txt'
            Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumsFile
            $want = (Select-String -Path $sumsFile -Pattern ([regex]::Escape($asset)) | Select-Object -First 1).Line.Split(' ')[0]
            $got  = (Get-FileHash -Algorithm SHA256 -Path $zip).Hash.ToLower()
            if ($want -and ($want.ToLower() -ne $got)) { throw "checksum mismatch for $asset" }
            elseif ($want) { Write-Host 'Checksum OK.' }
        } catch [System.Net.WebException] { } # no checksums published — skip
    }

    Expand-Archive -Path $zip -DestinationPath $tmp -Force
    $exe = Join-Path $tmp "$Binary.exe"
    if (-not (Test-Path $exe)) { throw "archive did not contain '$Binary.exe'" }

    New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    Copy-Item -Path $exe -Destination (Join-Path $Dir "$Binary.exe") -Force

    # Add the install dir to the user PATH if missing.
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (($userPath -split ';') -notcontains $Dir) {
        [Environment]::SetEnvironmentVariable('Path', "$userPath;$Dir", 'User')
        Write-Host "Added $Dir to your user PATH — restart your shell to use '$Binary'."
    }

    Write-Host "Installed: $(& (Join-Path $Dir "$Binary.exe") version)"
    Write-Host "Run '$Binary help' to get started."
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
