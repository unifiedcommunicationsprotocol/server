---
name: memory-location-constraint
description: "All Claude Code memories must be saved within the project directory, not external system locations"
metadata:
  type: feedback
---

## Rule

**All Claude Code memories MUST be saved within this project directory.**

Store memories in locations like:
- `.claude/memory/` (this directory)
- `docs/memory/`
- Other project-local paths

**Never save memories to:**
- `~/.claude-me/...` (global user directory)
- `~/.claude/...` (external configuration directory)
- Any path outside the project root

**Why:** 
- Memories are project-specific context, not global
- Should be version-controlled and shared with the team
- Keeps project state self-contained and portable
- Avoids dependency on system-wide configuration

## How to Apply

When saving a memory:
1. Create the file in `.claude/memory/` or equivalent project path
2. Stage and commit the memory file with related code changes
3. Reference the memory file in `.claude/memory/MEMORY.md` (project index)

This applies to ALL memory types (user, feedback, project, reference).

## Scope

This constraint is absolute and applies to:
- All future conversations
- All memory saves on this project
- No exceptions or workarounds
