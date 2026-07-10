# aria2t installer (Windows): downloads the latest release binary and installs
# it to %LOCALAPPDATA%\Programs\aria2t, adding it to the user PATH.
$ErrorActionPreference = 'Stop'

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
Copy-Item (Join-Path $tmp "aria2t-$tag-windows-$arch\aria2t.exe") (Join-Path $dest 'aria2t.exe') -Force
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
