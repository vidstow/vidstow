#!/bin/sh
set -eu

app_path=${1:-}
if [ -z "$app_path" ]; then
  echo "package-js-helper: missing Wails output path" >&2
  exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)

case "$app_path" in
  */Contents/MacOS/*)
    helper_dir=$(CDPATH= cd -- "$(dirname -- "$app_path")" && pwd)
    app_bundle=${app_path%/Contents/MacOS/*}
    ;;
  *)
    helper_dir=$(CDPATH= cd -- "$(dirname -- "$app_path")" && pwd)
    app_bundle=
    ;;
esac

helper_name=ytdlp-js-helper
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*|Windows_NT)
    helper_name=ytdlp-js-helper.exe
    ;;
esac

# Wails may invoke this script through bash on Windows even when GOOS=windows.
case "$app_path" in
  *.exe)
    helper_name=ytdlp-js-helper.exe
    ;;
esac

mkdir -p "$helper_dir"
helper_path="$helper_dir/$helper_name"
temporary_path="$helper_path.tmp.$$"
trap 'rm -f "$temporary_path"' EXIT HUP INT TERM

printf '%s\n' "package-js-helper: building sibling helper ($helper_name)"
(
  cd "$repo_dir"
  if [ "$(uname -s)" = "Darwin" ]; then
    # The macOS helper uses its pinned, bare QuickJS runtime for current
    # YouTube EJS workloads; other targets retain the pure-Go fallback.
    CGO_ENABLED=1 go build -trimpath -o "$temporary_path" github.com/tejasa97/ytdlp-go/cmd/ytdlp-js-helper
  else
    CGO_ENABLED=0 go build -trimpath -o "$temporary_path" github.com/tejasa97/ytdlp-go/cmd/ytdlp-js-helper
  fi
)
chmod 755 "$temporary_path" 2>/dev/null || true
mv -f "$temporary_path" "$helper_path"

"$script_dir/verify-js-helper.sh" "$app_path"

if [ "$(uname -s)" = "Darwin" ] && [ -n "$app_bundle" ] && [ -d "$app_bundle" ]; then
  engine_dir=$(cd "$repo_dir" && go list -m -f '{{.Dir}}' github.com/tejasa97/ytdlp-go)
  legal_dir="$app_bundle/Contents/Resources/legal"
  engine_licenses="$legal_dir/ytdlp-go-licenses"
  mkdir -p "$engine_licenses"
  chmod -R u+w "$legal_dir" 2>/dev/null || true
  cp "$repo_dir/LICENSE" "$legal_dir/VidStow-LICENSE"
  cp "$repo_dir/NOTICE" "$legal_dir/VidStow-NOTICE"
  cp "$engine_dir/LICENSE" "$legal_dir/ytdlp-go-LICENSE"
  cp "$engine_dir/THIRD_PARTY_NOTICES.md" "$legal_dir/ytdlp-go-THIRD_PARTY_NOTICES.md"
  for license in "$engine_dir"/third_party/licenses/*; do
    if [ -f "$license" ]; then
      cp "$license" "$engine_licenses/"
    fi
  done
  test -f "$engine_licenses/quickjs-go-v0.7.7.LICENSE"
  test -f "$engine_licenses/quickjs-ng.LICENSE"
  find "$legal_dir" -type f -exec chmod 444 {} \;

  printf '%s\n' "package-js-helper: ad-hoc re-signing bundle after helper and legal notice placement"
  /usr/bin/codesign --force --deep --sign - "$app_bundle"
  /usr/bin/codesign --verify --deep --strict "$app_bundle"
fi

printf '%s\n' "package-js-helper: verified $helper_path"
