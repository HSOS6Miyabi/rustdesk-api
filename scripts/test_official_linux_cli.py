#!/usr/bin/env python3
"""Exercise official installed RustDesk 1.5.1 Linux CLI against a loopback API.

The isolated HTTP server records native requests and returns protocol-compatible
responses. The test does not contact any real RustDesk deployment.
"""
import contextlib
import http.server
import json
import os
import subprocess
import threading

CLIENT_ID = "rustdesk-test-cli-100001"
TOKEN = "linux-cli-test-token"


class LocalAPI(http.server.BaseHTTPRequestHandler):
    requests = []

    def do_POST(self):
        try:
            size = int(self.headers.get("Content-Length", "0"))
            if size <= 0 or size > 4096:
                self.send_error(400, "invalid native request")
                return
            payload = json.loads(self.rfile.read(size))
            LocalAPI.requests.append((self.path, self.headers.get("Authorization"), payload))
            if self.headers.get("Authorization") != "Bearer " + TOKEN:
                self.send_error(401)
                return
            if self.path == "/api/devices/deploy":
                if not all(payload.get(field) for field in ("id", "uuid", "pk")):
                    self.send_error(400)
                    return
                encoded = b'{"result":"OK"}'
            elif self.path == "/api/devices/cli":
                if not all(payload.get(field) for field in ("id", "uuid")):
                    self.send_error(400)
                    return
                encoded = b""
            else:
                self.send_error(404)
                return
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
        except (ValueError, json.JSONDecodeError):
            self.send_error(400, "invalid JSON")

    def log_message(self, _format, *_args):
        pass


def call(*args):
    result = subprocess.run(
        ["rustdesk", *args],
        capture_output=True, text=True, timeout=30, check=False,
        env=os.environ.copy(),
    )
    message = (result.stdout + result.stderr).strip()
    if result.returncode:
        raise AssertionError(f"official CLI {args[0]} exited with {result.returncode}: {message[-600:]}")
    return message


def main():
    if os.geteuid() != 0:
        raise RuntimeError("official Linux CLI integration must run with administrator privileges")
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), LocalAPI)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        url = f"http://127.0.0.1:{server.server_port}"
        call("--option", "api-server", url)
        readback = call("--option", "api-server")
        if readback.strip() != url:
            raise AssertionError(f"the installed RustDesk client did not save its API endpoint: {readback!r}")
        deployed = call("--deploy", "--token", TOKEN, "--id", CLIENT_ID)
        if "Device deployed." not in deployed:
            raise AssertionError(f"the official deployment command did not succeed: {deployed}")
        assigned = call("--assign", "--token", TOKEN, "--device_name", "CI Office")
        if "Done!" not in assigned:
            raise AssertionError(f"the official assignment command did not succeed: {assigned}")
        deployments = [item for item in LocalAPI.requests if item[0] == "/api/devices/deploy"]
        assignments = [item for item in LocalAPI.requests if item[0] == "/api/devices/cli"]
        if len(deployments) != 1 or len(assignments) != 1:
            raise AssertionError("the client did not submit both official API requests")
        if deployments[0][2]["id"] != CLIENT_ID:
            raise AssertionError("the custom device ID was not sent with deployment")
        if assignments[0][2]["id"] != CLIENT_ID:
            raise AssertionError("the deployed device ID was not retained for assignment")
        if assignments[0][2].get("device_name") != "CI Office":
            raise AssertionError("the assignment options were not transmitted")
        if deployments[0][2]["uuid"] != assignments[0][2]["uuid"]:
            raise AssertionError("the device UUID changed between commands")
        if len(deployments[0][2]["pk"]) < 40:
            raise AssertionError("the official public key was not transmitted")
        print("Installed RustDesk 1.5.1 Linux client invoked --deploy and --assign successfully.")
        print("Device ID, UUID, public key, authorization and assignment fields were verified.")
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


if __name__ == "__main__":
    main()
