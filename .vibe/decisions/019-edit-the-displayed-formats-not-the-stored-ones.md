---
date: 2026-07-26
status: accepted
---
# Edit the displayed formats, not the stored ones

**Context:** Two of the eight editable metadata fields are stored in Batocera's own conventions: the
rating is a decimal string between 0 and 1 (`0.85`), the release date is `YYYYMMDDTHHMMSS`
(`19910101T000000`). Both are rendered in a lossy, friendlier form — five stars plus a `4/5` label,
and the year alone. A form has to pick which of the two to expose.

**Decision:** The form edits what the page displays: the rating is a `<select>` offering "Not rated"
plus `0/5`…`5/5`, the release date is a year `<input type="number">`. On save, a field whose
submitted value maps back to the same displayed value as the stored one is left **byte-identical** —
`0.85` stays `0.85`, `19910101T000000` keeps its month, day and time. Only a value the user actually
changed is rewritten (`4/5` → `0.8`, a year → `YYYY0101T000000`), and an emptied field clears the
stored one. "Not rated" (empty) and `0/5` (`0`) stay distinct, as they already render differently.
The conversions live next to their formatters in `internal/site`, so the two directions can never
drift apart.

**Reason:** Exposing `0.85` and `19910101T000000` shows Batocera's storage convention to someone who
just wants to fix a typo. Editing the displayed value is only safe if an untouched lossy field is
never rewritten — otherwise merely opening the form and saving would degrade `0.85` to `0.8` and
invent a month and a day.

**Rejected alternatives:**
- *Raw `0..1` and `YYYYMMDDTHHMMSS` inputs* — faithful, but unreadable and easy to corrupt.
- *`<input type="date">`* — invents a month and a day the UI never shows, and cannot represent an
  existing value it fails to parse.
- *Storing whatever the form submits, unvalidated* — a value the view cannot parse renders as a
  placeholder dash, i.e. silent data loss discovered much later.
