---
status: done
depends_on: [015]
---
# Protection De Tous Les Champs D'un Jeu

## Description
Aujourd'hui la protection contre le ré-écrasement par `update` se pose champ par champ, et uniquement comme effet d'une correction dans la web UI : seuls les champs dont la valeur change réellement au Save sont marqués dans `manual_fields` (décision 017). Il n'existe aucun moyen de dire « ce jeu est bon, n'y touche plus » sans modifier artificiellement chacun de ses champs. Il faut pouvoir protéger un jeu entier — c'est-à-dire marquer d'un coup tous ses champs éditables — depuis la ligne de commande et depuis la fiche du jeu dans la web UI, ainsi que lever cette protection.

## Acceptance Criteria
- [ ] Une commande CLI protège un jeu désigné par son système et son fichier ROM : tous ses champs éditables se retrouvent dans `manual_fields`, sans qu'aucune de leurs valeurs ne change
- [ ] La même commande sait lever la protection : `manual_fields` redevient vide et le jeu est de nouveau rafraîchi par `update`
- [ ] Après protection puis `update` sur un dossier ROMs dont le `gamelist.xml` porte des valeurs différentes, aucun champ éditable du jeu n'est modifié dans le registre
- [ ] La fiche du jeu dans la web UI propose la protection et sa levée, indique l'état courant, et le changement prend effet sans redémarrer `serve`
- [ ] Protéger un jeu inconnu répond `ErrGameNotFound` côté CLI et 404 côté web, sans rien modifier dans le registre

## Notes
La logique appartient au registre, pas aux interfaces : ajouter dans `internal/registry/metadata.go` de quoi marquer/démarquer l'ensemble de `editableFields`, réutilisé tel quel par la commande et par le handler web — un garde placé dans la web UI serait contourné par la CLI (même raisonnement que la décision 017). La protection totale doit rester compatible avec la protection par champ existante : elle en est le cas limite, pas un second mécanisme, et la case « hand back » du formulaire d'édition doit continuer de fonctionner sur un jeu protégé en entier.

Question ouverte : forme de la commande — `protect <system> <rom-filename>` avec un `unprotect` symétrique, ou une commande unique avec un drapeau. Suivre la convention des commandes existantes (`remove <system> <rom-filename>`) et fournir un `--help` comme l'exige l'item 013.

À noter aussi pour l'aide et la doc : protéger un jeu n'empêche pas Batocera d'afficher la valeur du `gamelist.xml` du dossier ROMs, qui n'est pas touché.
