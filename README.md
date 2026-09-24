# maklyskills

Personal skill bundle for Claude Code, Codex, and ChatGPT Work, installable as
a plugin.

Two kinds of skill live here:

- **`flow-*`** — a development pipeline that takes a task from investigation to
  a reviewed change, one stage at a time. Nothing in it hardcodes a build
  system, VCS, or test runner; project-specific facts live in a per-repository
  profile.
- **focused skills** — single-purpose passes usable on their own:
  `anti-slop-code` for the code, `anti-slop-tests` for the test suite.

## Install

```bash
# Claude Code
/plugin marketplace add ~/maklyskills
/plugin install maklyskills@maklyskills

# Codex CLI / ChatGPT desktop app
codex plugin marketplace add ~/maklyskills
# Then install maklyskills from the marketplace and start a new task.
```

From another machine, point the marketplace at the git remote instead of the
local path.

## The pipeline

| Stage | Skill | Produces |
|---|---|---|
| 1 | `flow-explore` | A brief: files to touch, the existing code to imitate, constraints, options |
| 2 | `flow-plan` | A staged plan, or an openspec change when the service uses one |
| 3 | `flow-implement` | The change, built stage by stage in the local idiom, tested as it lands |
| 4 | `flow-test` | The project's verification, run and recorded |
| 5 | `flow-cleanup` | Slop and worthless tests removed, behaviour unchanged |
| 6 | `flow-review` | Verified findings from parallel single-lens subagents |
| 7 | `flow-resolve` | A verdict on every finding, fixes applied and verified |

Stages 6 and 7 repeat until a full review round adds no new blocker or serious
finding. Cleanup runs before the first review and never between rounds;
`flow-resolve` may run one narrower pass over the round fixes at exit. Whoever
edits in a cleanup runs its verification once; nobody repeats it.

Not every change runs all seven. The run is sized first, by how a defect in it
would be found. A **mechanical** change — the compiler or an existing test would
catch a mistake — runs implement and test. A **contained** one — a new branch of
behaviour, a reach you can see whole — adds an in-place cleanup, a two-or-three
lens review and resolve. A **feature** runs everything. Four escalators move a
change up regardless of size: persistent state and contracts, auth and secrets,
concurrency, paths the profile marks `core`. The route is said in one line
before the first edit and only ever revised upward.

`flow` is the umbrella skill: it sizes the run, routes to a stage, owns the task
ledger, and knows how to resume. `flow-setup` detects a repository's commands once and
writes them to `.claude/flow-profile.md`. This remains the canonical shared
profile location for both hosts.

Stage 5 is a dispatcher over the two focused skills: `anti-slop-code` cleans the
code, `anti-slop-tests` prunes the suite. Both work standalone — on a package a
model just filled with generated tests, `anti-slop-tests` is the whole job.

## Unattended runs

`flow` can be handed a goal and left to run — normally from the point the plan
is approved. Two things are declared at launch: where autonomy begins, and where
it ends (work left in the tree, committed, or pushed with a pull request). The
human gates do not disappear, they change shape: a blocker finding is fixed or
the run stops, a question that cannot be asked becomes an `ASSUMED` entry in the
ledger that leads the final report, and the review loop gets a round budget so
it terminates with an honest "not converged, here is what is open" instead of
running all night.

## Two files carry the state

**`<project-root>/.claude/flow-profile.md`** — per repository. Build, test, lint
and codegen commands, VCS and base ref, how critical the code is and what this
repository accepts trading away, whether specs are expected, which rule files
bind, which paths may keep a one-line invariant comment (everywhere else the
count is zero), which traps to avoid. Written by `flow-setup` after it has watched
the commands succeed. This is the only file the pipeline puts in the repository,
because it is about the repository; it stays untracked unless you decide
otherwise, and the skills never touch your ignore files.

**`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`** —
per task, deliberately outside the repository. Current stage, decisions with
their rejected alternatives, open questions, and every review finding with its
verdict. It holds dead ends and "we consciously decided not to fix this", which
is exactly the material that gets self-censored once it is visible in a shared
tree. The findings list is what makes the review loop converge: a finding
recorded as consciously accepted — or as a defect this change inherited rather
than caused — is not raised again.

## Credits

The plan gate's interview protocol adapts ideas from Matt Pocock's
[grilling](https://github.com/mattpocock/skills) skill (MIT).

## Adding a skill

One directory under `skills/`, containing `SKILL.md` with `name` and
`description` frontmatter. The description is the only thing an agent sees before
deciding to use the skill, so it carries both what the skill does and the
situations that should trigger it. Reference material goes in `references/`
beside it and is read on demand.
