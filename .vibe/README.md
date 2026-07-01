# .vibe/ — Project context

Maintained by the `/vibe:*` commands. Lifecycle of each entry:

| Entry | Lifecycle |
|---|---|
| `README.md`, `index.md`, `modules/`, `models.md` | ♻ Regenerable — recreated by `/vibe:sync`, do not edit by hand |
| `glossary.md` | ⚠ Hybrid — structure generated, **definitions refined by hand**: never delete without review |
| `decisions/` | 🔒 ADRs — append-only, never regenerated, do not delete |
| `backlog/` | 🔒 Work data — never regenerated, do not delete |

Never delete the whole `.vibe/` directory: only the ♻ entries can be recovered by re-running `/vibe:sync`.
