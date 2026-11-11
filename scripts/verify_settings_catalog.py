#!/usr/bin/env python3
"""Verify that every registered setting has a documented purpose and environment name."""
import json
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parents[1]
settings_source = (ROOT / "config" / "settings.go").read_text(encoding="utf-8")
reference = (ROOT / "docs" / "CONFIG_REFERENCE.md").read_text(encoding="utf-8")
matched = re.search(r'var settingsMetadata = ("(?:[^"\\]|\\.)*")', settings_source)
if not matched:
    raise SystemExit("Cannot read the settings catalog")
lines = json.loads(matched.group(1)).splitlines()
keys = set()
for line in lines:
    parts = line.split("|", 2)
    if len(parts) != 3 or not all(parts):
        raise SystemExit("An entry in the settings catalog lacks a name or description")
    key, name, description = parts
    if key in keys:
        raise SystemExit("A settings catalog entry is repeated: " + key)
    keys.add(key)
    if f"| `{key}` |" not in reference:
        raise SystemExit("The configuration reference does not describe: " + key)
if len(keys) < 90:
    raise SystemExit("The settings catalog is incomplete")

excluded = {ROOT / "config" / "config.go", ROOT / "config" / "settings.go"}
pattern = re.compile(r'os\.(?:Getenv|LookupEnv)\(\s*"([^"]+)"\s*\)')
for file in ROOT.rglob("*.go"):
    if file.name.endswith("_test.go") or file.resolve() in excluded:
        continue
    if "vendor" in file.parts:
        continue
    source = file.read_text(encoding="utf-8")
    for match in pattern.finditer(source):
        name = match.group(1)
        if name.startswith("RUSTDESK_"):
            raise SystemExit(f"Direct RustDesk environment variable read in {file.relative_to(ROOT)}: {name}")
print(f"Checked {len(keys)} documented setting descriptions and runtime environment access")
