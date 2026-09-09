---
name: flow-setup
description: >-
  Work out how a specific repository is built, tested, linted, versioned and
  specified, verify those commands actually run, and write them to
  .claude/flow-profile.md so every later stage and every review subagent uses
  real commands instead of guesses. Use this skill when the user says "настрой
  флоу под этот репо", "set up the pipeline here", asks what the test or build
  command is, or wants the profile refreshed after the build system changed —
  and use it automatically, without asking, whenever another flow-* stage needs
  a build/test/lint command and .claude/flow-profile.md does not exist yet.
---

# Flow setup

The pipeline holds no knowledge of build systems. Every command it runs comes
from `.claude/flow-profile.md`, and this skill is what puts it there.

The profile is worth its own stage for one reason: **a wrong command is worse
than a missing one.** A missing command makes the next stage stop and ask. A
wrong one — `npm test` in a repository whose tests run through something else —
produces a confident green that nobody earned, and every later stage builds on
it. So the deliverable here is not a plausible-looking file; it is a file whose
commands you have watched succeed.

## Steps

**1. Check for an existing profile.** If `.claude/flow-profile.md` exists, read
it and stop, unless the user asked for a refresh or you have concrete evidence
it is stale (a command in it just failed, or the build system markers changed).
Re-detecting on every run wastes time and invites drift.

**2. Detect.** Walk the marker table in
[references/profile.md](references/profile.md). Detection is cheap; ranking the
evidence is the part that needs care, because sources disagree:

1. What the **user** says, now or in the project's memory. Their stated
   preference wins even when a rule file says otherwise — they know why.
2. What the repository's own **instruction files** say (`CLAUDE.md`,
   `AGENTS.md`, `CONTRIBUTING.md`, rule directories).
3. What **CI** runs. This is the strongest evidence of what must pass, because
   it is what actually gates a merge.
4. A **task runner** (`Makefile`, `justfile`, `Taskfile.yml`) if one exists —
   in a repository that has one, the language's default command is usually the
   wrong entry point.
5. Language and build-system **defaults**, as a last resort.

**3. Verify by running.** Pick the cheapest command that proves the toolchain
works — a lint pass, or the tests of one small package, not the whole suite —
and run it. Record what you ran and what came back in the `## Verified` section.
If it fails for a reason that is about the command rather than the code, fix the
command and run again. Do not write down a command you have not seen work.

**4. Ask only what is left.** Detection settles most of the profile. Ask the
user only where two plausible answers remain and the choice is theirs — which of
two test entry points they prefer, what the base branch is, whether spec
artifacts are expected for changes in this repository, how critical the code is
where the signals below disagree. One round of questions, not an interview.

**5. Write the file** using the schema in
[references/profile.md](references/profile.md). Fixed headings, free text
underneath: review subagents are handed these sections verbatim, so the headings
have to be predictable, while the content stays something a human can edit.

**6. Tell the user where it is** and that it is untracked. Do not add it to
`.gitignore`, or whatever the repository's ignore file is: those are shared
files, and editing one on your own initiative to hide a file you just created is
a bigger intrusion than the untracked file itself. If they want it committed or ignored, they will say so.

## Judging criticality

`## Criticality` is the one section that is a judgement rather than a detection,
and it is the one every later close call reads. The signals, in the order they
settle the question: who imports this code — a package with consumers outside
its own tree is core almost regardless of what it does; whether it owns
persistent state or a schema, because a rollback does not undo a write; whether
a deploy manifest, an on-call rotation or an alert names it; and whether the
tree is a scratch surface — fixtures, scripts, an admin page nobody is paged
for.

Where the signals disagree, ask. It is one question with a durable answer, which
makes it the cheapest question in the pipeline — cheaper by far than the same
judgement re-derived, differently, in every review round.

## Which directory is the project root

The profile goes in `<project-root>/.claude/flow-profile.md`, and that location
then *defines* the project root for everything else — the ledger path is derived
from it. In a plain repository the root is obvious. In a monorepo it is not:
prefer the directory the user actually works in (the service or package),
not the top of a mount that contains thousands of unrelated projects. When two
readings are both plausible, ask — this is one of the questions worth spending
in step 4.

## Scope

The profile describes the repository, not the task: commands, tooling,
conventions, traps. Anything that changes per task belongs in the ledger, which
lives outside the repository entirely — see
`../flow/references/ledger.md`, relative to this skill's directory.

Keep it to one screen. It is read at the start of every stage and pasted into
every review subagent's context; a long profile is a tax paid on every
invocation. Detail that only matters occasionally belongs in the repository's
own documentation, with a pointer from the profile.
