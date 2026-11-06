"""Exercise the shipped read-only record audit with historical and new output."""
import importlib.util
import io
import json
import sys
from contextlib import redirect_stdout
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("output_audit", ROOT / "scripts/nli-output-audit.py")
audit = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit)


class OutputAccounting(unittest.TestCase):
    def invoke(self, *args):
        output = io.StringIO()
        with redirect_stdout(output):
            code = audit.run(list(map(str, args)))
        return code, json.loads(output.getvalue())

    def test_49_records_do_not_include_earlier_hourly_history(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            raw, normal, quarantine, before = [root / name for name in ("sample.log", "normal.jsonl", "quarantine.jsonl", "before.json")]
            raw.write_text("same legitimate source line\n" * 49)
            # Existing hourly history must stay byte-for-byte intact.
            history = b'{"old":true}\n' * 12
            normal.write_bytes(history)
            quarantine.write_text('{"old_failure":true}\n')
            self.invoke("--normalised", normal, "--quarantine", quarantine, "--snapshot", before)
            with normal.open("ab") as stream:
                stream.write(b'{"message":"identical legitimate value"}\n' * 47)
            with quarantine.open("ab") as stream:
                stream.write(b'{"raw":"failed source occurrence"}\n' * 2)
            code, report = self.invoke("--input", raw, "--normalised", normal, "--quarantine", quarantine, "--baseline", before, "--single-line")
            self.assertEqual(code, 0)
            self.assertEqual(report["comparison"], "balanced")
            self.assertEqual(report["output_delta"], {"normalised": 47, "quarantine": 2})
            self.assertEqual(report["expected_logical_records"], 49)
            self.assertTrue(normal.read_bytes().startswith(history))
            self.assertEqual(report["totals"]["normalised"], 59)

    def test_extra_record_is_reported_and_no_content_deduplication_occurs(self):
        with tempfile.TemporaryDirectory() as folder:
            normal = Path(folder) / "normal.jsonl"
            payload = b'{"message":"same"}\n' * 50
            normal.write_bytes(payload)
            code, report = self.invoke("--normalised", normal, "--expected-records", 49)
            self.assertEqual(code, 1)
            self.assertEqual(report["comparison"], "mismatch")
            self.assertEqual(report["totals"]["normalised"], 50)
            self.assertEqual(normal.read_bytes(), payload)

    def test_truncation_and_inflight_or_invalid_rows_never_claim_balance(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            normal, baseline = root / "normal.jsonl", root / "before.json"
            normal.write_text('{"old":true}\n' * 4)
            self.invoke("--normalised", normal, "--snapshot", baseline)
            normal.write_text('{"new":true}\n')
            code, report = self.invoke("--normalised", normal, "--baseline", baseline, "--expected-records", 1)
            self.assertEqual((code, report["comparison"]), (1, "unknown"))
            for content in ('{"new":true}', '{"new":true}\ninvalid\n', '{"new":true}\n\n'):
                normal.write_text(content)
                code, report = self.invoke("--normalised", normal, "--expected-records", 1)
                self.assertEqual((code, report["comparison"]), (1, "unknown"))

    def test_lazy_empty_quarantine_file_is_zero_not_an_error(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            normal = root / "normal.jsonl"
            normal.write_text('{"ok":true}\n' * 49)
            code, report = self.invoke("--normalised", normal, "--quarantine", root / "missing*.jsonl", "--expected-records", 49)
            self.assertEqual((code, report["comparison"]), (0, "balanced"))

    def test_baseline_before_lazy_output_creation_then_49_records(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            pattern, baseline = root / "normal*.jsonl", root / "before.json"
            code, report = self.invoke("--normalised", pattern, "--snapshot", baseline)
            self.assertEqual(code, 0)
            self.assertEqual(report["totals"]["normalised"], 0)
            (root / "normal-hour.jsonl").write_text('{"ok":true}\n' * 49)
            code, report = self.invoke("--normalised", pattern, "--baseline", baseline, "--expected-records", 49)
            self.assertEqual((code, report["comparison"]), (0, "balanced"))


if __name__ == "__main__":
    unittest.main(argv=[sys.argv[0]])
