---
status: in_progress
---
# Rewrite Only Changed Registry Entries

## Description
`registry.Save` rewrites every entry's JSON file on every call, even when a single game changed — flagged by a `/vibe:review` pass (2026-07-30) as a performance concern on registries described elsewhere as holding "several thousand games". A one-field correction from the web UI (edit, protect, upload a medium, delete) currently performs thousands of file writes instead of the one that actually changed.

## Acceptance Criteria
- [ ] Saving a single corrected game writes only that game's JSON file, not every entry's
- [ ] A full `Load()`+`Save()` round trip on an untouched registry still produces byte-identical files for every unrelated entry
- [ ] Existing registry tests (`TestSave_*`) continue to pass, and a new test proves an untouched entry's file is not rewritten (e.g. its mtime or inode is unchanged) when a different entry is saved

## Notes
`Save` already writes atomically (temp file + rename, fixed by the same review pass) — this item is only about writing *fewer* files, not about write safety. `Registry.Clone()` is a shallow copy of the `Entries` slice, so diffing "what changed" likely means comparing the candidate against the snapshot it was cloned from, rather than diffing on disk.
