---
date: 2026-07-29
status: accepted
---
# An uploaded medium is named by the registry, never by the client
**Context:** Managing a game's media from the web UI (backlog item 017) is the first flow of the tool that takes a *file* from a request and writes it into the registry folder. Every other write so far derived its destination from data the tool itself owns — a `gamelist.xml` it parsed, a path it validated through `ValidatePath`.

**Decision:** The name of the stored file is composed by the registry from the entry's game ID, the medium's own subfolder and suffix, and the extension of the uploaded file — and the extension is accepted only if it belongs to that medium's allow-list. The filename the client supplied is read for its extension and for nothing else; it never reaches the disk. The four media are described by one table (accessor, subfolder, suffix, accepted extensions), which is the same table `Remove` and every media copy already walk.

**Reason:** A name taken from a request is a path to anywhere: `../../..`, an absolute path, another game's file, a `.php` under a folder someone else serves. Validating such a name means enumerating what is dangerous; deriving it means only what the registry can name is ever written. The allow-list per medium follows from the same reasoning — a video accepted as a cover art would render as a broken image, and an extension outside the list buys nothing.

Deriving the name also makes a replacement natural: the same medium re-uploaded with the same extension lands on the same file and overwrites it, while a different extension yields a different file, whose predecessor the caller erases *after* the registry was written — the write-first-erase-after order [`decisions/024`](024-the-rom-path-is-an-identity-renamed-write-first-erase-after.md) established for a rename, and for the same reason: the write is what fulfills the intent. A deletion keeps the opposite order ([`decisions/022`](022-a-deletion-is-committed-when-the-game-file-is-gone.md)) — there the erasure is the intent.

**Rejected alternatives:**
- *Keep the client's filename, sanitized.* Sanitizing is an enumeration of what is dangerous, and every such list is one case short; it also lets two games of a system collide on one file, which is the very drift [`decisions/014`](014-dedupe-by-extension-stripped-filename-too.md) was written about.
- *Name the file from a hash of its content.* Collision-free and traversal-free, but the registry folder would stop being browsable by a human — and it is browsed, both by the consultation site's relative links and by whoever inspects it. The game ID is what names every other file of an entry.
- *Sniff the real content type instead of trusting the extension.* The extension is what decides the stored reference either way, since `gamelist.xml` names files rather than types; sniffing would refuse a correctly named file whose bytes the standard library does not recognize, for a threat the derived name already closes.
