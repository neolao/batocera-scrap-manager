---
status: done
---
# Régénération Du Site Après Un Remove En CLI

## Description
La commande `remove <system> <rom-filename>` supprime bien l'entrée du registre et ses médias du dossier, mais ne régénère pas `index.html` — contrairement à `update`, et contrairement à la suppression depuis la web UI (item 016). Le site statique continue donc d'afficher le jeu supprimé, avec une jaquette pointant vers un fichier qui n'existe plus, jusqu'au prochain `update`. C'est un comportement préexistant, découvert en réalisant l'item 016.

## Acceptance Criteria
- [ ] Après un `remove` réussi, le site statique `index.html` ne contient plus le jeu supprimé, sans avoir à lancer `update`
- [ ] Si le registre a bien été modifié mais que le site n'a pas pu être régénéré, la commande dit ce qui a été supprimé et signale que le site est obsolète, plutôt que de laisser croire que rien n'a été fait
- [ ] Un `remove` qui échoue (jeu inconnu) ne régénère pas le site et ne change rien

## Notes
`internal/cli/common.go` a déjà `saveAndGenerateSite`, un mince wrapper sur `store.Save` que `update` et `scrape` utilisent — `runRemove` est le seul à ne pas passer par là. Attention à l'ordre : `registry.Remove` efface déjà les fichiers du jeu avant que `store.Save` n'intervienne, donc la suppression est acquise dès ce moment-là ; un échec de la régénération du site ne doit pas être présenté comme un échec de la suppression (même raisonnement que [`decisions/022`](../decisions/022-a-deletion-is-committed-when-the-game-file-is-gone.md), et le sentinel `store.ErrSiteNotRegenerated` existe déjà pour distinguer ce cas). Voir aussi `registry.ErrMediaLeftBehind`, que `runRemove` traite déjà comme un avertissement et non comme une erreur.
