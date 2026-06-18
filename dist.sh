#!/usr/bin/env bash
# Build phantom-http-server and assemble ./dist/ for console deployment.
#
# Usage:
#   ./dist.sh            # native binary -> dist/phantom-http-server
#   ./dist.sh --amd64    # native + linux/amd64 -> dist/phantom-http-server-amd64
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

DIST="$ROOT/dist"
BUILD_AMD64=0

usage() {
    cat <<'EOF'
Usage: ./dist.sh [OPTIONS]

Options:
  --amd64    Also build linux/amd64 binary (dist/phantom-http-server-amd64)
  -h, --help Show this help
EOF
}

for arg in "$@"; do
    case "$arg" in
        --amd64) BUILD_AMD64=1 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "error: unknown option: $arg" >&2; usage >&2; exit 1 ;;
    esac
done

build_binary() {
    local out="$1"
    local goos="${2:-}"
    local goarch="${3:-}"

    if [[ -n "$goos" && -n "$goarch" ]]; then
        echo "==> building phantom-http-server (${goos}/${goarch})"
        CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
            go build -trimpath -ldflags="-s -w" -o "$out" ./cmd/server
    else
        echo "==> building phantom-http-server (native)"
        CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$out" ./cmd/server
    fi
}

build_binary "$DIST/phantom-http-server"
if [[ "$BUILD_AMD64" -eq 1 ]]; then
    build_binary "$DIST/phantom-http-server-amd64" linux amd64
fi

echo "==> preparing dist layout"
mkdir -p "$DIST/web" "$DIST/logs"

echo "==> copying web assets"
for f in index.html detail.html style.css themes.css theme.js app.js; do
    cp "web/$f" "$DIST/web/$f"
done

echo "==> copying settings"
if [[ -f "$DIST/setting.yml" ]]; then
    echo "    kept existing dist/setting.yml (not overwritten)"
elif [[ -f "setting.yml" ]]; then
    cp setting.yml "$DIST/setting.yml"
    echo "    copied from ./setting.yml"
else
    cp settings.example/setting.yml "$DIST/setting.yml"
    echo "    copied from ./settings.example/setting.yml"
fi

chmod +x "$DIST/phantom-http-server" "$DIST/run.sh" "$DIST/stop.sh" "$DIST/status.sh"
[[ -f "$DIST/phantom-http-server-amd64" ]] && chmod +x "$DIST/phantom-http-server-amd64"

echo ""
echo "done:"
echo "  native : $DIST/phantom-http-server"
if [[ -f "$DIST/phantom-http-server-amd64" ]]; then
    echo "  amd64  : $DIST/phantom-http-server-amd64"
fi
echo "  run    : ./dist/run.sh"
echo "  stop   : ./dist/stop.sh"
echo "  status : ./dist/status.sh"
