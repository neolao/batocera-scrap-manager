---
status: done
depends_on: [014]
---
# Suppression D'une Entrée Depuis La Web UI

## Description
La suppression d'une entrée du registre n'est possible qu'en ligne de commande (`remove <system> <rom-filename>`, item 005), ce qui oblige à connaître le nom exact du fichier ROM. Depuis la fiche d'un jeu dans la web UI, on doit pouvoir supprimer son entrée du registre — métadonnées et médias — avec une confirmation explicite avant l'action, puis être renvoyé vers la liste.

## Acceptance Criteria
- [ ] La fiche d'un jeu propose une suppression demandant une confirmation explicite avant d'agir
- [ ] Après confirmation, le jeu disparaît de la page d'accueil et du site statique `index.html`, et son fichier JSON ainsi que ses médias ne sont plus présents dans le dossier du registre
- [ ] Une requête `GET` sur l'URL de suppression ne supprime rien
- [ ] Supprimer un jeu déjà supprimé (ou inconnu) répond 404 sans modifier le registre

## Notes
Réutiliser `registry.Remove`, qui supprime déjà le fichier JSON et les quatre médias via la table `mediaFields`, puis régénérer le site statique comme le fait la commande `remove`. Rediriger en `303` vers la page d'accueil après suppression.
