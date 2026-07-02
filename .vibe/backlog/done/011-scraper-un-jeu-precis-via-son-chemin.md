---
status: done
---
# Scraper un Jeu Précis via son Chemin

## Description
Aujourd'hui, la commande `scrape` complète tous les jeux de tous les dossiers ROMs configurés. Une option doit permettre de cibler un seul jeu en fournissant son chemin, pour ne compléter que ce jeu précis depuis le registre, sans avoir à traiter tout le dossier.

## Acceptance Criteria
- [ ] L'utilisateur peut lancer `scrape` en indiquant le chemin d'un jeu précis, et seul ce jeu est complété depuis le registre
- [ ] Si le jeu indiqué n'a pas d'entrée correspondante dans le registre, un message d'erreur clair est affiché et la commande retourne un code d'erreur
- [ ] Le résumé affiché reste cohérent avec ce mode ciblé (ex : 1 traité, 1 complété ou 0 complété selon le cas)
- [ ] Sans cette option, le comportement existant (compléter tous les jeux de tous les dossiers ROMs configurés) reste inchangé

## Notes
S'appuie sur le mécanisme de complétion déjà en place (`registry.CompleteRomsFolder`, item 003). À trancher : le chemin fourni est-il relatif au dossier ROMs configuré, ou un chemin absolu sur le disque ? Il faudra aussi déterminer comment en déduire le système (sous-dossier parent) du jeu ciblé.
