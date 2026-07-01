# Decisions

## [2026-07-01] Registry storage format et périmètre de l'import

**Context:** Implémentation de l'item de backlog 001 (configuration du registry et des ROMs) — la structure exacte du registry était explicitement laissée libre par l'item.
**Decision:** Le registre est un fichier JSON unique (`registry.json`) sous le chemin configuré, indexant les métadonnées des jeux (dédoublonnées par clé système + chemin du ROM) sans copier les fichiers media. La fonction d'import (`registry.ImportFromRomsFolder`) reste interne et n'est pas exposée via une commande CLI dans cet item — elle sera branchée par l'item 002 (commande `update`).
**Reason:** Éviter la complexité de gestion de copie/synchronisation de fichiers media dès ce premier item ; séparer clairement la brique de configuration/import (001) de la commande CLI dédiée avec résumé et code de sortie (002), pour livrer un item testable en CLI sans anticiper la commande de 002.
**Rejected alternatives:** Stockage du registry en base de données (SQLite) — jugé prématuré pour un projet CLI simple ; copie des fichiers media dans le registry — reporté à une future itération si le besoin se confirme.
