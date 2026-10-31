#!/bin/sh
# Build and package a native macOS app for local installation.
set -eu

if [ "$(uname -s)" != "Darwin" ]; then
  printf '%s\n' 'This packaging script must run on macOS.' >&2
  exit 1
fi

package_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$package_root"

for package_tool in go bun python3 wails3 qlmanage codesign plutil; do
  if ! command -v "$package_tool" >/dev/null 2>&1; then
    printf 'Missing prerequisite: %s. See README.md for setup.\n' "$package_tool" >&2
    exit 1
  fi
done

printf '%s\n' 'Installing frontend dependencies and building embedded assets…'
(
  cd frontend
  bun install --frozen-lockfile
  bun run build
)

printf '%s\n' 'Building the production executable for this Mac…'
mkdir -p bin dist
CGO_ENABLED=1 GOOS=darwin GOARCH="$(go env GOHOSTARCH)" \
  go build -tags production -trimpath -buildvcs=false -ldflags='-w -s' \
  -o bin/vocab-desktop ./cmd/vocab-desktop

package_tmp=$(mktemp -d "${TMPDIR:-/tmp}/vocabulary-package.XXXXXX")
trap 'rm -rf "$package_tmp"' EXIT
trap 'exit 1' HUP INT TERM
package_app="$package_tmp/Vocabulary Studio.app"
mkdir -p "$package_app/Contents/MacOS" "$package_app/Contents/Resources"
cp bin/vocab-desktop "$package_app/Contents/MacOS/vocab-desktop"

printf '%s\n' 'Packaging the app icon and metadata…'
qlmanage -t -s 1024 -o "$package_tmp" frontend/public/favicon.svg
wails3 generate icons \
  -input "$package_tmp/favicon.svg.png" \
  -macfilename "$package_app/Contents/Resources/icons.icns" \
  -windowsfilename "$package_tmp/icons.ico"

python3 - "$package_app" <<'PY'
import json
import plistlib
import sys
from pathlib import Path

app = Path(sys.argv[1])
version = json.loads(Path("frontend/package.json").read_text())["version"]
info = {
    "CFBundlePackageType": "APPL",
    "CFBundleName": "Vocabulary Studio",
    "CFBundleDisplayName": "Vocabulary Studio",
    "CFBundleExecutable": "vocab-desktop",
    "CFBundleIdentifier": "com.veryshyjelly.vocabularystudio",
    "CFBundleVersion": version,
    "CFBundleShortVersionString": version,
    "CFBundleIconFile": "icons",
    "LSMinimumSystemVersion": "12.0",
    "NSHighResolutionCapable": True,
}
(app / "Contents/Info.plist").write_bytes(plistlib.dumps(info))
PY

printf '%s\n' 'Signing and verifying the local app bundle…'
codesign --force --deep --sign - "$package_app"
codesign --verify --deep --strict "$package_app"
plutil -lint "$package_app/Contents/Info.plist"

# Replace only the generated bundle; leave installed copies untouched.
rm -rf "$package_root/dist/Vocabulary Studio.app"
mv "$package_app" "$package_root/dist/Vocabulary Studio.app"
printf '\nBuilt: %s/dist/Vocabulary Studio.app\n' "$package_root"
printf '%s\n' 'Run "open dist" and drag Vocabulary Studio.app into Applications.'
