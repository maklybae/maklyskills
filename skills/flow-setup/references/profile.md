# The project profile

`<project-root>/.claude/flow-profile.md`. Fixed headings, free text inside them.

This is the one file the pipeline writes into the repository, because it
describes the repository rather than the person working on it: objective,
short, and useful to anyone who opens the tree. Per-task working state goes in
the ledger, outside the repository. Leave the profile untracked and leave the
repository's ignore files alone.

## Schema

```markdown
# Flow profile

## Stack
Languages, frameworks, how the tree is laid out, where the code you touch lives.
Two or three lines.

## Criticality
- core: `<paths>` — shared or infrastructural; a defect reaches callers who
  cannot see it, and a rollback does not undo what it already wrote
- service: `<paths>` — a production service with its own on-call and rollback
- peripheral: `<paths>` — internal tooling, admin surfaces, scripts, fixtures
- may be traded here: one line naming what this repository accepts skipping
A level this repository does not have is omitted, not written empty.

## Commands
- build: `<cmd> <dir>`
- test (scoped): `<cmd> <dir>`
- test (full): `<cmd>`
- test (single): `<cmd> -run <name>`
- lint: `<cmd> <dir>`
- format: `<cmd>`
- codegen: `<cmd>` — and what must never be hand-edited
Each line is one copy-pasteable command with `<dir>`/`<name>` placeholders.
Omit a line rather than inventing one; "none" is a legitimate value.

## VCS
- tool: git | hg | svn | jj | none
- base ref: the branch or revision changes are diffed against
- diff vs base: `<cmd>`
- file at base: `<cmd>` — one file as it stood at the base ref
- ship: how a change reaches review here (commit → PR command, or "ask the user")

## Spec workflow
- openspec: none | roots: `<paths>`
- other conventions: ADRs, PRDs, ticket links, where they live

## Rules
Files that constrain how code is written here, most authoritative first.
Include the two or three rules that are most often violated by a fresh agent.

## Constraints
Traps someone new to this repository falls into: filesystem behaviour, search
that must not be run, network restrictions, builds slow enough to change how you
work.

## Verified
- <date> `<cmd>` → <result>
```

## Marker table

Presence of a marker suggests the tooling; the ranking in `SKILL.md` decides
when two markers disagree. A task runner or CI config beats the language default.

| Marker | Likely toolchain | Notes |
|---|---|---|
| `BUILD.bazel`, `WORKSPACE`, `MODULE.bazel`, `BUCK`, `pants.toml` | Bazel / Buck / Pants | Work in targets, not directories, and prefer the repo's wrapper (`./bazelw`, `./pants`) over a globally installed binary. Build manifests are usually generated, not hand-edited — find the command that regenerates them |
| A build/test CLI checked into the repo and referenced by its docs or CI | House toolchain | Large repositories often front the whole toolchain with one command of their own. When there is one, the language default is the wrong entry point. Note what it regenerates, and any restriction it places on searching the tree |
| `go.mod` | Go | `go build ./...`, `go test ./... -race`. In a monorepo the vendored or wrapped toolchain often replaces the bare `go` binary — check the rule files |
| `package.json` | Node | Read `scripts`; the lockfile names the package manager (`package-lock` → npm, `pnpm-lock` → pnpm, `yarn.lock` → yarn, `bun.lockb` → bun). Note any pinned Node version — installing under the wrong one rewrites the lock |
| `pyproject.toml`, `setup.py`, `requirements.txt` | Python | `uv`/`poetry`/`hatch` from the `[tool]` tables; tests usually `pytest` |
| `Cargo.toml` | Rust | `cargo build`, `cargo test`, `cargo clippy` |
| `pom.xml`, `build.gradle(.kts)` | JVM | `mvn`/`gradle` wrappers (`./mvnw`, `./gradlew`) when present |
| `Makefile`, `justfile`, `Taskfile.yml` | Task runner | Read the target list. In a repository that has one, this is usually the intended entry point |
| `.github/workflows/*`, `.gitlab-ci.yml`, `Jenkinsfile`, `.circleci/config.yml`, `azure-pipelines.yml`, `.teamcity/` | CI | The strongest evidence of what must pass before a merge |
| `.git` / `.hg` / `.svn` / `.jj` | VCS | Also note the default branch name — `main`, `master`, `trunk`, `develop` |
| `openspec/` with `config.yaml` | openspec | Record every root; in a monorepo they sit per service, not once at the top |
| `specs/`, `docs/adr/`, `docs/rfc/` | Spec conventions | Note the format actually used in recent files, not the template |
| `CLAUDE.md`, `AGENTS.md`, `.cursor/rules/`, `.claude/rules/`, `CONTRIBUTING.md` | Rule sources | List by authority; nested files override the root for their subtree |

## Filling the sections well

**Commands.** Prefer the narrowest form that still proves something. A profile
whose only test command runs the entire monorepo will be skipped under time
pressure, which is how unverified work ships. Give a scoped command and a full
one, and let the stages choose.

**Criticality.** This is what a close call reads when it has to decide how much
robustness a change owes — and writing it down once is the point, because a
judgement re-derived per task is a mood. The `may be traded` line carries most of
the value: name the concrete thing this repository routinely lets go — two
writes without a transaction that a retry reconciles anyway, a counter nobody
alerts on — because a section that only forbids says nothing about what is
allowed, which is the half that gets guessed wrong.

**VCS.** `file at base` is how a reviewer tells a defect this change introduced
from one it merely inherited. Without it that distinction is read off the diff
alone, and the diff is silent exactly where it matters — in the code the change
touched but did not write.

**Rules.** Do not summarise every rule file — point at them, then call out the
two or three that a fresh agent breaks most often here. The value is in the
surprises: a repository that forbids the standard JSON package, or that runs
tests through a wrapper, or that regenerates a directory you would otherwise
edit.

**Constraints.** Write only what changes behaviour. "The repo is large" changes
nothing; "recursive search outside the project directory hangs the filesystem"
changes everything.

**Verified.** One line per command you actually ran, with the real result. This
section is the difference between a profile and a guess, and it is the first
thing to re-check when a stage's command fails unexpectedly.
