#!/usr/bin/env python3
"""Compare the native RustDesk client command payloads with this API's request fields."""
import pathlib
import re
import time
import urllib.error
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[1]
CLIENT_VERSIONS = ("1.4.5", "1.5.0", "1.5.1")
ASSIGN_FLAG = re.compile(r'get_value\("--([a-z_]+)"\)')
JSON_TAG = re.compile(r'json:"([a-z_]+)"')


def source(version, path):
    url = f"https://raw.githubusercontent.com/rustdesk/rustdesk/{version}/{path}"
    last_error = None
    for attempt in range(3):
        try:
            request = urllib.request.Request(url, headers={"User-Agent": "rustdesk-compat-check"})
            with urllib.request.urlopen(request, timeout=30) as response:
                return response.read(1024 * 1024).decode("utf-8")
        except (OSError, urllib.error.URLError) as exc:
            last_error = exc
            if attempt < 2:
                time.sleep(2)
    raise RuntimeError(f"Unable to read upstream RustDesk {version} source: {last_error}")


def extract_go_fields(contents, struct_name):
    match = re.search(r"type\s+" + re.escape(struct_name) + r"\s+struct\s*\{([^}]+)\}", contents)
    if match is None:
        raise AssertionError(f"Missing request structure: {struct_name}")
    return set(JSON_TAG.findall(match.group(1)))


def check_assign(version, source_code, available_fields):
    marker = 'args[0] == "--assign"'
    start = source_code.find(marker)
    if start == -1:
        raise AssertionError(f"{version}: native --assign is missing")
    end = source_code.find('args[0] == "--', start + len(marker))
    body = source_code[start:end if end > start else start + 8000]
    if "/api/devices/cli" not in body:
        raise AssertionError(f"{version}: unexpected device assignment route")
    if '"id": id' not in body or '"uuid": uuid' not in body:
        raise AssertionError(f"{version}: native device identity fields changed")
    fields = set(ASSIGN_FLAG.findall(body))
    missing = fields - available_fields
    if missing:
        raise AssertionError(f"{version}: unrecognized assignment fields: {sorted(missing)}")
    if len(fields) < 11:
        raise AssertionError(f"{version}: assignment options could not be read completely")
    print(f"{version}: native --assign recognized, {len(fields)} optional fields matched")


def check_deploy(version, core_main, ui_interface, available_fields):
    marker = 'args[0] == "--deploy"'
    if marker not in core_main:
        if version != "1.4.5":
            raise AssertionError(f"{version}: expected native --deploy support")
        print(f"{version}: native --deploy is not present; this command is not supported")
        return
    match = re.search(
        r"pub fn deploy_device\([^\n]+\).*?"
        r"let body\s*=\s*serde_json::json!\(\{([^}]+)\}\)",
        ui_interface, re.DOTALL,
    )
    if not match:
        raise AssertionError(f"{version}: native deployment request was not found")
    fields = set(re.findall(r'"([a-z_]+)"\s*:', match.group(1)))
    if fields != {"id", "uuid", "pk"} or not fields.issubset(available_fields):
        raise AssertionError(f"{version}: unexpected native deployment fields {fields}")
    if '"/api/devices/deploy"' not in ui_interface:
        raise AssertionError(f"{version}: native deployment route changed")
    for expected in ("OK", "NOT_ENABLED", "INVALID_INPUT", "ID_TAKEN"):
        if f'"{expected}"' not in ui_interface:
            raise AssertionError(f"{version}: client cannot interpret {expected}")
    print(f"{version}: native --deploy fields and response codes matched")


def main():
    api = (ROOT / "http/controller/api/device_commands.go").read_text(encoding="utf-8")
    routes = (ROOT / "http/router/api.go").read_text(encoding="utf-8")
    assign_fields = extract_go_fields(api, "deviceAssignForm")
    deploy_fields = extract_go_fields(api, "deviceDeploymentRequest")
    if "Assign" not in routes or "Deploy" not in routes:
        raise AssertionError("The official client command routes are not registered")
    for version in CLIENT_VERSIONS:
        core = source(version, "src/core_main.rs")
        check_assign(version, core, assign_fields)
        if version == "1.4.5":
            check_deploy(version, core, "", deploy_fields)
        else:
            ui = source(version, "src/ui_interface.rs")
            check_deploy(version, core, ui, deploy_fields)
    print("Official native command compatibility source checks passed.")


if __name__ == "__main__":
    main()
