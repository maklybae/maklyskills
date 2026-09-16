#!/usr/bin/env python3
"""Deterministic pre-scan for slop signals in test files.

Emits candidate signals, not verdicts. The scanner cannot see the one fact that
overrides everything else -- whether a test is the last coverage of a real
branch -- so every hit must be confirmed by reading, and by the falsification
probe when the case is arguable.

Only test files are scanned (*_test.go, test_*.py, *_test.py, *.test.ts,
*.spec.js, *Test.java, *_spec.rb and friends); everything else is skipped.

Usage:
    python3 scan_tests.py FILE [FILE ...]
    python3 scan_tests.py DIR              # walks, skipping vendor/.git/etc.
    git diff --name-only | python3 scan_tests.py --stdin-list
    python3 scan_tests.py FILE --json
"""

import json
import os
import re
import sys

MAX_BYTES = 2_000_000
MAX_SAMPLES = 6
SKIP_DIRS = {
    ".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build",
    "__pycache__", ".venv", "venv", ".mypy_cache", ".pytest_cache", "target",
}
TEST_FILE = re.compile(
    r"(?:_test\.(?:go|py|exs?|rb)$|^test_[^/]*\.py$|\.(?:test|spec)\.(?:t|j)sx?$"
    r"|Test\.java$|Tests?\.cs$|_spec\.rb$|_test\.ts$)"
)

BLOCK_START = re.compile(
    r"^(\s*)(?:"
    r"func\s+(?:Test|Benchmark|Fuzz|Example)\w*\s*\("
    r"|(?:async\s+)?def\s+(test_\w*)\s*\("
    r"|(?:it|test)(?:\.\w+)?\s*\(\s*[\"'`]"
    r"|\[(?:Test|Fact|Theory)\]"
    r")"
)
BLOCK_NAME = re.compile(r"func\s+(\w+)|def\s+(\w+)|(?:it|test)(?:\.\w+)?\s*\(\s*[\"'`]([^\"'`]*)")

ASSERTION = re.compile(
    r"\b(?:require|assert|check|must)\.\w+\s*\("
    r"|\bt\.(?:Error|Fatal|Errorf|Fatalf)\b"
    r"|\bexpect\s*\(|\bExpect\s*\("
    r"|\bself\.assert\w*\s*\("
    r"|^\s*assert\b"
    r"|\.should\b|\bshould\.\w+\("
)
INTERACTION_VERIFY = re.compile(
    r"\bAssertExpectations\b|\bAssertCalled\b|\bAssertNotCalled\b"
    r"|\bAssertNumberOfCalls\b|\bMinTimes\b|\bassert_called\w*\b"
    r"|\bassert_any_call\b|\bassert_has_calls\b|\bassert_not_called\b"
    r"|\btoHaveBeenCalled\w*\b|\bverify\s*\(|\bVerify\s*\(\s*\)"
)
MOCK_SETUP = re.compile(
    r"\.(?:Return|Returns|ReturnsOnCall|AndReturn)\s*\(|\breturn_value\s*=|\bside_effect\s*="
    r"|\bmockReturnValue\w*\s*\(|\bmockResolvedValue\w*\s*\(|\bthenReturn\s*\("
)
LITERAL = re.compile(r"\"([^\"\\]{2,60})\"|\b(\d{2,})\b")

