# aria2t installer (Windows): downloads the latest release binary and installs
# it to %LOCALAPPDATA%\Programs\aria2t, adding it to the user PATH.
#
# Pass a Chrome extension id to ALSO register the browser extension's native
# messaging host — the same aria2t.exe serves both the TUI and the extension,
# so there is no separate helper to install:
#   $env:ARIA2T_EXT_ID='<extension-id>'; iwr -useb https://aria2t.c0nn3ct.info/windows.ps1 | iex
#
# When aria2c is missing the script offers to download aria2's own official
# Windows build; set $env:ARIA2T_INSTALL_ARIA2 to '1' (or '0') to answer that
# prompt unattended.
[CmdletBinding()]
param(
    [string]$ExtensionId = $env:ARIA2T_EXT_ID
)
$ErrorActionPreference = 'Stop'

if ($ExtensionId -and $ExtensionId -notmatch '^[a-p]{32}$') {
    Write-Error "invalid extension id: $ExtensionId (expected 32 chars a-p)"
    exit 1
}

$repo = 'c0nn3ct-info/aria2t'
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }

$rel = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$tag = $rel.tag_name
$pkg = "aria2t-$tag-windows-$arch.zip"
$url = "https://github.com/$repo/releases/download/$tag/$pkg"

$tmp = Join-Path $env:TEMP 'aria2t-install'
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
Write-Host "downloading $pkg"
Invoke-WebRequest -UseBasicParsing $url -OutFile (Join-Path $tmp $pkg)
Expand-Archive -Force (Join-Path $tmp $pkg) $tmp

$dest = Join-Path $env:LOCALAPPDATA 'Programs\aria2t'
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$binPath = Join-Path $dest 'aria2t.exe'
# Stop a host still running from a previous install so the (locked) .exe can be
# replaced; the browser respawns it on the next native message.
Get-CimInstance Win32_Process |
    Where-Object { $_.ExecutablePath -eq $binPath } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
Copy-Item (Join-Path $tmp "aria2t-$tag-windows-$arch\aria2t.exe") $binPath -Force
Remove-Item -Recurse -Force $tmp

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$dest*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$dest", 'User')
    Write-Host "added $dest to your user PATH (restart the terminal to pick it up)"
}

Write-Host "installed aria2t $tag to $dest"

# aria2 publishes an official static build for Windows: no installer, no
# runtime DLLs, a single aria2c.exe. Fetch it straight from that release
# instead of going through winget - no App Installer/Store dependency, and
# nothing is redistributed by us, so no GPL obligation attaches here.
#
# Pinned; bump the version and the hash together. Upstream has no win-arm64
# build, so an ARM64 host runs the x64 one under emulation.
$aria2Version = '1.37.0'
$aria2Dir     = "aria2-$aria2Version-win-64bit-build1"
$aria2Url     = "https://github.com/aria2/aria2/releases/download/release-$aria2Version/$aria2Dir.zip"
$aria2Sha256  = '67D015301EEF0B612191212D564C5BB0A14B5B9C4796B76454276A4D28D9B288'

# Installs aria2c.exe beside aria2t.exe, where the daemon's FindBinary looks
# for it before the shell (or the browser spawning the native host) has
# re-read the user PATH. $env:ARIA2T_INSTALL_ARIA2 = '1'/'0' answers the
# prompt for an unattended run.
function Install-Aria2 {
    param([Parameter(Mandatory)][string]$Dir)
    $note = "note: aria2c not found - the built-in daemon needs aria2 ($aria2Url)"

    $ans = $env:ARIA2T_INSTALL_ARIA2
    if (-not $ans) {
        # Read-Host talks to the host UI rather than stdin, so it still works
        # when the script arrives through `iwr | iex`. A host with no console
        # to ask on (a service, -NonInteractive) gets the note instead.
        if (-not [Environment]::UserInteractive) { Write-Host $note; return }
        try {
            $ans = Read-Host "aria2c not found; the built-in daemon needs it. Download aria2 $aria2Version from GitHub now? [Y/n]"
        } catch {
            Write-Host $note
            return
        }
    }
    if ($ans -match '^\s*(n|no|0|false)\s*$') { Write-Host $note; return }

    $tmp = Join-Path $env:TEMP 'aria2t-aria2'
    try {
        New-Item -ItemType Directory -Force -Path $tmp | Out-Null
        $zip = Join-Path $tmp 'aria2.zip'
        Write-Host "downloading $aria2Dir.zip"
        Invoke-WebRequest -UseBasicParsing $aria2Url -OutFile $zip
        $got = (Get-FileHash -Algorithm SHA256 -Path $zip).Hash
        if ($got -ne $aria2Sha256) {
            throw "checksum mismatch (expected $aria2Sha256, got $got)"
        }
        Expand-Archive -Force $zip $tmp
        $exe = Join-Path $Dir 'aria2c.exe'
        # A daemon left running by an earlier install holds the old file open.
        Get-CimInstance Win32_Process |
            Where-Object { $_.ExecutablePath -eq $exe } |
            ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
        Copy-Item (Join-Path $tmp "$aria2Dir\aria2c.exe") $exe -Force
        Write-Host "installed aria2c $aria2Version to $Dir"
    } catch {
        Write-Host "note: could not install aria2c - $($_.Exception.Message)"
        Write-Host $note
    } finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }
}

