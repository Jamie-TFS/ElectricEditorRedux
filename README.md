# Electric Editor

Recolour the Mishima electrics in Tekken 8, live, from a standalone app.
No UE4SS, no DLLs, no files copied into the game folder: run `ElectricEditor.exe`
alongside Tekken and it connects on its own.

By MonkeyTFS.

## Using it

1. Start Tekken 8 and Electric Editor (either order).
2. Pick a fighter, pick a layer, choose **Solid** or **Gradient**.
3. Changes show up in game straight away. The dot on each layer turns green
   when that effect is loaded in the current match.

Profiles keep separate colour sets. **Share** gives you a code (`EE1.…`) that
anyone can import.

Settings live in `%APPDATA%\ElectricEditor\`.

"Game colours" unhooks a layer: if the app changed it during this Tekken
session it puts back exactly what was there, then leaves it alone.

**Manage → Save curve list** writes every colour curve currently loaded to
`curves.txt`, which is how new moves get mapped.

## How it works

Tekken's electrics are Niagara effects whose colour over time comes from
`NiagaraDataInterfaceColorCurve` objects (their `ShaderLUT` arrays). The app:

- attaches to `Polaris-Win64-Shipping.exe` with standard Windows APIs,
- finds Unreal's name table and object array by searching for the data
  itself (the first name entries, and a self-consistent object-array header)
  instead of version-specific code patterns, so game patches rarely break it,
- indexes every colour curve, remembers the original values, and writes the
  colours you pick, re-applying whenever the game reloads an effect.

## Editions and updates

The same code builds several editions (see `edition.go`):

| Edition | File | Updates from | Extras |
|---|---|---|---|
| standard | `ElectricEditor.exe` | stable releases | |
| dev | `ElectricEditor-dev.exe` | stable + pre-releases | research tools in the Manage menu |
| amy | `ElectricEditor-amy.exe` | stable releases | Amy's theme and greeting |

On launch each edition checks GitHub Releases for a newer version of its own
file, verifies its ed25519 signature against the public key built into the app,
installs it, and offers a restart. Unsigned or wrongly signed files are refused.

Releasing:

1. One time: `go run ./tools/signer keygen`, paste the printed public key into
   `build.ps1`, and back up `signing.key` somewhere safe (it's git-ignored).
2. `./build.ps1 1.2.0` builds and signs every edition into `release/`.
3. Create a GitHub release tagged `v1.2.0` and upload every `.exe` and `.exe.sig`.
   Mark it as a pre-release to send it only to the dev edition.

Any edition can show a background image (Manage → Set background image). It's
stored only on that PC in `%APPDATA%\ElectricEditor\`.

## Building

Requires Go 1.26+, Node 20+ and the [Wails](https://wails.io) CLI.

```
wails build -platform windows/amd64
```

`wails dev` runs it with hot reload. Opening `frontend/` with `npm run dev` in a
normal browser uses a mock backend, which is handy for UI work.

Layout:

- `internal/game` – process attach, Unreal object/name lookup, curve read/write (Windows only)
- `internal/fx` – the effects catalog (which curves make up each layer) and colour rendering
- `internal/store` – profiles and share codes
- `engine.go` – background loop keeping the game in sync with the active profile
- `frontend/` – TypeScript UI (bolt preview, colour picker, gradient editor)

Adding a fighter or effect means adding its curve paths to `internal/fx/catalog.go`.
