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

## Keys and updates

There's one download, `ElectricEditor.exe`. Keys unlock extras (Manage → Enter a key):

| Key tier | Unlocks |
|---|---|
| none | everything a regular user needs |
| `donator` | a Supporter badge (more perks can hang off this later) |
| `dev` | research tools in the Manage menu, and pre-release updates |
| `amy` | Amy's theme and greeting |

A key is signed with the maintainer's private key, so it can't be forged. To
turn off a leaked key, add its ID to `revoked.txt` in this repo; the app checks
that file on launch. New tiers go in `edition.go`.

On launch the app checks GitHub Releases for a newer `ElectricEditor.exe`,
verifies its signature against the public key built into the app, installs it
and offers a restart. Unsigned or wrongly signed files are refused.

Maintainer setup (once, in this folder):

1. `go run ./tools/signer keygen`, (build.ps1 reads the public key from `signing.key` itself),
   and back up `signing.key` (git-ignored; losing it means installed copies can't update).

Making keys:

```
go run ./tools/signer key amy Amy
go run ./tools/signer key donator "Their name"
go run ./tools/signer key dev Jamie
```

Each key's ID is logged to `issued-keys.txt` (git-ignored) so you can revoke it later.

Releasing:

1. `./build.ps1 1.2.0` builds and signs `release/ElectricEditor.exe`.
2. Create a GitHub release tagged `v1.2.0` and upload the `.exe` and `.exe.sig`.
   Mark it as a pre-release to send it only to dev keys.

Any copy can show a background image (Manage → Set background image). It's
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

## Claw trail colours (mod pak)

Some effects (Devil Jin's claw trails) are materials, which Tekken won't
recolour live. For those the app builds a small mod in
`Polaris\Content\Paks\mods\ElectricEditor_P.*`:

1. the original materials are extracted once with retoc (`to-legacy`), cached
   in `%APPDATA%\ElectricEditor\pak\original` and re-extracted after a game update
2. their colour parameters are patched in Go (`internal/pak/legacy.go`)
3. retoc `to-zen --version UE5_2` packs them, and the files are copied to `mods`

Setting every claw layer back to "Game colours" deletes the mod. If Tekken has
the files open, the change is applied as soon as it closes. Logs are in
`%APPDATA%\ElectricEditor\pak\log.txt`.

## Credits

Bundles [retoc](https://github.com/trumank/retoc) by Truman Kilen and
Archengius (MIT licence, see `internal/pak/RETOC_LICENSE.txt`).