ERROR_STRING_MATCH = re.compile(
    r"\bEqualError\b|\berr\.Error\(\)\s*(?:==|!=)|Contains\([^)]*\.Error\(\)"
    r"|\bErrorContains\b|str\(\s*\w*(?:err|exc|exception)\w*\s*\)\s*==|\.message\s*(?:===?|toBe)"
)
DEEP_EQUAL = re.compile(r"\breflect\.DeepEqual\b")
SLEEP = re.compile(
    r"\btime\.Sleep\s*\(|\bthread\.sleep\b|\btime\.sleep\s*\(|\bsetTimeout\s*\(|\bdelay\s*\("
)
SKIPPED = re.compile(
    r"\bt\.Skip(?:Now|f)?\s*\(|@pytest\.mark\.skip|\bpytest\.skip\s*\(|\b(?:it|test|describe)\.skip\b"
    r"|@unittest\.skip|\bt\.SkipUnless\b|\[Ignore"
)
LOG_ASSERT = re.compile(
    r"\b(?:logBuf\w*|logOutput|logHook|logRecords?|logLines?|logEntries|caplog|"
    r"observedLogs|logSink|logCapture)\b"
)
DELEGATES_TO_HELPER = re.compile(r"\b[A-Za-z_]\w*(?:\.[A-Za-z_]\w*)?\s*\(\s*t\s*[,)]")
ROULETTE_THRESHOLD = 10
TABLE_MIN_ROWS = 4
TRIVIAL_WANT = {"true", "false", "nil", "none", "0", "1", "-1", '""', "''", "[]", "{}"}
DURATION_LITERAL = re.compile(r"\d+\s*\*\s*time\.|\btime\.(?:Duration|Millisecond|Second|Minute|Hour)\b")
TRIVIAL_PREFIXES = ("new", "get", "set", "string", "constructor", "getter", "setter")
TRIVIAL_MAX_LINES = 8
STRING_LITERAL = re.compile(r"\"(?:[^\"\\]|\\.)*\"|'(?:[^'\\]|\\.)*'|`[^`]*`")
NIL_ONLY_ASSERT = re.compile(
    r"NotNil|NotNone|is not None|NoError|toBeDefined|toBeTruthy|assertIsNotNone"
)
FIELD_ACCESS = re.compile(r"\b([a-z]\w*)\.([A-Za-z_]\w{1,})\b")
RECEIVER_STOPLIST = {
    "require", "assert", "check", "must", "expect", "self", "t", "s", "b", "f",
    "err", "cmp", "cmpopts", "json", "time", "fmt", "os", "np", "pd", "mock",
    "ctx", "tc", "tt", "test", "it", "describe", "reflect", "errors", "strings",
    "http", "context", "sql", "proto", "protocmp", "gomock", "sinon", "jest",
}
TABLE_WANT = re.compile(r"\b(?:want|wantErr|expected|expect|exp)\s*:\s*([^,}\n]+)")
BARE_NUMBER_IN_ASSERT = re.compile(r"(?<![\w.\"'])(-?\d{2,}(?:\.\d+)?)(?![\w.\"'])")


class Findings:
    def __init__(self):
        self.buckets = {}
        self.tests = 0

    def add(self, category, confidence, path, lineno, text):
        key = (category, confidence)
        entry = self.buckets.setdefault(key, {"count": 0, "samples": []})
        entry["count"] += 1
        if len(entry["samples"]) < MAX_SAMPLES:
            entry["samples"].append(f"{path}:{lineno}: {text.strip()[:140]}")


def is_test_file(path):
    return bool(TEST_FILE.search(os.path.basename(path)))


def block_name(line):
    m = BLOCK_NAME.search(line)
    if not m:
        return "?"
    return next((g for g in m.groups() if g), "?")


def split_blocks(lines):
    starts = [i for i, ln in enumerate(lines) if BLOCK_START.match(ln)]
    for n, start in enumerate(starts):
        end = starts[n + 1] if n + 1 < len(starts) else len(lines)
        yield start, lines[start:end]


def code_only(block):
    out = []
    for ln in block:
        s = ln.strip()
        if s.startswith("//") or s.startswith("#") or s.startswith("*"):
            continue
        out.append(ln)
    return out


def names_a_trivial_target(name):
    """True when the test name points at a constructor, getter, setter or String()."""
    stem = re.sub(r"^(?:Test|Benchmark|test_|it_)", "", name)
    candidates = [stem] + stem.split("_")[1:]
    return any(c.lower().startswith(TRIVIAL_PREFIXES) for c in candidates if c)


def receivers_in(line):
    """Field-access receivers, with string literals removed so URLs and emails do not count."""
    bare = STRING_LITERAL.sub("", line)
    seen = []
    for receiver, _ in FIELD_ACCESS.findall(bare):
        if receiver not in RECEIVER_STOPLIST and receiver not in seen:
            seen.append(receiver)
    return seen


