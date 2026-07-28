---
status: done
depends_on: [014]
---
# Import ROMs Folders From The Web UI

## Description
Feeding the registry from the configured ROMs folders currently means leaving the web UI, running `update` on the command line, then restarting the server — the served snapshot being read at startup only. The web UI must be able to trigger that same import, show its outcome, and refresh the served pages immediately. This is the ROMs → registry counterpart of item 023, which covers the opposite direction.

## Acceptance Criteria
- [ ] The web UI offers a control triggering the import of every configured ROMs folder; once the operation ends, a page displays the same summary as the CLI ("N added, N updated, N unchanged")
- [ ] After a successful import, the served pages (home, system page, game page) list the newly imported entries without restarting the server, and the registry's static site is regenerated
- [ ] A configured ROMs folder that cannot be found produces an error message naming that folder, and the served snapshot stays consistent with what the registry holds on disk
- [ ] No configured ROMs folder is a valid case, not an error: the page displays a zero summary along with an explicit message

## Notes
`webui.Handler(reg, registryFolder)` does not know about the configuration: the ROMs folders will have to be passed to it, without duplicating the `internal/config` read the `serve` command already performs. The import is a long batch operation (`registry.ImportFromRomsFolder` with its `ProgressEvent` callback), whereas every mutating route today is instantaneous: holding the write lock for its whole duration would block reads too. Open question to settle at implementation time: synchronous execution rendering a result page, or background execution with a status page — knowing the web UI ships no JavaScript, which rules out any client-side progress bar and leaves meta refresh or a manual reload as the only options. Reuse the discipline already in place for registry changes: apply on a `Clone()`, persist through `internal/store`, swap the served snapshot in only once the write succeeded, and treat a failed site regeneration as a caveat rather than a failure. The route must be a `POST` with the same `crossSite` check as the edit and delete flows, followed by a `303`.
