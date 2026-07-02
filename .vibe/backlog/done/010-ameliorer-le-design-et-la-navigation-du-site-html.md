---
status: done
---
# Améliorer le Design et la Navigation du Site HTML

## Description
Le site HTML de consultation du registre (item 007) est aujourd'hui du HTML brut, sans mise en forme ni moyen de circuler rapidement entre les systèmes. Le rendu doit être amélioré visuellement (mise en page cohérente pour chaque jeu) et doter la page d'une navigation permettant de se déplacer facilement entre les différents systèmes.

## Acceptance Criteria
- [x] Le site affiche un sommaire ou une barre de navigation permettant d'accéder directement à un système donné sans faire défiler toute la page
- [x] Chaque jeu est présenté avec une mise en page cohérente (jaquette, nom, description alignés) grâce à une feuille de style appliquée, plutôt qu'un HTML sans mise en forme
- [x] Un lien permet de revenir en haut de page ou au sommaire des systèmes depuis n'importe quelle section
- [x] La page reste lisible sur un petit écran, sans contenu qui déborde

## Notes
Raffinement visuel et ergonomique de la fonctionnalité livrée par l'item 007. Le style pourra être embarqué directement dans le fichier `index.html` généré (pas de fichier CSS externe), puisque le site reste un artefact statique auto-suffisant.