def scan_field_runs(f, path, offset, block):
    run_receiver, run_len, run_start = None, 0, 0
    for i, ln in enumerate(block):
        if not ASSERTION.search(ln):
            run_receiver, run_len = None, 0
            continue
        receivers = receivers_in(ln)
        if run_receiver is not None and run_receiver in receivers:
            run_len += 1
            if run_len == 4:
                f.add("assert/per-field-run", "medium", path, offset + run_start + 1,
                      f"4+ consecutive assertions on `{run_receiver}` fields; "
                      "compare the struct once with a diff")
        elif receivers:
            run_receiver, run_len, run_start = receivers[0], 1, i
        else:
            run_receiver, run_len = None, 0


def scan_table_rows(f, path, offset, block):
    wants = {}
    for i, ln in enumerate(block):
        for m in TABLE_WANT.finditer(ln):
            value = m.group(1).strip().rstrip(",")
            if len(value) > 60:
                continue
            wants.setdefault(value, []).append(i)
    for value, hits in wants.items():
        if len(hits) >= TABLE_MIN_ROWS and value.lower() not in TRIVIAL_WANT:
            f.add("table/rows-share-one-outcome", "low", path, offset + hits[0] + 1,
                  f"{len(hits)} rows expect `{value}`; check they do not all walk the same branch")


def scan_block(f, path, offset, block):
    f.tests += 1
    name = block_name(block[0])
    body = code_only(block[1:])
    joined = "\n".join(body)
    header_line = offset + 1

    asserts = [ln for ln in body if ASSERTION.search(ln)]
    interactions = [ln for ln in body if INTERACTION_VERIFY.search(ln)]
    state_asserts = [ln for ln in asserts if not INTERACTION_VERIFY.search(ln)]

    if SKIPPED.search(joined):
        f.add("flag/skipped-test", "high", path, header_line,
              f"{name} is skipped; flag it, do not delete silently")

    delegates = any(DELEGATES_TO_HELPER.search(ln) for ln in body)
    if not asserts and not interactions and not delegates:
        f.add("test/no-assertion", "high", path, header_line,
              f"{name} asserts nothing; passes unless something panics")
    elif interactions and not state_asserts:
        f.add("test/interaction-only", "high", path, header_line,
              f"{name} verifies calls but asserts no result")

    setups = [ln for ln in body if MOCK_SETUP.search(ln)]
    if setups and state_asserts:
        stubbed = set()
        for ln in setups:
            for a, b in LITERAL.findall(ln):
                stubbed.add(a or b)
        echoed = set()
        for ln in state_asserts:
            echoed |= {a or b for a, b in LITERAL.findall(ln)} & stubbed
        if echoed:
            f.add("test/mock-echo", "medium", path, header_line,
                  f"{name} asserts {sorted(echoed)[:3]} which a double was told to return")

    trivial_shape = (
        names_a_trivial_target(name)
        and state_asserts
        and not setups
        and len([ln for ln in body if ln.strip()]) <= TRIVIAL_MAX_LINES
        and all(NIL_ONLY_ASSERT.search(ln) or receivers_in(ln) for ln in state_asserts)
    )
    if trivial_shape:
        f.add("test/constructor-or-getter", "medium", path, header_line,
              f"{name} may be asserting that assignment assigns")

    if len(asserts) >= ROULETTE_THRESHOLD:
        f.add("assert/roulette", "low", path, header_line,
              f"{name} has {len(asserts)} assertions; one test, one behaviour")

    # A repeated NoError/NotNil is one step of setup per call, not a duplicate assert.
    seen = set()
    for ln in asserts:
        if NIL_ONLY_ASSERT.search(ln):
            continue
        key = re.sub(r"\s+", " ", ln.strip())
        if key in seen:
            f.add("assert/duplicate", "medium", path, header_line,
                  f"{name} repeats `{key[:80]}`")
            break
        seen.add(key)

    for i, ln in enumerate(body):
        lineno = offset + i + 2
        if ERROR_STRING_MATCH.search(ln):
            f.add("assert/error-string-match", "high", path, lineno, ln)
        if DEEP_EQUAL.search(ln):
            f.add("assert/reflect-deepequal", "medium", path, lineno, ln)
        if SLEEP.search(ln):
            f.add("flag/sleep-as-synchronisation", "medium", path, lineno, ln)
        if ASSERTION.search(ln):
            if LOG_ASSERT.search(ln):
                f.add("assert/log-output", "medium", path, lineno, ln)
            bare = STRING_LITERAL.sub("", ln)
            if BARE_NUMBER_IN_ASSERT.search(bare) and not DURATION_LITERAL.search(bare):
                f.add("assert/magic-number", "low", path, lineno, ln)

    scan_field_runs(f, path, offset, body)
    scan_table_rows(f, path, offset, body)


