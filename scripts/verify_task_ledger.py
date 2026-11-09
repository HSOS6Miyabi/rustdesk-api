#!/usr/bin/env python3
"""Validate that the two-repository implementation ledger is complete.

This check only reads repository files and never changes release state.
"""
from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
MAIN = ROOT / "docs" / "IMPLEMENTATION_LEDGER.md"
DETAIL = ROOT / "docs" / "TASK_VERIFICATION_RECORD.md"
MATRIX = ROOT / "docs" / "FEATURE_MATRIX.md"
README = ROOT / "README.md"
REQUIRED = (
    ROOT / "docs" / "API.md",
    ROOT / "docs" / "TESTING.md",
    ROOT / "docs" / "CLIENT_BUILDER.md",
    ROOT / "docs" / "DATABASE_BACKUP_RESTORE.md",
    ROOT / "docs" / "UPGRADE.md",
    ROOT / "docs" / "UPSTREAM_FEATURE_WATCH.md",
)


def unique_ids(pattern, source, expected, name):
    matches = re.findall(pattern, source, re.MULTILINE)
    if len(matches) != len(expected) or set(matches) != expected:
        missing = sorted(expected - set(matches))
        unexpected = sorted(set(matches) - expected)
        duplicates = sorted({item for item in matches if matches.count(item) > 1})
        raise AssertionError(
            f"{name}: missing={missing}, unexpected={unexpected}, duplicates={duplicates}"
        )


def main():
    for filepath in (MAIN, DETAIL, MATRIX, README, *REQUIRED):
        if not filepath.is_file():
            raise AssertionError(f"Required project documentation is missing: {filepath}")
    main_doc = MAIN.read_text(encoding="utf-8")
    detail = DETAIL.read_text(encoding="utf-8")
    matrix = MATRIX.read_text(encoding="utf-8")
    readme = README.read_text(encoding="utf-8")
    tasks = {f"T{number:03d}" for number in range(1, 23)}
    features = {f"F{number:02d}" for number in range(1, 21)}
    table = main_doc.split("### 4. 完整待办事项及处理顺序", 1)[1].split(
        "### 5. 重要实现细节", 1
    )[0]
    feature_table = main_doc.split("### 3. 功能实施状态", 1)[1].split(
        "### 4. 完整待办事项及处理顺序", 1
    )[0]
    unique_ids(r"^\\| (T\\d{3}) \\|", table, tasks, "Primary work ledger")
    unique_ids(r"^\\| (F\\d{2}) \\|", feature_table, features, "Feature register")
    unique_ids(r"^### (T\\d{3})：", detail, tasks, "Detailed task register")
    for task in sorted(tasks):
        section = detail.split(f"### {task}：", 1)[1].split("\n### T", 1)[0]
        if "**当前状态：" not in section or "**完成条件：" not in section:
            raise AssertionError(f"{task}: status or completion requirements missing")
        if len(section.strip()) < 180:
            raise AssertionError(f"{task}: implementation and testing detail is too short")
    for name, source in (("README", readme), ("Primary ledger", main_doc)):
        if "TASK_VERIFICATION_RECORD.md" not in source:
            raise AssertionError(f"{name} must link to the complete task register")
    if "282" not in main_doc or "282" not in detail:
        raise AssertionError("The current database migration version must be documented")
    for label, text in (
        ("Draft pull request", "草稿"),
        ("Publishing restrictions", "正式发布"),
        ("Automated checks", "自动化"),
        ("Real environments", "实际环境"),
    ):
        if text not in main_doc or text not in detail:
            raise AssertionError(f"{label} not recorded in both ledgers")
    for name in (
        "客户端安装文件",
        "数据库",
        "邮件",
        "中继",
        "权限",
        "审计",
    ):
        if name not in detail:
            raise AssertionError(f"Detailed register omits feature area: {name}")
    if len(re.findall(r"^\\| ", matrix, re.MULTILINE)) < 15:
        raise AssertionError("The feature matrix has too few entries")
    print("Project ledger verified: 20 features and all 22 tasks have status and completion details.")
    print("Cross-repository references and release restrictions are recorded.")


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, ValueError, IndexError) as exc:
        print(f"Project ledger validation failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
