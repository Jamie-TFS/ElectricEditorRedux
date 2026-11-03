# Builds every edition, signs them, and puts release files in .\release
#
#   .\build.ps1 1.0.0            (stable release)
#   .\build.ps1 1.1.0-beta.1     (pre-release: only the dev edition picks these up)
#
# One-time setup: go run ./tools/signer keygen, then paste the public key below.
param([Parameter(Mandatory = $true)][string]$Version)

$ErrorActionPreference = "Stop"
$PubKey = "852df7xXKKYL1epftM+yHaau56kqsKBc/gpq5N/zQfY="          
$Repo = "Jamie-TFS/ElectricEditorRedux"

if (-not $PubKey) { throw "Set `$PubKey in build.ps1 first (go run ./tools/signer keygen)." }
if (-not (Test-Path signing.key)) { throw "signing.key not found in this folder." }

$out = "release"
New-Item -ItemType Directory -Force $out | Out-Null

$editions = @{ "standard" = "ElectricEditor.exe"; "dev" = "ElectricEditor-dev.exe"; "amy" = "ElectricEditor-amy.exe" }
foreach ($ed in $editions.Keys) {
    $file = $editions[$ed]
    Write-Host "Building $ed -> $file"
    $ld = "-s -w -X main.version=$Version -X main.edition=$ed -X main.updateRepo=$Repo -X main.updatePubKey=$PubKey"
    wails build -clean -trimpath -ldflags $ld -o $file
    Copy-Item "build\bin\$file" "$out\$file" -Force
}

go run ./tools/signer sign (Get-ChildItem "$out\*.exe" | ForEach-Object { $_.FullName })

Write-Host ""
Write-Host "Release files are in .\$out. Create a GitHub release tagged v$Version and upload"
Write-Host "every .exe and .exe.sig. Each edition updates itself from its own file."
