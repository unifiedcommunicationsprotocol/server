# Constraints

Hard constraints for this codebase. Must be respected unconditionally.

---

## Never Do

### Architecture
- Do not write auth logic in route handlers — middleware handles it
- Do not add Docker or managed cloud services
- [TODO: project-specific architectural constraints]

### Database
- Do not modify Drizzle-generated migration files, snapshots, or journal
- Do not modify Better Auth tables manually — let `bunx auth@latest generate` manage them
- Do not use `bun db:generate` + `migrate` — use `bun db:push` only
- Do not use the `pg` package — use `drizzle-orm/bun-sql`

### Code
- Do not use Node.js built-ins or polyfills — Bun-native APIs only
- Do not write package versions in `package.json` by hand — `bun add <pkg>@latest` on the CLI
- Do not use `process.env` directly in packages — typed build constants or injected config
- Do not commit secrets, tokens, or credentials of any kind

## Always Do

- Generate Better Auth schema (`bunx auth@latest generate`) before first migration
- Keep `docs/llm.md` (CLAUDE.md) factual and concise — procedures go in `.claude/skills/`
- [TODO: project-specific always-do rules]

## External API / Rate Limit Notes

| Service | Limit  | Notes  |
|---------|--------|--------|
| [TODO]  | [TODO] | [TODO] |

## Regulatory / Legal

[TODO: any legal or compliance constraints — GDPR, data residency, retention, etc.]

---

*Last updated: [DATE]*