# Test-Path as well as Get-Command: a re-run in the same terminal has not
# picked up $dest on PATH yet, and re-downloading what is already there is
# pure waste.
if (-not (Get-Command aria2c -ErrorAction SilentlyContinue) -and
    -not (Test-Path (Join-Path $dest 'aria2c.exe'))) {
    Install-Aria2 -Dir $dest
}

# ---------------------------------------------------------------------------
# Optional: register the browser extension's native messaging host, pointing
# the manifest at the aria2t.exe just installed.
if (-not $ExtensionId) { return }

$manifestPath = Join-Path $dest 'com.aria2t.host.json'

# Merge ids into allowed_origins instead of overwriting: each browser/profile
# has its own extension id, so re-running from another browser must not evict
# the first. Union of (ids already in the file) + the passed id, deduped.
$origins = New-Object System.Collections.Generic.List[string]
if (Test-Path $manifestPath) {
    try {
        $prev = Get-Content -Raw -Path $manifestPath | ConvertFrom-Json
        foreach ($o in @($prev.allowed_origins)) { if ($o) { $origins.Add([string]$o) } }
    } catch { }
}
$origins.Add("chrome-extension://$ExtensionId/")
$uniqueOrigins = @($origins | Sort-Object -Unique)

# Hand-build the JSON so a single-element array still serializes as an array
# (ConvertTo-Json unwraps one-item arrays into a bare scalar). Each value goes
# through ConvertTo-Json individually for correct quoting/escaping.
$originsJson = ($uniqueOrigins | ForEach-Object { $_ | ConvertTo-Json }) -join ",`n    "
$pathJson = $binPath | ConvertTo-Json
$manifest = @"
{
  "name": "com.aria2t.host",
  "description": "aria2t native host",
  "path": $pathJson,
  "type": "stdio",
  "allowed_origins": [
    $originsJson
  ]
}
"@
[System.IO.File]::WriteAllText($manifestPath, $manifest)

$registryRoots = @(
    'Software\Google\Chrome\NativeMessagingHosts',
    'Software\Chromium\NativeMessagingHosts',
    'Software\BraveSoftware\Brave-Browser\NativeMessagingHosts',
    'Software\Microsoft\Edge\NativeMessagingHosts',
    'Software\Vivaldi\NativeMessagingHosts',
    'Software\Opera Software\Opera Stable\NativeMessagingHosts',
    'Software\Yandex\YandexBrowser\NativeMessagingHosts'
)

$written = 0
foreach ($root in $registryRoots) {
    $key = "$root\com.aria2t.host"
    try {
        # Registry.SetValue creates the key (and intermediates) when missing and
        # never deletes existing subkeys. New-Item -Force delete-recreates
        # instead, wiping sibling host registrations.
        [Microsoft.Win32.Registry]::SetValue("HKEY_CURRENT_USER\$key", '', $manifestPath)
        Write-Host "  registered HKCU\$key"
        $written++
    } catch {
        Write-Warning "  skipped HKCU\$key ($($_.Exception.Message))"
    }
}

if ($written -eq 0) {
    Write-Error 'could not register the extension host for any browser'
    exit 1
}
Write-Host "extension host registered for $written browser(s); reload aria2t on chrome://extensions"
