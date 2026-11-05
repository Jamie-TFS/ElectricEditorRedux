# Builds and signs the release exe into .\release
#
#   .\build.ps1 1.0.0            (stable release)
#   .\build.ps1 1.1.0-beta.1     (pre-release: only dev and supporter keys get these)
#
# Needs signing.key in this folder (go run ./tools/signer keygen, once ever).
# The public key is worked out from it, so there's nothing to paste in here.
param([Parameter(Mandatory = $true)][string]$Version)

$ErrorActionPreference = "Stop"
$Repo = "Jamie-TFS/ElectricEditorRedux"

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
if (-not (Test-Path signing.key)) { throw "signing.key not found in this folder." }
# signing.key is a base64 ed25519 private key: bytes 32-63 are the public key
$keyBytes = [Convert]::FromBase64String((Get-Content signing.key -Raw).Trim())
if ($keyBytes.Length -ne 64) { throw "signing.key doesn't look right (expected 64 bytes)." }
$PubKey = [Convert]::ToBase64String($keyBytes[32..63])
Write-Host "Public key: $PubKey"

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
