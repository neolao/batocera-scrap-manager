---
status: done
---
# Feedback Visuel CLI Pendant Update

## Description
La commande `update` reste actuellement silencieuse pendant toute la durée de son exécution : elle ne parle qu'une fois tout traité, en affichant un unique résumé final ("X added, Y updated, Z unchanged"). Pour une collection de jeux volumineuse, avec copie de médias potentiellement lourds, cette absence de retour visuel peut donner l'impression que l'outil est bloqué. Il faut donc afficher une progression pendant l'exécution, pas seulement à la fin.

## Acceptance Criteria
- [ ] L'utilisateur voit s'afficher le nom du système en cours de traitement au fur et à mesure que `update` parcourt les dossiers de ROMs configurés
- [ ] L'utilisateur voit un indicateur de progression par jeu (ex. compteur "jeu X sur Y" ou nom du jeu traité) pendant le traitement d'un système
- [ ] Le résumé final ("added/updated/unchanged") continue de s'afficher à la fin de l'exécution, inchangé
- [ ] Le format de sortie de `update` reste exploitable en environnement non interactif (pas de rendu qui casse la sortie quand elle est redirigée vers un fichier ou un pipe)

## Notes
La commande `update` (`internal/cli/update.go`) et l'import (`registry.ImportFromRomsFolder`) sont déjà implémentés (item 002) ; ce n'est pas une dépendance bloquante puisqu'ils existent déjà, mais cet item viendra les instrumenter pour émettre la progression au fil de l'exécution plutôt qu'en une seule fois à la fin. Le niveau de détail exact du feedback (par système seulement, ou aussi par jeu) reste à affiner avec l'utilisateur au moment de l'implémentation.
