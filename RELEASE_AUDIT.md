# cmdguard release audit — 2026-06-18

Audit binary: `/tmp/cmdguard-audit` built from current HEAD with
`-ldflags "-X main.version=audit-<sha> -X main.commit=<sha>"`.

Sweep environment:
- `CMDGUARD_CONFIG_DIR=/tmp/cmdguard-sweep.<id>/cfg` (isolated)
- `CMDGUARD_NONINTERACTIVE=1`
- `PATH` left at the agent default (this matters — see Finding 1)
- Playground: `/tmp/cmdguard-sweep.<id>/playground`

Severity legend:
- **P0** must fix before tag
- **P1** should fix before tag
- **P2** nice-to-have, can ship without
- **P3** doc/cosmetic

---

## Findings

### 🚨 P0 — recursion when PATH wrapper points at a different cmdguard

**Repro**: with `CMDGUARD_CONFIG_DIR` pointing to a non-default path
but `PATH` still containing `~/.cmdguard/bin/`, run `cmdguard rm <file>`.

**Observed**: process hangs for seconds, executes
`/Users/haolin/.cmdguard/bin/rm` (the wrapper), wrapper calls
`/Users/haolin/Bin/cmdguard rm`, which loops back into cmdguard.
Output literally says `executing: /Users/haolin/.cmdguard/bin/rm`.

**Root cause**: `findRealCommand` skips ONLY paths under
`config.ConfigDir()/bin`. When the runtime overrides `CONFIG_DIR`
to a sandbox dir, the *production* `~/.cmdguard/bin/` wrapper is
not skipped. `os.SameFile(self, target)` does not save us either
because the wrapper is a shell script, not the cmdguard binary.

**Why it matters in production**: low risk for real users (they
don't override `CONFIG_DIR`), but:
- breaks any test/CI that wants to sandbox cmdguard while user has
  a real install
- if a user ever installs cmdguard to two locations, recursion
  becomes a live hazard
- silent recursion on a destructive-command tool is reputationally
  bad even if rare

**Fix sketch**: detect cmdguard wrappers by content (shebang +
`exec ... cmdguard`) rather than by path prefix. Or refuse to exec
any shell script whose first non-shebang line contains the literal
"cmdguard". Both are heuristics; the cleanest fix is to put a
recognisable marker line in the generated wrappers and skip on
that.

---
*more findings to come as sweep progresses*
