---
status: done
depends_on: [014]
---
# Édition Des Métadonnées Depuis La Web UI

## Description
Quand un jeu a été mal scrapé (nom approximatif, description tronquée, genre absent), la seule façon de corriger le registre est aujourd'hui d'éditer le `gamelist.xml` source puis de relancer `update`. La fiche d'un jeu servie par la web UI doit proposer un formulaire pré-rempli permettant de corriger ses métadonnées texte et de les enregistrer directement dans le registre. Après enregistrement, le site statique `index.html` est régénéré pour rester cohérent avec le registre.

## Acceptance Criteria
- [ ] La fiche d'un jeu affiche un formulaire pré-rempli avec ses champs texte éditables : nom, description, note, date de sortie, développeur, éditeur, genre, nombre de joueurs
- [ ] Après enregistrement, la valeur modifiée est visible au rechargement de la fiche et présente dans le fichier JSON du jeu dans le dossier du registre
- [ ] Le chemin du ROM et les champs de médias du jeu ne sont jamais modifiés par ce formulaire
- [ ] Le site statique `index.html` est régénéré après un enregistrement, et reflète la nouvelle valeur
- [ ] L'enregistrement n'est accepté qu'en `POST` (une requête `GET` ne modifie rien) et enregistrer sur un jeu inconnu répond 404

## Notes
Côté domaine, ajouter dans `internal/registry` une fonction qui localise l'entrée via `indexOf` et applique uniquement les champs texte éditables, renvoyant `ErrGameNotFound` sinon — même convention d'erreur que `Remove`. La persistance réutilise la séquence `registry.Save` + `site.Generate` déjà appliquée par `saveAndGenerateSite` (`internal/cli/common.go`), à rendre appelable depuis `internal/webui` sans dépendre du package `cli`. Après écriture, rediriger en `303` vers la fiche pour éviter un renvoi du formulaire au rafraîchissement.
