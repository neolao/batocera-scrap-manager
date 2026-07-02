---
status: in_progress
---
# Exclure les Jeux sans Données du Registre

## Description
Actuellement, la commande `update` importe dans le registre tous les jeux trouvés dans les `gamelist.xml`, même ceux qui n'ont ni description ni image (donc aucune donnée de scraping exploitable). Ces jeux ne doivent plus être ajoutés au registre : seuls les jeux ayant au moins une description ou une image doivent y entrer.

## Acceptance Criteria
- [ ] Lors d'un `update`, un jeu sans description et sans image n'est pas ajouté au registre
- [ ] Un jeu ayant au moins une description ou une image continue d'être importé normalement
- [ ] Le résumé affiché (added/updated/unchanged) ne compte pas ces jeux ignorés parmi les jeux ajoutés

## Notes
À trancher : que faire d'un jeu déjà présent dans le registre si une future mise à jour de son `gamelist.xml` local ne contient plus ni description ni image (le retirer du registre, ou conserver l'entrée existante sans la modifier) ?
