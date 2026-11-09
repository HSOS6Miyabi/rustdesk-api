#!/usr/bin/env python3
"""Check that the RustDesk handoff documents cover every feature and task."""
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
main_path = ROOT / "docs" / "IMPLEMENTATION_LEDGER.md"
detail_path = ROOT / "docs" / "TASK_VERIFICATION_RECORD.md"
snapshot_path = ROOT / "docs" / "HANDOFF_SNAPSHOT_2026-11-09.md"
matrix_path = ROOT / "docs" / "FEATURE_MATRIX.md"
readme_path = ROOT / "README.md"
required_files = (
    main_path, detail_path, snapshot_path, matrix_path, readme_path,
    ROOT / "docs" / "API.md",
    ROOT / "docs" / "TESTING.md",
    ROOT / "docs" / "CLIENT_BUILDER.md",
    ROOT / "docs" / "DATABASE_BACKUP_RESTORE.md",
    ROOT / "docs" / "UPGRADE.md",
    ROOT / "docs" / "UPSTREAM_FEATURE_WATCH.md",
)

def check_ids(name, contents, pattern, expected):
    values = re.findall(pattern, contents, flags=re.MULTILINE)
    if len(values) != len(expected) or set(values) != expected:
        raise AssertionError(f"{name} identifiers are incomplete or repeated")

def check():
    for file in required_files:
        if not file.is_file():
            raise AssertionError(f"Required handoff file is missing: {file}")
    main = main_path.read_text(encoding="utf-8")
    details = detail_path.read_text(encoding="utf-8")
    snapshot = snapshot_path.read_text(encoding="utf-8")
    matrix = matrix_path.read_text(encoding="utf-8")
    readme = readme_path.read_text(encoding="utf-8")
    start = main.index("### 3. 功能实施状态")
    tasks_start = main.index("### 4. 完整待办事项及处理顺序", start)
    end = main.index("### 5. 重要实现细节", tasks_start)
    features = main[start:tasks_start]
    tasks = main[tasks_start:end]
    all_features = {f"F{i:02d}" for i in range(1, 21)}
    all_tasks = {f"T{i:03d}" for i in range(1, 23)}
    check_ids("Feature table", features, r"^\| (F\d{2}) \|", all_features)
    check_ids("Task table", tasks, r"^\| (T\d{3}) \|", all_tasks)
    check_ids("Detailed task register", details, r"^### (T\d{3})：", all_tasks)
    check_ids("Handoff snapshot", snapshot, r"^\| (T\d{3}) \|", all_tasks)
    for task in sorted(all_tasks):
        section = details.split(f"### {task}：", 1)[1].split("\n### T", 1)[0]
        if "**当前状态：" not in section or "**完成条件：" not in section:
            raise AssertionError(f"{task} needs an explicit status and completion requirement")
        if len(section.strip()) < 180:
            raise AssertionError(f"{task} has insufficient handoff detail")
    for name, contents in (
        ("README", readme), ("Unified ledger", main),
    ):
        for reference in ("HANDOFF_SNAPSHOT_2026-11-09.md", "TASK_VERIFICATION_RECORD.md"):
            if reference not in contents:
                raise AssertionError(f"{name} does not link to {reference}")
    for doc in (main, details, snapshot):
        if "282" not in doc or "草稿" not in doc or "正式" not in doc:
            raise AssertionError("Version or release approval status is incomplete")
    if len(re.findall(r"^\| ", matrix, flags=re.MULTILINE)) < 15:
        raise AssertionError("Feature inventory is incomplete")
    print("Verified 20 features, 22 task rows, 22 detailed records and handoff links.")

if __name__ == "__main__":
    try:
        check()
    except (AssertionError, ValueError, IndexError) as error:
        print(f"Documentation verification did not pass: {error}", file=sys.stderr)
        sys.exit(1)
