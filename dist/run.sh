#!/usr/bin/env bash
# Start phantom-http-server as a background daemon.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

BIN="$ROOT/phantom-http-server"
PID_FILE="$ROOT/phantom-http-server.pid"
ENV_FILE="$ROOT/phantom-http-server.env"

H2H_SETTING_FILE="${H2H_SETTING_FILE:-$ROOT/setting.yml}"

if [[ ! -x "$BIN" ]]; then
    echo "error: $BIN not found. Run ../dist.sh from the project root first." >&2
    exit 1
fi

if [[ ! -f "$H2H_SETTING_FILE" ]]; then
    echo "error: settings file not found: $H2H_SETTING_FILE" >&2
    exit 1
fi

if [[ -f "$PID_FILE" ]]; then
    pid="$(cat "$PID_FILE")"
    if kill -0 "$pid" 2>/dev/null; then
        echo "already running (pid $pid)"
        exit 0
    fi
    rm -f "$PID_FILE"
fi

mkdir -p "$ROOT/logs"

export H2H_SETTING_FILE

cat >"$ENV_FILE" <<EOF
H2H_SETTING_FILE=$H2H_SETTING_FILE
EOF

nohup "$BIN" >>"$ROOT/logs/nohup.log" 2>&1 &
echo $! >"$PID_FILE"

sleep 0.3
pid="$(cat "$PID_FILE")"
if kill -0 "$pid" 2>/dev/null; then
    echo "started (pid $pid)"
    echo "  settings: ${H2H_SETTING_FILE}"
    echo "  log dir  : ${ROOT}/logs"
    echo "  gui      : http://127.0.0.1:8080/ (see setting.yml for port/scheme)"
else
    echo "error: failed to start. See $ROOT/logs/nohup.log" >&2
    rm -f "$PID_FILE" "$ENV_FILE"
    exit 1
fi