def scan_text(f, path, text):
    lines = text.splitlines()
    for start, block in split_blocks(lines):
        scan_block(f, path, start, block)


def iter_files(paths):
    for p in paths:
        if os.path.isdir(p):
            for root, dirs, files in os.walk(p):
                dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
                for name in files:
                    yield os.path.join(root, name)
        else:
            yield p


def readable(path):
    try:
        if os.path.getsize(path) > MAX_BYTES:
            return None
        with open(path, "r", encoding="utf-8", errors="strict") as fh:
            return fh.read()
    except (OSError, UnicodeDecodeError):
        return None


def render_text(f, scanned):
    order = {"high": 0, "medium": 1, "low": 2}
    keys = sorted(f.buckets, key=lambda k: (order.get(k[1], 3), k[0]))
    if not keys:
        return (f"TEST PRE-SCAN: {f.tests} test(s) in {scanned} file(s), no deterministic signals. "
                "Read them anyway and name the bug each one guards against.")
    out = ["TEST PRE-SCAN -- candidate signals, not verdicts. Confirm each by reading.\n"]
    total = 0
    for (category, confidence) in keys:
        entry = f.buckets[(category, confidence)]
        total += entry["count"]
        out.append(f"[{confidence}] {category}: {entry['count']}")
        for s in entry["samples"]:
            out.append(f"    {s}")
        if entry["count"] > len(entry["samples"]):
            out.append(f"    ... +{entry['count'] - len(entry['samples'])} more")
    out.append("")
    out.append(f"Total candidate signals: {total} across {f.tests} test(s) in {scanned} file(s).")
    out.append(
        "The rule: a test earns its place if it would fail when a plausible bug is introduced "
        "into the behaviour it names. For an arguable case, prove it -- mutate the production "
        "line, run that one test, revert immediately."
    )
    out.append(
        "Signals prefixed flag/ are findings for a human, not edits. A test that is the LAST "
        "coverage of a real branch gets rewritten, never deleted -- and the scanner cannot tell."
    )
    return "\n".join(out)


def render_json(f):
    payload = []
    for (category, confidence), entry in f.buckets.items():
        payload.append({
            "category": category,
            "confidence": confidence,
            "count": entry["count"],
            "samples": entry["samples"],
        })
    return json.dumps({"tests": f.tests, "signals": payload}, ensure_ascii=False, indent=2)


def main(argv):
    args = [a for a in argv if not a.startswith("--")]
    flags = {a for a in argv if a.startswith("--")}
    if "--stdin-list" in flags:
        args = [ln.strip() for ln in sys.stdin if ln.strip()] + args
    if not args:
        print(__doc__)
        return 1

    f = Findings()
    scanned = 0
    for path in iter_files(args):
        if not is_test_file(path):
            continue
        text = readable(path)
        if text is None:
            continue
        scan_text(f, path, text)
        scanned += 1

    if not scanned:
        print("TEST PRE-SCAN: no test files matched. Check the paths, or scan the package directory.")
        return 0

    if "--json" in flags:
        print(render_json(f))
    else:
        print(render_text(f, scanned))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
