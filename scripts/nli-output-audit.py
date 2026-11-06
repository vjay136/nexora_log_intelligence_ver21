#!/usr/bin/env python3
"""Read-only JSONL record accounting. Never clears or rewrites NLI output files."""
import argparse
import glob
import hashlib
import json
import os
from pathlib import Path
import sys

MAX_LINE = 16 * 1024 * 1024


def scan(path, role, previous=None):
    counts = dict(physical_lines=0, newline_count=0, nonblank_lines=0,
                  json_records=0, invalid_json_lines=0, oversized_lines=0)
    digest, prefix = hashlib.sha256(), hashlib.sha256()
    warnings = []
    with open(path, "rb") as stream:
        stat = os.fstat(stream.fileno())
        remaining, consumed = stat.st_size, 0
        previous_size = previous.get("bytes", 0) if previous else 0
        terminated = True
        while remaining:
            row = stream.readline(min(remaining, MAX_LINE + 1))
            if not row:
                warnings.append("File changed while counting; take another snapshot at EOF.")
                break
            oversized = len(row) > MAX_LINE
            head = row
            while True:
                digest.update(row)
                if consumed < previous_size:
                    prefix.update(row[:previous_size-consumed])
                consumed += len(row)
                remaining -= len(row)
                terminated = row.endswith(b"\n")
                if terminated:
                    counts["newline_count"] += 1
                if terminated or not remaining or not oversized:
                    break
                row = stream.readline(min(remaining, MAX_LINE + 1))
                if not row:
                    warnings.append("File changed while counting; take another snapshot at EOF.")
                    remaining = 0
                    break
            counts["physical_lines"] += 1
            if not head.strip():
                continue
            counts["nonblank_lines"] += 1
            if oversized:
                counts["oversized_lines"] += 1
                continue
            if role != "input":
                try:
                    value = json.loads(head)
                    if not isinstance(value, dict):
                        raise ValueError("JSONL event must be an object")
                    counts["json_records"] += 1
                except (ValueError, UnicodeError, RecursionError):
                    counts["invalid_json_lines"] += 1
        if consumed and not terminated:
            warnings.append("Final line has no newline; it may still be in flight.")
        result = dict(role=role, bytes=consumed, device=stat.st_dev, inode=stat.st_ino,
                      sha256=digest.hexdigest(), terminated=terminated, **counts)
        if previous:
            unchanged_prefix = (stat.st_dev == previous["device"] and
                                stat.st_ino == previous["inode"] and
                                consumed >= previous["bytes"] and
                                prefix.hexdigest() == previous["sha256"] and
                                previous.get("terminated", False))
            if unchanged_prefix:
                result["delta"] = {key: counts[key]-previous[key] for key in counts}
            else:
                warnings.append("Baseline file was replaced, truncated, edited or incomplete; delta is unknown.")
                result["delta"] = None
    result["warnings"] = warnings
    return result


def run(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("input", "normalised", "quarantine"):
        parser.add_argument("--"+name, nargs="+", default=[], metavar="PATH_OR_GLOB")
    parser.add_argument("--snapshot", help="Write count-only baseline JSON to a separate file")
    parser.add_argument("--baseline", help="Subtract previous output counts, preserving earlier hourly history")
    parser.add_argument("--single-line", action="store_true", help="Explicitly assume one nonblank input line per logical record")
    parser.add_argument("--expected-records", type=int, help="Known logical input count for this controlled test, including filtered records")
    parser.add_argument("--filtered-records", type=int, default=0, help="Known intentionally filtered logical records (default 0)")
    args = parser.parse_args(argv)
    if args.filtered_records < 0 or args.expected_records is not None and args.expected_records < 0:
        parser.error("record counts cannot be negative")
    old = {}
    if args.baseline:
        baseline = json.loads(Path(args.baseline).read_text(encoding="utf-8-sig"))
        if baseline.get("schema_version") != 1:
            parser.error("unsupported baseline schema")
        old = baseline["files"]
    if args.single_line and not args.input and args.expected_records is None:
        parser.error("--single-line requires --input or an explicit --expected-records")
    selected, warnings, notes = {}, [], []
    for role in ("input", "normalised", "quarantine"):
        for pattern in getattr(args, role):
            matches = glob.glob(pattern)
            if not matches:
                if role == "input":
                    parser.error("no input files match: " + pattern)
                notes.append("No existing output files match (zero records): " + pattern)
            for name in matches:
                path = str(Path(name).resolve())
                if path in selected and selected[path] != role:
                    parser.error("one file cannot be counted in multiple roles: " + path)
                selected[path] = role
    if not (args.input or args.normalised or args.quarantine):
        parser.error("select at least one input or output path/pattern")
    files = {}
    for path, role in sorted(selected.items()):
        previous = old.get(path)
        if previous and previous["role"] != role:
            parser.error("baseline role changed: " + path)
        result = scan(path, role, previous)
        if args.baseline and previous is None:
            result["delta"] = {key: result[key] for key in ("physical_lines", "newline_count", "nonblank_lines", "json_records", "invalid_json_lines", "oversized_lines")}
        files[path] = result
        warnings.extend(path + ": " + warning for warning in result["warnings"])
        if role != "input" and (result["invalid_json_lines"] or result["oversized_lines"] or result["physical_lines"] != result["nonblank_lines"]):
            warnings.append(path + ": output contains blank, invalid or oversized JSONL rows; inspect before comparing.")
    for path, previous in old.items():
        if previous["role"] != "input" and path not in files:
            warnings.append("Baseline output file is missing from current selection; include rotated files: " + path)
    totals = {role: sum(row["nonblank_lines" if role == "input" else "json_records"] for row in files.values() if row["role"] == role) for role in ("input", "normalised", "quarantine")}
    delta = None
    if args.baseline:
        delta = {role: sum(row["delta"]["json_records"] for row in files.values() if row["role"] == role and row.get("delta") is not None) for role in ("normalised", "quarantine")}
    expected = args.expected_records
    if expected is None and args.single_line:
        expected = totals["input"]
    observed = delta if delta is not None else totals
    actual = observed["normalised"] + observed["quarantine"] + args.filtered_records
    # Snapshot counts cannot prove delivery acknowledgement or read offsets.
    verdict = "not_compared" if expected is None else "unknown" if warnings else "balanced" if actual == expected else "mismatch"
    report = dict(schema_version=1, read_only=True, files=files, totals=totals,
                  output_delta=delta, expected_logical_records=expected,
                  observed_output_plus_filtered=actual, comparison=verdict, warnings=warnings, notes=notes,
                  scope="Selected local files at scan time; wait for EOF/drain. Hourly files append history. Replay, fan-out and multiline framing require an explicit test scope. This does not prove Loki acknowledgement.")
    if args.snapshot:
        target = Path(args.snapshot).resolve()
        if str(target) in selected or target.exists() and any(target.samefile(path) for path in selected):
            parser.error("baseline destination must not overwrite an input/output file")
        target.write_text(json.dumps(dict(schema_version=1, files=files), indent=2)+"\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    return 1 if verdict in ("mismatch", "unknown") else 0


if __name__ == "__main__":
    try:
        sys.exit(run())
    except (OSError, ValueError, KeyError, TypeError) as error:
        print("Audit failed: " + str(error), file=sys.stderr)
        sys.exit(2)
