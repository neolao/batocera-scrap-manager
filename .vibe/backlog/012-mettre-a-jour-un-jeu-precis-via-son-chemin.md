---
status: in_progress
---
# Mettre à Jour un Jeu Précis via son Chemin

## Description
Aujourd'hui, la commande `update` importe dans le registre tous les jeux de tous les dossiers ROMs configurés. Une option doit permettre de cibler un seul jeu en fournissant son chemin, pour n'importer/mettre à jour que ce jeu précis dans le registre, sans avoir à traiter tout le dossier.

## Acceptance Criteria
- [ ] L'utilisateur peut lancer `update` en indiquant le chemin d'un jeu précis, et seul ce jeu est importé ou mis à jour dans le registre
- [ ] Si le jeu indiqué n'existe pas ou n'a pas d'entrée correspondante dans le `gamelist.xml` de son système, un message d'erreur clair est affiché et la commande retourne un code d'erreur
- [ ] Le résumé affiché reste cohérent avec ce mode ciblé (ex : 1 added, ou 1 updated, ou 0 unchanged selon le cas)
- [ ] Sans cette option, le comportement existant (mettre à jour tous les jeux de tous les dossiers ROMs configurés) reste inchangé

## Notes
Symétrique de l'item 011 (scrape ciblé d'un jeu par chemin), mais dans le sens inverse (dossier ROMs → registre). À trancher : le chemin fourni est-il relatif au dossier ROMs configuré, ou un chemin absolu sur le disque ? Il faudra aussi déterminer comment en déduire le système (sous-dossier parent) du jeu ciblé.
