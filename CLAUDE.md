# SC Bridge Companion — Agent Instructions

## Release Process

When tagging a new release:

1. **Update the hardcoded version** in `app.go` (`var Version = "X.Y.Z"`) to match the new tag
2. **Update `wails.json`** `productVersion` to match
3. Commit the version bump
4. Tag with `vX.Y.Z` and push the tag

The CI injects the version via `-ldflags "-X main.Version=..."` but Wails may not always pass ldflags through correctly. The hardcoded fallback **must** stay in sync to prevent self-update restart loops — the app compares its compiled version against the latest GitHub release tag, and a stale fallback causes it to think it always needs updating.

## Build & CI

- CI runs on `windows-latest` via GitHub Actions, triggered by `v*` tag pushes
- Release artifacts use **stable names** (no version suffix): `SCBridgeCompanion-portable.exe`, `SCBridgeCompanion-setup.msi`
- Version lives in the git tag only — `/releases/latest/download/` URLs always resolve to the current release
- SignPath code signing is configured but not yet active

## Code Signing

The app is **not yet code-signed**. Users may see Windows SmartScreen warnings. They need to right-click the file, go to Properties, and check "Unblock". This will go away once SignPath is configured with a valid signing certificate.
