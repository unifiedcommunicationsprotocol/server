---
name: git-safety-constraint
description: "Critical safety constraint: never run git commands from project root's parent directories"
metadata:
  type: feedback
---

## Rule

**CRITICAL: Never run git commands from parent directories of the project root.**

Always `cd` into the project directory FIRST before running any git command (git status, git log, git add, git commit, git push, etc.).

**Why:** Running git commands from parent directories can:
- Accidentally initialize git repos in unexpected locations
- Pollute global git state
- Affect unrelated repositories
- Create merge conflicts across project boundaries
- Corrupt working trees

## How to Apply

Before EVERY git operation:
1. Check current working directory with `pwd`
2. Verify you are in the project root directory (not a parent path)
3. If not in the project directory, `cd` there first
4. Then run the git command

This applies to ALL git commands: `git status`, `git log`, `git diff`, `git add`, `git commit`, `git push`, `git reset`, `git branch`, `git checkout`, etc.

## Scope

This constraint is absolute and applies to:
- All future conversations
- All git operations on this project
- No exceptions or workarounds

Always verify your working directory before git operations.
