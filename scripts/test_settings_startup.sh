#!/usr/bin/env bash
set -euo pipefail

binary="${1:?Pass the compiled apimain executable}"
repo_dir="$(pwd)"
test_dir="$(mktemp -d)"
server_pid=""
cleanup() {
    if [[ -n "$server_pid" ]]; then
        kill -TERM "$server_pid" 2>/dev/null || true
        wait "$server_pid" 2>/dev/null || true
    fi
    rm -rf "$test_dir"
}
trap cleanup EXIT
mkdir -p "$test_dir/data" "$test_dir/runtime"
cp -R "$repo_dir/conf" "$test_dir/conf"
ln -s "$repo_dir/resources" "$test_dir/resources"

port="$(python3 - <<'PY'
import socket
s=socket.socket()
s.bind(("127.0.0.1",0))
print(s.getsockname()[1])
s.close()
PY
)"
sed -i "s/0.0.0.0:21114/127.0.0.1:$port/" "$test_dir/conf/config.yaml"
url="http://127.0.0.1:$port/api/admin/config/admin"

start() {
    (cd "$test_dir" && RUSTDESK_API_ADMIN_TITLE="Environment title" "$binary" > "$test_dir/server.log" 2>&1) &
    server_pid="$!"
    for _ in $(seq 1 80); do
        if curl -fsS --max-time 1 "$url" > "$test_dir/response.json" 2>/dev/null; then
            return
        fi
        if ! kill -0 "$server_pid" 2>/dev/null; then
            cat "$test_dir/server.log"
            echo "RustDesk API exited before serving HTTP" >&2
            return 1
        fi
        sleep 0.15
    done
    cat "$test_dir/server.log"
    echo "RustDesk API did not become ready" >&2
    return 1
}

stop() {
    kill -TERM "$server_pid"
    for _ in $(seq 1 60); do
        if ! kill -0 "$server_pid" 2>/dev/null; then
            wait "$server_pid" 2>/dev/null || true
            server_pid=""
            return 0
        fi
        sleep 0.15
    done
    kill -KILL "$server_pid" || true
    wait "$server_pid" 2>/dev/null || true
    server_pid=""
    echo "RustDesk API did not stop after SIGTERM" >&2
    return 1
}
expect_title() {
    python3 - "$test_dir/response.json" "$1" <<'PY'
import json,sys
with open(sys.argv[1], encoding="utf-8") as file:
    response=json.load(file)
assert response["data"]["title"]==sys.argv[2],response
PY
}

start
expect_title "Environment title"
stop

python3 - "$test_dir/data/rustdeskapi.db" <<'PY'
import sqlite3,sys,datetime
with sqlite3.connect(sys.argv[1]) as connection:
    version=connection.execute("SELECT MAX(version) FROM versions").fetchone()[0]
    assert version==290, f"Unexpected database schema version {version}"
    connection.execute(
        "INSERT INTO system_setting_overrides (key,enabled,value,updated_at) VALUES (?,?,?,?)",
        ("admin.title",1,"Stored webpage title",datetime.datetime.now().isoformat())
    )
PY

start
expect_title "Stored webpage title"
stop

python3 - "$test_dir/data/rustdeskapi.db" <<'PY'
import sqlite3,sys
with sqlite3.connect(sys.argv[1]) as connection:
    connection.execute("UPDATE system_setting_overrides SET enabled=0 WHERE key='admin.title'")
PY
start
expect_title "Environment title"
stop
echo "Settings startup, persistence, precedence and graceful shutdown checks passed"
