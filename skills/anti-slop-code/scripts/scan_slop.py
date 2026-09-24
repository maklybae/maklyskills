#!/usr/bin/env python3
"""Deterministic pre-scan for AI-slop signals in source code.

Emits candidate signals, not verdicts. Every hit must be confirmed with
judgment against the surrounding context before acting on it. The scan is
language-agnostic: it works on comment text and raw source patterns that hold
across languages, and deliberately stays silent on things that need semantic
understanding (over-abstraction, wrong abstractions, happy-path logic).

The comment policy is zero: every human-readable comment line, trailing comment
and docstring is a signal. Directives the tooling reads are exempt. Under a path
passed with --allow-invariants one comment line may stay; longer runs are still
signalled. Prose files (Markdown, reStructuredText, plain text) are not checked
for comments.

Usage:
    python3 scan_slop.py FILE [FILE ...]
    python3 scan_slop.py DIR              # walks, skipping vendor/.git/etc.
    git diff --name-only | python3 scan_slop.py --stdin-list
    python3 scan_slop.py FILE --json
    python3 scan_slop.py DIR --allow-invariants PATH [--allow-invariants PATH ...]
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

PROSE_EXT = {".md", ".markdown", ".mdx", ".txt", ".rst", ".adoc"}
SLASH_EXT = {
    ".go", ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".rs", ".java", ".kt", ".kts",
    ".scala", ".swift", ".c", ".h", ".cc", ".cpp", ".hpp", ".cs", ".proto", ".dart",
    ".php", ".svelte", ".vue",
}
HASH_EXT = {
    ".py", ".sh", ".bash", ".zsh", ".rb", ".yaml", ".yml", ".toml", ".pl", ".r",
    ".cfg", ".conf", ".ini", ".mk", ".tf",
}
HASH_FILES = {"Makefile", "makefile", "GNUmakefile", "ya.make", "Dockerfile"}
DASH_EXT = {".sql", ".lua", ".hs"}
BLOCK_ONLY_EXT = {".css", ".scss", ".less", ".html", ".xml"}

LINE_START = {
    "slash": re.compile(r"^(?://|/\*|\*/|\*(?:\s|$)|<!--|-->)"),
    "hash": re.compile(r"^#(?![\[!])"),
    "dash": re.compile(r"^(?:--(?=\s|$)|/\*|\*/|\*(?:\s|$))"),
    "block": re.compile(r"^(?:/\*|\*/|\*(?:\s|$)|<!--|-->)"),
    "generic": re.compile(r"^(?://|#(?![\[!])|--(?=\s|$)|;|/\*|\*/|\*(?:\s|$)|<!--|-->)"),
}
TRAILING = {
    "slash": re.compile(r"(?<=\s)//"),
    "hash": re.compile(r"(?<=\s)#(?=\s|$)"),
    "dash": re.compile(r"(?<=\s)--(?=\s|$)"),
}

CPP_DIRECTIVE = re.compile(
    r"^#\s*(include|define|ifn?def|ifdef|endif|pragma|if|else|elif|undef|error|line|import)\b"
)

LICENSE_MARK = re.compile(
    r"SPDX-License-Identifier|copyright|licen[cs]ed under|all rights reserved|"
    r"code generated|do not edit|@generated",
    re.IGNORECASE,
)
DIRECTIVE_COMMENT = re.compile(
    r"^\s*(?:"
    r"//go:|//\s*\+build|//\s*nolint|//\s*lint:|//export\s|//line\s|"
    r"//\s*eslint|/\*\s*eslint|//\s*@ts-|//\s*prettier-ignore|//\s*biome-ignore|"
    r"(?://|/\*)\s*(?:istanbul|c8|v8)\s+ignore|//\s*@(?:vitest|jest)-environment|"
    r"//\s*\+(?:k8s|kubebuilder|genclient)|//\s*swagger:|//\s*#nosec|"
    r"//\s*(?:unordered\s+)?output:|"
    r"#\s*type:\s*ignore|#\s*noqa|#\s*pylint:|#\s*mypy:|#\s*ruff:|#\s*fmt:|#\s*pyright:|"
    r"#\s*isort:|#\s*pragma\b|#\s*nosec|#\s*shellcheck\s|#\s*yamllint\s|#\s*syntax=|"
    r"#\s*-\*-|#\s*frozen_string_literal:|#\s*rubocop:|"
    r"--\s*\+goose|--\s*\+migrate|--\s*name:\s*\w+\s+:|"
    r"(?://|#|\*|--)\s*Deprecated:"
    r")",
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
        self.comment_lines = 0
        self.allowed_comment_lines = 0

    def add(self, category, confidence, path, lineno, text):
        key = (category, confidence)
        entry = self.buckets.setdefault(key, {"count": 0, "samples": []})
        entry["count"] += 1
        if len(entry["samples"]) < MAX_SAMPLES:
            entry["samples"].append(f"{path}:{lineno}: {text.strip()[:140]}")

    def count_comments(self, lines, allowed):
        self.comment_lines += lines
        if allowed:
            self.allowed_comment_lines += lines


class Docstring:
    def __init__(self, kind, start, quote):
        self.kind = kind
        self.start = start
        self.quote = quote
        self.lines = 1


def strip_comment(line):
    m_cont = BLOCK_CONT.match(line)
    if m_cont:
        return m_cont.group(1)
    if BLOCK_MARK.search(line):
        return line
    m = COMMENT_TEXT.search(line)
    return m.group(1) if m else None


def scan_line(f, path, lineno, line, prev_assign, prose):
    if EMOJI.search(line):
        f.add("cosmetic/emoji", "high", path, lineno, line)
    for ch, name in UNICODE_HAZARDS.items():
        if ch in line:
            f.add(f"cosmetic/unicode-{name}", "high", path, lineno, line)
            break
    if not prose and BANNER.search(line) and any(m in line for m in ("//", "#", "/*", "*", "--")):
        f.add("comment/banner", "medium", path, lineno, line)

    comment = None if prose else strip_comment(line)
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


def comment_family(path):
    base = os.path.basename(path)
    ext = os.path.splitext(base)[1].lower()
    if ext in PROSE_EXT:
        return "prose"
    if ext in SLASH_EXT:
        return "slash"
    if ext in HASH_EXT or base in HASH_FILES or base.startswith("Dockerfile"):
        return "hash"
    if ext in DASH_EXT:
        return "dash"
    if ext in BLOCK_ONLY_EXT:
        return "block"
    return "generic"


def is_comment_line(line, family):
    s = line.strip()
    if not s or s.startswith("#!") or CPP_DIRECTIVE.match(s):
        return False
    return bool(LINE_START[family].match(s))


def trailing_comment(line, family):
    pattern = TRAILING.get(family)
    if pattern is None:
        return None
    for m in pattern.finditer(line):
        prefix = line[: m.start()]
        if not prefix.strip():
            return None
        if any(prefix.count(q) % 2 for q in ('"', "'", "`")):
            continue
        return line[m.start():]
    return None


def flush_comment_run(f, path, start, lines, terminator, allowed):
    if not lines or any(LICENSE_MARK.search(ln) for ln in lines):
        return
    human = [ln for ln in lines if not DIRECTIVE_COMMENT.match(ln)]
    if not human:
        return
    f.count_comments(len(human), allowed)
    if not allowed:
        f.add("comment/present", "high", path, start, f"{len(human)} line(s): {human[0].strip()}")
    elif len(human) > 1:
        f.add("comment/block-over-one-line", "high", path, start,
              f"{len(human)} consecutive comment lines on an allowed path (cap is one)")
    if terminator is not None and EXPORTED_DECL.match(terminator):
        f.add("comment/public-doc-comment", "high", path, start, human[0])


def flush_trailing(f, path, lineno, line, family, allowed):
    comment = trailing_comment(line, family)
    if comment is None or DIRECTIVE_COMMENT.match(comment):
        return
    f.count_comments(1, allowed)
    if not allowed:
        f.add("comment/trailing", "high", path, lineno, line)


def flush_docstring(f, path, doc, allowed):
    f.count_comments(doc.lines, allowed)
    category = "comment/public-doc-comment" if doc.kind == "public" else "comment/docstring"
    f.add(category, "high", path, doc.start, f"{doc.kind} docstring, {doc.lines} line(s)")


def scan_text(f, path, text, allowed):
    family = comment_family(path)
    prose = family == "prose"
    prev_assign = None
    run_lines = []
    run_start = 0
    pending_def = None
    awaiting_docstring = "module" if path.endswith(".py") else None
    doc = None

    for i, line in enumerate(text.splitlines(), start=1):
        stripped = line.strip()

        if doc is not None:
            doc.lines += 1
            if doc.quote in stripped:
                flush_docstring(f, path, doc, allowed)
                doc = None
        elif not prose:
            comment_line = is_comment_line(line, family)
            if comment_line:
                if not run_lines:
                    run_start = i
                run_lines.append(line)
            else:
                flush_comment_run(f, path, run_start, run_lines, line, allowed)
                run_lines = []
                flush_trailing(f, path, i, line, family, allowed)

            if awaiting_docstring is not None and stripped and not comment_line and not stripped.startswith("#!"):
                m_doc = DOCSTRING_START.match(stripped)
                if m_doc:
                    doc = Docstring(awaiting_docstring, i, m_doc.group(1))
                    if doc.quote in stripped[m_doc.end():]:
                        flush_docstring(f, path, doc, allowed)
                        doc = None
                awaiting_docstring = None

            m_def = PY_DEF.match(line)
            if m_def:
                pending_def = "private" if m_def.group(1).startswith("_") else "public"
            if pending_def is not None and stripped.endswith(":"):
                awaiting_docstring = pending_def
                pending_def = None

        prev_assign = scan_line(f, path, i, line, prev_assign, prose)

    flush_comment_run(f, path, run_start, run_lines, None, allowed)
    if doc is not None:
        flush_docstring(f, path, doc, allowed)


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
    outside = f.comment_lines - f.allowed_comment_lines
    tally = (
        f"Comment lines: {f.comment_lines} ({outside} outside allowed paths, "
        f"{f.allowed_comment_lines} on allowed paths). The policy is zero outside allowed paths."
    )
    if not keys:
        return ("SLOP PRE-SCAN: no deterministic signals found. Read the code and apply judgment anyway.\n"
                + tally)
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
    out.append(tally)
    out.append(
        "Comment rules: zero human-readable comments, docstrings included; move a real why into a "
        "name, a test name, docs or the report, then delete. Directives (build tags, pragmas, linter "
        "switches, Deprecated:, license and generated-file banners) are exempt and already excluded. "
        "On an --allow-invariants path one line may stay; a public doc-comment never does unless the "
        "human asked for it in this session, and a linter rule is not asking."
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
    comment_lines = {
        "total": f.comment_lines,
        "outside_allowed": f.comment_lines - f.allowed_comment_lines,
    }
    return json.dumps({"signals": payload, "comment_lines": comment_lines}, ensure_ascii=False, indent=2)


def parse_args(argv):
    paths = []
    flags = set()
    allowed = []
    it = iter(argv)
    for a in it:
        if a == "--allow-invariants":
            value = next(it, None)
            if value is None:
                raise SystemExit("--allow-invariants needs a path")
            allowed.append(value)
        elif a.startswith("--allow-invariants="):
            allowed.append(a.split("=", 1)[1])
        elif a.startswith("--"):
            flags.add(a)
        else:
            paths.append(a)
    return paths, flags, [os.path.abspath(p) for p in allowed]


def is_allowed(path, roots):
    full = os.path.abspath(path)
    return any(full == r or full.startswith(r + os.sep) for r in roots)


def main(argv):
    args, flags, allowed_roots = parse_args(argv)
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
        scan_text(f, path, text, is_allowed(path, allowed_roots))
        scanned += 1

    if "--json" in flags:
        print(render_json(f))
    else:
        print(render_text(f))
        print(f"\nScanned {scanned} file(s).")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
