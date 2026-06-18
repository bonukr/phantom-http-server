#!/usr/bin/env bash
# Show phantom-http-server daemon status.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="$ROOT/phantom-http-server.pid"
ENV_FILE="$ROOT/phantom-http-server.env"

if [[ -f "$ENV_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$ENV_FILE"
fi
H2H_SETTING_FILE="${H2H_SETTING_FILE:-$ROOT/setting.yml}"

running=0
pid=""

if [[ -f "$PID_FILE" ]]; then
    pid="$(cat "$PID_FILE")"
    if kill -0 "$pid" 2>/dev/null; then
        running=1
    fi
fi

if [[ "$running" -eq 1 ]]; then
    echo "status   : running"
    echo "pid      : $pid"
    echo "settings : $H2H_SETTING_FILE"
    if command -v curl >/dev/null 2>&1; then
        port="$(grep -E '^[[:space:]]*port:' "$H2H_SETTING_FILE" 2>/dev/null | head -1 | awk '{print $2}')"
        port="${port:-8080}"
        if resp="$(curl -fsS --max-time 2 "http://127.0.0.1:${port}/api/status" 2>/dev/null)"; then
            echo "api      : $resp"
            echo "url      : http://127.0.0.1:${port}/"
        else
            echo "api      : (no response yet — check TLS settings)"
        fi
    fi
    exit 0
fi

echo "status : stopped"
if [[ -n "$pid" ]]; then
    echo "note   : stale pid file ($pid)"
fi
exit 1
