# aria2t installer (Windows): downloads the latest release binary and installs
# it to %LOCALAPPDATA%\Programs\aria2t, adding it to the user PATH.
#
# Pass a Chrome extension id to ALSO register the browser extension's native
# messaging host — the same aria2t.exe serves both the TUI and the extension,
# so there is no separate helper to install:
#   $env:ARIA2T_EXT_ID='<extension-id>'; iwr -useb https://aria2t.c0nn3ct.info/windows.ps1 | iex
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
if (-not (Get-Command aria2c -ErrorAction SilentlyContinue)) {
    Write-Host 'note: aria2c not found - the built-in daemon needs aria2 (winget install aria2.aria2)'
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
