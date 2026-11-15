# Releasing Vocabulary Studio

The supported macOS install is a binary Homebrew cask in
[`veryshyjelly/homebrew-tap`](https://github.com/veryshyjelly/homebrew-tap).
Users run `brew install --cask veryshyjelly/tap/vocabulary-studio`. Installation
requires macOS 13 or later, matching Go 1.27's minimum macOS version.

## Build and publish

1. Update `frontend/package.json` to the release version (`X.Y.Z`, without a `v`).
   The packaging script copies this version into both app bundle version fields.
2. Run `make test` and `make package-macos`. Review and commit the changes,
   including the workflow and packaging scripts for the first release.
3. Push the commit, then create and push the matching stable tag:

   ```sh
   git tag -a v1.0.0 -m "Release Vocabulary Studio v1.0.0"
   git push origin v1.0.0
   ```

The `Release macOS app` workflow rejects version mismatches and unstable tags,
builds and tests on Apple Silicon and Intel macOS runners, verifies the bundles,
and publishes a GitHub release only after both jobs succeed. Assets are:

- `vocabulary-studio_X.Y.Z_darwin_arm64.zip`
- `vocabulary-studio_X.Y.Z_darwin_amd64.zip`
- `checksums.txt`

Each ZIP contains `Vocabulary Studio.app` at its root and preserves executable
permissions and the app's code signature. To test the workflow before publishing,
use **Actions → Release macOS app → Run workflow** on a branch. Branch runs upload
build artifacts without publishing a release. Tag runs publish a release.

Do not replace assets of a published version: Homebrew verifies their SHA-256
hashes. Fix a release by incrementing the version and publishing a new tag.

## Update the Homebrew tap

The tap's hourly `Update Homebrew packages` workflow downloads the latest stable
release, verifies both architecture archives against `checksums.txt`, and commits
`Casks/vocabulary-studio.rb`. It uses the tap's own `GITHUB_TOKEN`; the application
repository needs no cross-repository token. Trigger that workflow manually for
an immediate update after publishing a release.

Alternatively, from the tap checkout:

```sh
python3 -m unittest discover -s scripts
python3 scripts/update_vocabulary_studio.py
brew style ./Casks/vocabulary-studio.rb
```

Review, commit, and push the generated cask. The updater skips a repository with
no published release and never writes a cask with placeholder URLs or hashes.
The first cask must be generated after the first release is published.

After the cask is published, validate it through the registered tap:

```sh
brew tap veryshyjelly/tap
brew trust --cask veryshyjelly/tap/vocabulary-studio # Homebrew 7 and newer
brew audit --cask --online veryshyjelly/tap/vocabulary-studio
```

## Signing and first launch

Builds currently use ad-hoc signing (`codesign --sign -`). They are not Apple
notarized, so macOS may block downloaded copies on first launch. The README,
release notes, and cask explain **System Settings → Privacy & Security → Open
Anyway**. Homebrew does not remove quarantine or change Gatekeeper settings.

To eliminate the prompt in a future release, use a Developer ID Application
certificate, sign with hardened runtime, notarize, and staple the ticket before
creating the ZIP. Keep signing credentials in GitHub Actions secrets. Ad-hoc
signing alone does not provide Apple notarization.
