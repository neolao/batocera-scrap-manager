---
status: done
---
# Génération d'un Site HTML de Consultation du Registre

## Description
En plus des fichiers de metadata et de médias déjà stockés, le registre doit aussi générer un petit site HTML statique permettant de consulter facilement son contenu dans un navigateur, sans avoir à ouvrir les fichiers JSON un par un. Ce site présente les jeux organisés par système, avec leurs informations principales (nom, description, jaquette, etc.), pour donner une vue d'ensemble rapide et lisible du registre.

## Acceptance Criteria
- [ ] Après une mise à jour ou une complétion du registre, un site HTML est généré (ou régénéré) dans le dossier du registre
- [ ] L'utilisateur peut ouvrir ce site dans un navigateur et voir la liste des jeux regroupés par système
- [ ] Pour chaque jeu, le site affiche au minimum son nom, sa description et sa jaquette (quand elle est disponible)
- [ ] Un système sans jeu, ou un jeu sans jaquette, s'affiche correctement sans erreur ni image cassée

## Notes
Reste à définir : à quel(s) moment(s) le site est (re)généré (à chaque commande `update`/`scrape`, ou via une commande dédiée), et le niveau de détail affiché par jeu (uniquement les champs principaux, ou toutes les metadata disponibles).
