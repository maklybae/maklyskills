# maklyskills

Personal Claude Code skill bundle, installable as a plugin.

Two kinds of skill live here:

- **`flow-*`** — a development pipeline that takes a task from investigation to
  a reviewed change, one stage at a time. Nothing in it hardcodes a build
  system, VCS, or test runner; project-specific facts live in a per-repository
  profile.
- **focused skills** — single-purpose passes usable on their own, such as
  `anti-slop-code`.

## Install

```bash
# in Claude Code
/plugin marketplace add ~/maklyskills
/plugin install maklyskills@maklyskills
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
finding.

`flow` is the umbrella skill: it routes to a stage, owns the task ledger, and
knows how to resume. `flow-setup` detects a repository's commands once and
writes them to `.claude/flow-profile.md`.

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
and codegen commands, VCS and base ref, whether specs are expected, which rule
files bind, which traps to avoid. Written by `flow-setup` after it has watched
the commands succeed. This is the only file the pipeline puts in the repository,
because it is about the repository; it stays untracked unless you decide
otherwise, and the skills never touch your ignore files.

**`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`** —
per task, deliberately outside the repository. Current stage, decisions with
their rejected alternatives, open questions, and every review finding with its
verdict. It holds dead ends and "we consciously decided not to fix this", which
is exactly the material that gets self-censored once it is visible in a shared
tree. The findings list is what makes the review loop converge: a finding
recorded as consciously accepted is not raised again.

## Credits

The plan gate's interview protocol adapts ideas from Matt Pocock's
[grilling](https://github.com/mattpocock/skills) skill (MIT).

## Adding a skill

One directory under `skills/`, containing `SKILL.md` with `name` and
`description` frontmatter. The description is the only thing Claude sees before
deciding to use the skill, so it carries both what the skill does and the
situations that should trigger it. Reference material goes in `references/`
beside it and is read on demand.
