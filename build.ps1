# Builds and signs the release exe into .\release
#
#   .\build.ps1 1.0.0            (stable release)
#   .\build.ps1 1.1.0-beta.1     (pre-release: only dev-key users get these)
#
# One-time setup: go run ./tools/signer keygen, then paste the public key below.
param([Parameter(Mandatory = $true)][string]$Version)

$ErrorActionPreference = "Stop"
$PubKey = ""          # <- paste the public key from keygen here
$Repo = "Jamie-TFS/ElectricEditor"

foreach ($tool in @("go", "node", "npm", "wails")) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        $hint = @{
            "go"    = "Install Go from https://go.dev/dl/"
            "node"  = "Install Node LTS: winget install OpenJS.NodeJS.LTS"
            "npm"   = "npm comes with Node: winget install OpenJS.NodeJS.LTS"
            "wails" = "go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        }[$tool]
        throw "'$tool' not found. $hint - then close and reopen PowerShell."
    }
}
if (-not $PubKey) { throw "Set `$PubKey in build.ps1 first (go run ./tools/signer keygen)." }
if (-not (Test-Path signing.key)) { throw "signing.key not found in this folder." }

$out = "release"
New-Item -ItemType Directory -Force $out | Out-Null
$file = "ElectricEditor.exe"

$ld = "-s -w -X main.version=$Version -X main.updateRepo=$Repo -X main.publicKey=$PubKey"
if (-not (Test-Path license_keys.go)) { throw "license_keys.go is missing - official builds need the private key system." }
wails build -clean -trimpath -tags keys -ldflags $ld -o $file
if ($LASTEXITCODE -ne 0) { throw "wails build failed (see the error above)." }
Copy-Item "build\bin\$file" "$out\$file" -Force

go run ./tools/signer sign "$out\$file"
if ($LASTEXITCODE -ne 0) { throw "Signing failed." }

Write-Host ""
Write-Host "Done: $out\$file and $out\$file.sig"
Write-Host "Create a GitHub release tagged v$Version and upload both files."
