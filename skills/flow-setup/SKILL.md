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
artifacts are expected for changes in this repository. One round of questions,
not an interview.

**5. Write the file** using the schema in
[references/profile.md](references/profile.md). Fixed headings, free text
underneath: review subagents are handed these sections verbatim, so the headings
have to be predictable, while the content stays something a human can edit.

**6. Tell the user where it is** and that it is worth committing if the team
would benefit — but leave it untracked unless they say so. Their repository,
their call.

## Scope

The profile describes the repository, not the task: commands, tooling,
conventions, traps. Anything that changes per task belongs in the ledger.

Keep it to one screen. It is read at the start of every stage and pasted into
every review subagent's context; a long profile is a tax paid on every
invocation. Detail that only matters occasionally belongs in the repository's
own documentation, with a pointer from the profile.
