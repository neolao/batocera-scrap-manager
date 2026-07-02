---
status: done
---
# Affichage du Dossier de ROMs pendant le Scrape

## Description
Pendant l'exécution de la commande `scrape`, l'utilisateur voit actuellement quel système et quel jeu sont en train d'être complétés, mais pas dans quel dossier de ROMs (parmi ceux configurés) la modification est effectuée. Lorsque plusieurs dossiers de ROMs sont configurés, cette information manque pour bien situer où chaque changement a lieu. La commande doit donc aussi indiquer, dans son affichage en direct, quel dossier de ROMs est en cours de mise à jour.

## Acceptance Criteria
- [ ] Quand un seul dossier de ROMs est configuré, l'utilisateur voit tout de même clairement quel dossier est mis à jour pendant le scrape
- [ ] Quand plusieurs dossiers de ROMs sont configurés, l'affichage en direct précise sans ambiguïté à quel dossier appartient chaque changement affiché
- [ ] Le résumé final (traité / complété / en échec) reste inchangé dans son format

## Notes
S'appuie sur la fonctionnalité de complétion déjà en place (commande `scrape`, voir items 003 et la révision de l'affichage filtré). Reste à définir : le dossier est-il affiché une fois par dossier (comme l'en-tête par système), ou répété sur chaque ligne de jeu.
