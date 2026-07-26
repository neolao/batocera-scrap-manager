---
status: todo
depends_on: [014]
---
# Gestion Des Médias Depuis La Web UI

## Description
Une jaquette manquante ou de mauvaise qualité ne peut pas être corrigée aujourd'hui : le registre ne reçoit que les médias copiés depuis les dossiers ROMs lors d'un `update`. Depuis la fiche d'un jeu, la web UI doit permettre d'envoyer un fichier pour remplacer un média (jaquette, vidéo, marquee, thumbnail) et de supprimer un média existant, le registre étant mis à jour en conséquence.

## Acceptance Criteria
- [ ] La fiche d'un jeu permet d'envoyer un fichier pour chacun des quatre médias (jaquette, vidéo, marquee, thumbnail) ; après envoi, le média est affiché sur la fiche et le champ correspondant du jeu pointe vers un fichier réellement présent dans le dossier du registre
- [ ] La fiche permet de supprimer un média existant : le fichier disparaît du dossier du registre et le champ correspondant du jeu est vidé
- [ ] Un fichier dont l'extension n'est pas autorisée pour le type de média visé est refusé avec un message d'erreur explicite, sans rien modifier dans le registre
- [ ] Un envoi dépassant la taille maximale autorisée est refusé avec un message explicite, sans rien modifier dans le registre
- [ ] Le site statique `index.html` reflète l'ajout ou la suppression d'un média après l'opération

## Notes
Le nom du fichier stocké dans le registre doit être dérivé de l'identifiant du jeu (`gameID`) et de l'extension validée, jamais du nom de fichier fourni par le client, pour éviter tout path traversal ou écrasement d'un autre jeu. Plafonner la taille des envois (`http.MaxBytesReader`) et n'autoriser qu'une liste d'extensions par type de média. Côté domaine, ajouter dans `internal/registry` l'écriture et la suppression d'un média d'une entrée (réutiliser `removeIfExists` et la table `mediaFields`), puis persister via la même séquence `registry.Save` + `site.Generate` que les autres modifications.
