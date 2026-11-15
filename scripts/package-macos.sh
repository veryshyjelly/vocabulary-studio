#!/bin/sh
# Build a macOS app and a versioned archive for local installation or releases.
set -eu

if [ "$(uname -s)" != "Darwin" ]; then
  printf '%s\n' 'This packaging script must run on macOS.' >&2
  exit 1
fi

package_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$package_root"

for package_tool in go bun python3 wails3 qlmanage codesign plutil ditto shasum; do
  if ! command -v "$package_tool" >/dev/null 2>&1; then
    printf 'Missing prerequisite: %s. See README.md for setup.\n' "$package_tool" >&2
    exit 1
  fi
done

package_arch=${GOARCH:-$(go env GOHOSTARCH)}
case "$package_arch" in
  arm64|amd64) ;;
  *) printf 'Unsupported macOS architecture: %s\n' "$package_arch" >&2; exit 1 ;;
esac
package_version=$(python3 - <<'PY'
import json
import re

version = json.load(open("frontend/package.json"))["version"]
if not re.fullmatch(r"(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)", version):
    raise SystemExit("frontend/package.json must contain a stable version, e.g. 1.0.0")
print(version)
PY
)

printf '%s\n' 'Installing frontend dependencies and building embedded assets…'
(
  cd frontend
  bun install --frozen-lockfile
  bun run build
)

printf 'Building the production executable for macOS/%s…\n' "$package_arch"
mkdir -p bin dist
CGO_ENABLED=1 GOOS=darwin GOARCH="$package_arch" MACOSX_DEPLOYMENT_TARGET=13.0 \
  CGO_CFLAGS="${CGO_CFLAGS:--O2 -g} -mmacosx-version-min=13.0" \
  CGO_LDFLAGS="${CGO_LDFLAGS:--O2 -g} -mmacosx-version-min=13.0" \
  go build -tags production -trimpath -buildvcs=false \
  -ldflags='-w -s -extldflags=-mmacosx-version-min=13.0' \
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
info = plistlib.loads(Path("build/macos/Info.plist").read_bytes())
info["CFBundleVersion"] = version
info["CFBundleShortVersionString"] = version
(app / "Contents/Info.plist").write_bytes(plistlib.dumps(info))
PY

printf '%s\n' 'Signing and verifying the local app bundle…'
codesign --force --deep --sign - "$package_app"
codesign --verify --deep --strict "$package_app"
plutil -lint "$package_app/Contents/Info.plist"

# Replace only the generated bundle; leave installed copies untouched.
rm -rf "$package_root/dist/Vocabulary Studio.app"
mv "$package_app" "$package_root/dist/Vocabulary Studio.app"

package_archive="vocabulary-studio_${package_version}_darwin_${package_arch}.zip"
printf 'Creating %s…\n' "$package_archive"
ditto -c -k --sequesterRsrc --keepParent \
  "$package_root/dist/Vocabulary Studio.app" "$package_root/dist/$package_archive"
(
  cd dist
  shasum -a 256 "$package_archive" > "$package_archive.sha256"
)
printf '\nBuilt: %s/dist/Vocabulary Studio.app\n' "$package_root"
printf 'Archive: %s/dist/%s\n' "$package_root" "$package_archive"
printf '%s\n' 'Run "open dist" and drag Vocabulary Studio.app into Applications.'
