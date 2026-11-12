#!/usr/bin/env python3
"""Include the complete DB-IP City Lite MMDB in application distributions."""

import argparse
import gzip
import hashlib
import os
from pathlib import Path
import shutil
import tempfile
import urllib.request

DB_MONTH = "2026-11"
DB_URL = f"https://download.db-ip.com/free/dbip-city-lite-{DB_MONTH}.mmdb.gz"
DB_FILENAME = "dbip-city-lite.mmdb"
MIN_SIZE = 50 * 1024 * 1024
MAX_SIZE = 300 * 1024 * 1024
MMDB_MARKER = b"\xab\xcd\xefMaxMind.com"


def verify_database(path: Path) -> tuple[int, str]:
    size = path.stat().st_size
    if not MIN_SIZE <= size <= MAX_SIZE:
        raise ValueError(f"Unexpected database file size: {size} bytes")
    with path.open("rb") as handle:
        handle.seek(-4096, os.SEEK_END)
        if MMDB_MARKER not in handle.read():
            raise ValueError("The downloaded file is not a complete MMDB database")
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return size, digest.hexdigest()


def install_database(destination: Path) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    if destination.is_file():
        size, checksum = verify_database(destination)
        print(f"GeoIP database already available: {size} bytes; SHA-256: {checksum}")
        return
    with tempfile.TemporaryDirectory(prefix="rustdesk-geoip-") as tmp:
        compressed = Path(tmp) / "source.mmdb.gz"
        extracted = Path(tmp) / DB_FILENAME
        with urllib.request.urlopen(
            urllib.request.Request(DB_URL, headers={"User-Agent": "RustDesk-Control-Center/1.0"}),
            timeout=120,
        ) as response:
            with compressed.open("wb") as target:
                shutil.copyfileobj(response, target, length=1024 * 1024)
        with gzip.open(compressed, "rb") as source, extracted.open("wb") as target:
            shutil.copyfileobj(source, target, length=1024 * 1024)
        size, checksum = verify_database(extracted)
        os.replace(extracted, destination)
        print(f"DB-IP City Lite {DB_MONTH}: {size} bytes; SHA-256: {checksum}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Package the DB-IP City Lite MMDB")
    parser.add_argument("--output", type=Path, default=Path("resources/geoip") / DB_FILENAME)
    parser.add_argument("--verify", action="store_true", help="Verify an existing database only")
    args = parser.parse_args()
    if args.verify:
        size, checksum = verify_database(args.output)
        print(f"Verified GeoIP database: {size} bytes; SHA-256: {checksum}")
    else:
        install_database(args.output)


if __name__ == "__main__":
    main()
