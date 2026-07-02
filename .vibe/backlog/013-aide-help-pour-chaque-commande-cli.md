---
status: in_progress
---
# Aide --help pour Chaque Commande CLI

## Description
Aujourd'hui, seul `batocera-scrap-manager --help` existe et se contente de lister les commandes disponibles. Chaque sous-commande (`config`, `update`, `scrape`, `remove`) doit accepter sa propre option `--help`, affichant un message d'usage détaillé et spécifique à cette sous-commande (arguments attendus, options, exemples), pour que l'utilisateur n'ait pas à deviner la syntaxe exacte.

## Acceptance Criteria
- [ ] L'utilisateur peut lancer `batocera-scrap-manager config --help`, `update --help`, `scrape --help` et `remove --help`, et obtient un message d'usage propre à cette commande (et non le message générique global)
- [ ] Chaque message d'aide décrit les arguments/sous-commandes attendus (ex : `remove --help` mentionne `<system> <rom-filename>`)
- [ ] Lancer `<commande> --help` retourne un code de sortie 0, sans exécuter l'action réelle de la commande
- [ ] Le comportement existant de `batocera-scrap-manager --help` (sans sous-commande) reste inchangé

## Notes
Plusieurs sous-commandes (`remove`) affichent déjà un message d'usage en cas d'arguments manquants (`removeUsage`) ; il pourrait être réutilisé pour `--help`. À trancher : ce message doit-il être affiché aussi automatiquement en cas d'erreur d'arguments (comportement déjà en place pour `remove`), ou uniquement sur demande explicite via `--help` ?
