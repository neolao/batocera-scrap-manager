# Decisions

## [2026-07-01] Registry storage format et périmètre de l'import

**Context:** Implémentation de l'item de backlog 001 (configuration du registry et des ROMs) — la structure exacte du registry était explicitement laissée libre par l'item.
**Decision:** Le registre est un fichier JSON unique (`registry.json`) sous le chemin configuré, indexant les métadonnées des jeux (dédoublonnées par clé système + chemin du ROM) sans copier les fichiers media. La fonction d'import (`registry.ImportFromRomsFolder`) reste interne et n'est pas exposée via une commande CLI dans cet item — elle sera branchée par l'item 002 (commande `update`).
**Reason:** Éviter la complexité de gestion de copie/synchronisation de fichiers media dès ce premier item ; séparer clairement la brique de configuration/import (001) de la commande CLI dédiée avec résumé et code de sortie (002), pour livrer un item testable en CLI sans anticiper la commande de 002.
**Rejected alternatives:** Stockage du registry en base de données (SQLite) — jugé prématuré pour un projet CLI simple ; copie des fichiers media dans le registry — reporté à une future itération si le besoin se confirme.

## [2026-07-01] Détection des mises à jour et gestion de l'absence de dossiers de ROMs (item 002)
**Context:** Implémentation de la commande `update` (item de backlog 002). L'item exige un résumé distinguant "ajoutées / mises à jour / inchangées", alors que le mécanisme d'import existant (001) ne distinguait que "ajoutées / inchangées" (une entrée déjà connue n'était jamais comparée à son contenu à jour).
**Decision:** `Registry.Import` est étendu pour comparer le contenu complet du jeu (toutes les métadonnées) à l'entrée déjà connue (clé système + chemin du ROM) ; en cas de différence, l'entrée est remplacée et comptée comme "mise à jour" plutôt que "inchangée". Par ailleurs, l'absence de dossier de ROMs configuré n'est pas traitée comme une erreur par la commande `update` — elle affiche simplement un résumé à zéro.
**Reason:** Répondre fidèlement au résumé à 3 catégories exigé par les critères d'acceptation ; une absence de dossier de ROMs configuré est un état valide (ex. juste après l'item 001, avant tout `config add-roms-folder`), pas une erreur bloquante — contrairement à un dossier configuré mais introuvable sur le disque, qui reste une erreur.
**Rejected alternatives:** Garder "inchangées" pour les entrées déjà connues sans comparaison de contenu — rejeté car ne respecte pas le critère d'acceptation du résumé à 3 catégories ; faire échouer `update` quand aucun dossier de ROMs n'est configuré — rejeté pour rester cohérent avec le comportement tolérant de `config list` (qui affiche "(none)" plutôt que d'échouer).
