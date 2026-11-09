#!/usr/bin/env python3
"""Run the installed official RustDesk Linux CLI against an actual local API process."""

import base64
import json
import os
from pathlib import Path
import secrets
import shutil
import sqlite3
import subprocess
import sys
import tempfile
import time
from urllib.error import URLError
from urllib.request import Request, urlopen
import socket


def available_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def client_cli(*args):
    completed = subprocess.run(
        ["rustdesk", *args],
        capture_output=True, text=True, timeout=40, check=False,
        env=os.environ.copy(),
    )
    result = (completed.stdout + completed.stderr).strip()
    if completed.returncode:
        raise AssertionError(
            f"Official RustDesk {args[0]} returned {completed.returncode}: {result[-500:]}"
        )
    return result


def wait_for_api(url, process):
    deadline = time.monotonic() + 50
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError("API process terminated during startup")
        try:
            with urlopen(url + "/_admin/", timeout=3) as response:
                if response.status == 200:
                    return
        except (OSError, URLError):
            pass
        time.sleep(0.25)
    raise TimeoutError("API management page was not ready")


def api_request(url, path, payload):
    request = Request(
        url + path, data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"}, method="POST",
    )
    with urlopen(request, timeout=8) as response:
        return response.read().decode("utf-8")


def issue_isolated_token(dbfile):
    token = secrets.token_urlsafe(32)
    with sqlite3.connect(dbfile, timeout=12) as db:
        db.execute("PRAGMA busy_timeout = 10000")
        admin = db.execute(
            "SELECT id FROM users WHERE username = 'admin'"
        ).fetchone()
        if admin is None:
            raise AssertionError("API did not create the isolated administrator account")
        db.execute(
            "INSERT INTO user_tokens (user_id, token, expired_at) VALUES (?, ?, ?)",
            (admin[0], token, int(time.time()) + 1800),
        )
    return admin[0], token


def main():
    if os.geteuid() != 0:
        raise RuntimeError("Installed RustDesk Linux CLI commands require administrator privileges")
    if len(sys.argv) != 2:
        raise SystemExit("Usage: test_official_linux_real_api.py /path/to/apimain")
    binary = Path(sys.argv[1]).resolve()
    if not binary.is_file():
        raise FileNotFoundError(binary)
    source = Path(__file__).resolve().parents[1]
    port = available_port()
    url = f"http://127.0.0.1:{port}"
    with tempfile.TemporaryDirectory(prefix="rustdesk-official-client-api-") as directory:
        root = Path(directory)
        shutil.copytree(source / "conf", root / "conf")
        shutil.copytree(
            source / "resources", root / "resources",
            ignore=shutil.ignore_patterns("node_modules", ".git"),
        )
        (root / "data").mkdir()
        (root / "runtime").mkdir()
        config = root / "conf/config.yaml"
        original = 'api-addr: "0.0.0.0:21114"'
        contents = config.read_text(encoding="utf-8")
        if original not in contents:
            raise AssertionError("Standard API listen address is missing")
        config.write_text(
            contents.replace(original, f'api-addr: "127.0.0.1:{port}"'),
            encoding="utf-8",
        )
        env = os.environ.copy()
        env.update({
            "REQUIRE_DEVICE_DEPLOYMENT": "Y",
            "RUSTDESK_API_INTERNAL_KEY": secrets.token_urlsafe(32),
        })
        logpath = root / "api.log"
        with logpath.open("wb") as stream:
            process = subprocess.Popen(
                [str(binary), "--config", "./conf/config.yaml"],
                cwd=root, env=env, stdout=stream, stderr=subprocess.STDOUT,
            )
            try:
                wait_for_api(url, process)
                admin_id, token = issue_isolated_token(root / "data/rustdeskapi.db")
                client_cli("--option", "api-server", url)
                if client_cli("--option", "api-server").strip() != url:
                    raise AssertionError("Official client did not retain the local API address")
                device_id = client_cli("--get-id").strip()
                if not (6 <= len(device_id) <= 253):
                    raise AssertionError("Official client returned an invalid device ID")
                result = client_cli("--deploy", "--token", token)
                if "Device deployed." not in result:
                    raise AssertionError("Official deployment did not return success")
                with sqlite3.connect(root / "data/rustdeskapi.db", timeout=10) as db:
                    row = db.execute(
                        "SELECT uuid, public_key, owner_user_id, status "
                        "FROM device_deployments WHERE device_id = ?",
                        (device_id,),
                    ).fetchone()
                if row is None or row[2:] != (admin_id, "approved"):
                    raise AssertionError("Official deployment did not create the expected database row")
                if len(base64.b64decode(row[0])) < 8 or len(base64.b64decode(row[1])) != 32:
                    raise AssertionError("Official client supplied an invalid identity")
                result = client_cli("--deploy", "--token", token)
                if "Device deployed." not in result:
                    raise AssertionError("Repeated deployment was not idempotent")

                # A real registered device normally supplies this data through /api/sysinfo.
                response = api_request(url, "/api/sysinfo", {
                    "id": device_id, "uuid": row[0], "hostname": "ci-client",
                    "os": "Linux", "version": "1.5.1",
                })
                if response != "SYSINFO_UPDATED":
                    raise AssertionError(f"Official device record was not created: {response!r}")
                result = client_cli(
                    "--assign", "--token", token, "--device_name", "CI Office",
                )
                if "Done!" not in result:
                    raise AssertionError("Official assignment did not return success")
                with sqlite3.connect(root / "data/rustdeskapi.db", timeout=10) as db:
                    registered = db.execute(
                        "SELECT uuid, hostname FROM peers WHERE id = ?",
                        (device_id,),
                    ).fetchone()
                    deployment = db.execute(
                        "SELECT uuid, public_key, owner_user_id, status "
                        "FROM device_deployments WHERE device_id = ?",
                        (device_id,),
                    ).fetchone()
                if registered != (row[0], "CI Office") or deployment != row:
                    raise AssertionError("Official CLI assignment changed the device identity or failed to save")
                print(
                    "Official Linux 1.5.1 client: real API deployment and assignment succeeded; "
                    "SQLite identity, owner, repeat deployment and device name verified."
                )
            except Exception:
                stream.flush()
                print(
                    f"\nAPI output:\n{logpath.read_text(errors='replace')[-3500:]}",
                    file=sys.stderr,
                )
                raise
            finally:
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=8)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait(timeout=8)


if __name__ == "__main__":
    main()
