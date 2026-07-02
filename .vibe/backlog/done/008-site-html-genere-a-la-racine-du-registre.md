---
status: done
---
# Site HTML Généré à la Racine du Registre

## Description
Actuellement, le site HTML de consultation du registre est généré dans un sous-dossier `site/` (`<registre>/site/index.html`). Ce fichier `index.html` doit plutôt être généré directement à la racine du dossier du registre (`<registre>/index.html`), pour être plus facile à trouver et à ouvrir.

## Acceptance Criteria
- [ ] Après un `update`, le fichier `index.html` généré se trouve directement à la racine du dossier du registre, et non plus dans un sous-dossier `site/`
- [ ] Les jaquettes affichées dans la page restent correctement accessibles depuis ce nouvel emplacement (chemins relatifs vers les médias ajustés en conséquence)
- [ ] Un registre vide génère toujours une page `index.html` valide à la racine, avec le message indiquant qu'il n'y a pas encore de jeux

## Notes
Modifie l'emplacement choisi lors de l'implémentation initiale du site (item 007, voir décision `.vibe/decisions/006`), qui plaçait le fichier dans un sous-dossier `site/`. À trancher : que faire de l'ancien sous-dossier `site/` s'il existe encore d'un précédent `update` (le supprimer, le laisser tel quel) ?
