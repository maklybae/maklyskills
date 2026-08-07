#!/usr/bin/env python3
"""Deterministic pre-scan for AI-slop signals in source code.

Emits candidate signals, not verdicts. Every hit must be confirmed with
judgment against the surrounding context before acting on it. The scan is
language-agnostic: it works on comment text and raw source patterns that hold
across languages, and deliberately stays silent on things that need semantic
understanding (over-abstraction, wrong abstractions, happy-path logic).

Usage:
    python3 scan_slop.py FILE [FILE ...]
    python3 scan_slop.py DIR              # walks, skipping vendor/.git/etc.
    git diff --name-only | python3 scan_slop.py --stdin-list
    python3 scan_slop.py FILE --json
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
SKIP_EXT = {
    ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".pdf", ".zip", ".gz",
    ".tar", ".woff", ".woff2", ".ttf", ".eot", ".mp4", ".mp3", ".lock", ".svg",
    ".min.js", ".min.css",
}

EMOJI = re.compile(
    "["
    "\U0001F300-\U0001FAFF"
    "\U00002600-\U000027BF"
    "\U0001F000-\U0001F0FF"
    "\U00002190-\U000021FF"
    "\U00002B00-\U00002BFF"
    "\U0000FE00-\U0000FE0F"
    "\U0001F1E6-\U0001F1FF"
    "\U00002700-\U000027BF"
    "]"
)

# Escapes, not literals: a literal NBSP reads as a plain space and matches everything.
UNICODE_HAZARDS = {
    "\u2014": "em-dash",
    "\u2013": "en-dash",
    "\u2018": "smart-quote",
    "\u2019": "smart-quote",
    "\u201c": "smart-quote",
    "\u201d": "smart-quote",
    "\u2026": "ellipsis-char",
    "\u00a0": "non-breaking-space",
    "\u200b": "zero-width-space",
    "\ufeff": "byte-order-mark",
}

COMMENT_TEXT = re.compile(r"(?:^|[^:/])(?://|#(?!!)|--(?!\S)|;;?)\s?(.*)$")
BLOCK_MARK = re.compile(r"/\*+|\*/|<!--|-->|\"\"\"|'''")

# JSDoc continuation line; the required space keeps C dereferences (*ptr) out.
BLOCK_CONT = re.compile(r"^\s*\*(?!/)\s+(.*)$")

COMMENT_LINE_START = re.compile(r"^(//|#|--|;|\*|/\*|\*/|\"\"\"|'''|<!--|-->)")
CPP_DIRECTIVE = re.compile(
    r"^#\s*(include|define|ifn?def|ifdef|endif|pragma|if|else|elif|undef|error|line|import)\b"
)

# One human-readable comment line is the cap; anything longer is a defect.
MAX_COMMENT_RUN = 1

# Blocks that are multi-line by mandate are exempt from the cap.
LICENSE_MARK = re.compile(
    r"SPDX-License-Identifier|copyright|licen[cs]ed under|all rights reserved|"
    r"code generated|do not edit|@generated",
    re.IGNORECASE,
)
DIRECTIVE_COMMENT = re.compile(
    r"^\s*(?://go:|//\s*\+build|//\s*nolint|//\s*eslint|//\s*@ts-|/\*\s*eslint|"
    r"#\s*type:\s*ignore|#\s*noqa|#\s*pylint:|#\s*mypy:|#\s*ruff:|#\s*fmt:|"
    r"(?://|#|\*)\s*Deprecated:)",
    re.IGNORECASE,
)

# A comment run terminated by one of these is documenting a public symbol.
EXPORTED_DECL = re.compile(
    r"^\s*(?:"
    r"func\s+(?:\([^)]*\)\s*)?[A-Z]"
    r"|(?:type|var|const)\s+[A-Z]"
    r"|export\s+(?:default\s+)?(?:async\s+)?"
    r"(?:function|class|const|let|var|interface|type|enum|abstract)\b"
    r"|pub(?:\([^)]*\))?\s+(?:fn|struct|enum|trait|mod|const|type|unsafe|async)\b"
    r"|(?:public|protected)\s+"
    r")"
)

PY_DEF = re.compile(r"^\s*(?:async\s+)?(?:def|class)\s+([A-Za-z_]\w*)")
DOCSTRING_START = re.compile(r"^[rbuf]*(\"\"\"|''')")

CONVERSATIONAL = re.compile(
    r"\b(here'?s|here is|let'?s|now (?:we|let)|first,|next,|then we|"
    r"as (?:we can see|mentioned|a large language model|an ai)|"
    r"note that|we (?:can|will|need to)|in this (?:function|method|example)|"
    r"the following code|as requested|as an ai)\b",
    re.IGNORECASE,
)

HOLLOW = re.compile(
    r"\b(optimi[sz]ed|efficient|performant|blazing(?:ly)?|lightning[- ]fast|"
    r"seamless(?:ly)?|robust|powerful|flexible|scalable|maintainable|"
    r"production[- ]ready|enterprise[- ]grade|cutting[- ]edge|state[- ]of[- ]the[- ]art|"
    r"world[- ]class|best[- ]practice|industry[- ]standard|"
    r"cinematic|silky|elegant|organic|buttery)\b",
    re.IGNORECASE,
)

PERF_CLAIM = re.compile(r"\b\d+(?:\.\d+)?\s*(?:%|x)\s*(?:faster|slower|better|less|more)\b", re.IGNORECASE)

RESTATEMENT = re.compile(
    r"^\s*(creates?|returns?|increments?|decrements?|loops?|iterates?|"
    r"initiali[sz]es?|inits?|sets? up|checks?|gets?|sets?|handles?|processes?|"
    r"updates?|deletes?|removes?|adds?|calls?|defines?|declares?|imports?|"
    r"instantiates?|assigns?|prints?|logs?)\b",
    re.IGNORECASE,
)

DOC_TAG = re.compile(r"^\s*[@\\](param|returns?|arg|type|throws|example|brief)\b", re.IGNORECASE)

GENERIC_TODO = re.compile(
    r"\b(?:TODO|FIXME|XXX)\b(?!\s*[(\[])[:\s-]*(implement|add|fix|handle|finish|complete|do)?",
    re.IGNORECASE,
)

BANNER = re.compile(r"(?:[=*#\-~_]){8,}")

GENERIC_NAMES = {
    "data", "result", "results", "temp", "tmp", "val", "obj", "item",
    "info", "payload", "res", "ret", "retval", "thing", "things", "stuff",
    "foo", "bar", "baz", "qux",
}
GENERIC_NAME_RE = re.compile(r"\b(" + "|".join(sorted(GENERIC_NAMES)) + r")\b")

DEBUG_PRINT = re.compile(
    r"\b(console\.log|console\.debug|print|println!?|fmt\.Print(?:ln|f)?|"
    r"System\.out\.print(?:ln)?|puts|var_dump|dd)\s*\(",
)

ASSIGN = re.compile(r"^\s*(?:(?:const|let|var|final)\s+)?([A-Za-z_]\w*)\s*(?::?=|=)\s*[^=].*$")
RETURN_VAR = re.compile(r"^\s*return\s+([A-Za-z_]\w*)\s*;?\s*$")


class Findings:
    def __init__(self):
        self.buckets = {}

    def add(self, category, confidence, path, lineno, text):
        key = (category, confidence)
        entry = self.buckets.setdefault(key, {"count": 0, "samples": []})
        entry["count"] += 1
        if len(entry["samples"]) < MAX_SAMPLES:
            entry["samples"].append(f"{path}:{lineno}: {text.strip()[:140]}")


def strip_comment(line):
    m_cont = BLOCK_CONT.match(line)
    if m_cont:
        return m_cont.group(1)
    if BLOCK_MARK.search(line):
        return line
    m = COMMENT_TEXT.search(line)
    return m.group(1) if m else None


def scan_line(f, path, lineno, line, prev_assign):
    if EMOJI.search(line):
        f.add("cosmetic/emoji", "high", path, lineno, line)
    for ch, name in UNICODE_HAZARDS.items():
        if ch in line:
            f.add(f"cosmetic/unicode-{name}", "high", path, lineno, line)
            break
    if BANNER.search(line) and any(m in line for m in ("//", "#", "/*", "*", "--")):
        f.add("comment/banner", "medium", path, lineno, line)

    comment = strip_comment(line)
    if comment is not None and comment.strip():
        c = comment.strip()
        if CONVERSATIONAL.search(c):
            f.add("comment/conversational-artifact", "high", path, lineno, c)
        if HOLLOW.search(c):
            f.add("comment/hollow-claim", "high", path, lineno, c)
        if PERF_CLAIM.search(c):
            f.add("comment/unverified-perf-claim", "high", path, lineno, c)
        if DOC_TAG.search(c):
            f.add("comment/doc-tag-ceremony", "high", path, lineno, c)
        if GENERIC_TODO.search(c):
            f.add("comment/generic-todo", "medium", path, lineno, c)
        if RESTATEMENT.search(c) and len(c.split()) <= 6:
            f.add("comment/restatement", "medium", path, lineno, c)

    if DEBUG_PRINT.search(line) and strip_comment(line) != line:
        f.add("debug/print-statement", "low", path, lineno, line)

    names = set(GENERIC_NAME_RE.findall(line))
    for n in names:
        f.add("naming/generic-token", "low", path, lineno, line)
        break

    m_ret = RETURN_VAR.match(line)
    if m_ret and prev_assign and prev_assign == m_ret.group(1):
        f.add("structure/assign-then-return", "medium", path, lineno, line)

    m_assign = ASSIGN.match(line)
    return m_assign.group(1) if m_assign else None


def is_comment_line(line):
    s = line.strip()
    if not s or s.startswith("#!") or CPP_DIRECTIVE.match(s):
        return False
    return bool(COMMENT_LINE_START.match(s))


def is_machine_mandated(lines):
    return any(LICENSE_MARK.search(ln) or DIRECTIVE_COMMENT.match(ln) for ln in lines)


def flush_comment_run(f, path, start, lines, terminator):
    if not lines or is_machine_mandated(lines):
        return
    if len(lines) > MAX_COMMENT_RUN:
        f.add("comment/block-over-one-line", "high", path, start,
              f"{len(lines)} consecutive comment lines (cap is {MAX_COMMENT_RUN}; condense to one)")
    if terminator is not None and EXPORTED_DECL.match(terminator):
        f.add("comment/public-doc-comment", "high", path, start, lines[0])


def scan_text(f, path, text):
    prev_assign = None
    run_lines = []
    run_start = 0
    pending_def = None
    awaiting_docstring = None

    for i, line in enumerate(text.splitlines(), start=1):
        stripped = line.strip()

        if is_comment_line(line):
            if not run_lines:
                run_start = i
            run_lines.append(line)
        else:
            flush_comment_run(f, path, run_start, run_lines, line)
            run_lines = []

        if awaiting_docstring is not None and stripped:
            if awaiting_docstring == "public" and DOCSTRING_START.match(stripped):
                f.add("comment/public-doc-comment", "high", path, i, stripped)
            awaiting_docstring = None

        m_def = PY_DEF.match(line)
        if m_def:
            pending_def = "private" if m_def.group(1).startswith("_") else "public"
        if pending_def is not None and stripped.endswith(":"):
            awaiting_docstring = pending_def
            pending_def = None

        prev_assign = scan_line(f, path, i, line, prev_assign)

    flush_comment_run(f, path, run_start, run_lines, None)


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
    _, ext = os.path.splitext(path)
    if ext.lower() in SKIP_EXT:
        return None
    try:
        if os.path.getsize(path) > MAX_BYTES:
            return None
        with open(path, "r", encoding="utf-8", errors="strict") as fh:
            return fh.read()
    except (OSError, UnicodeDecodeError):
        return None


def render_text(f):
    order = {"high": 0, "medium": 1, "low": 2}
    keys = sorted(f.buckets, key=lambda k: (order.get(k[1], 3), k[0]))
    if not keys:
        return "SLOP PRE-SCAN: no deterministic signals found. Read the code and apply judgment anyway."
    out = ["SLOP PRE-SCAN -- candidate signals, not verdicts. Confirm each against context before acting.\n"]
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
    out.append(f"Total candidate signals: {total}")
    out.append(
        "Comment rules: the cap is ONE line, and public/exported symbols get no doc-comment "
        "unless the human authorized it (a linter rule is not authorization). License, "
        "generated-file, and build-tag blocks are exempt and already excluded here."
    )
    out.append(
        "Reminders: naming/generic-token is LOW confidence -- role-suffixed domain names "
        "(UserManager, PaymentClient) and tiny-scope locals (i, err, ctx) are NOT slop. "
        "This scan says nothing about over-abstraction or happy-path logic; judge those by reading."
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
    return json.dumps({"signals": payload}, ensure_ascii=False, indent=2)


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
        text = readable(path)
        if text is None:
            continue
        scan_text(f, path, text)
        scanned += 1

    if "--json" in flags:
        print(render_json(f))
    else:
        print(render_text(f))
        print(f"\nScanned {scanned} file(s).")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
