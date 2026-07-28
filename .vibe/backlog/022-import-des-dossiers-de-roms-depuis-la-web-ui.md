---
status: todo
depends_on: [014]
---
# Import Des Dossiers De ROMs Depuis La Web UI

## Description
Alimenter le registre depuis les dossiers de ROMs configurés impose aujourd'hui de quitter la web UI, de lancer `update` en ligne de commande, puis de redémarrer le serveur — l'instantané servi n'étant lu qu'au démarrage. La web UI doit pouvoir déclencher ce même import, afficher son résultat, et rafraîchir immédiatement les pages servies. C'est le pendant, dans le sens ROMs → registre, de l'item 023 qui traite le sens inverse.

## Acceptance Criteria
- [ ] La web UI offre un contrôle déclenchant l'import de tous les dossiers de ROMs configurés ; à l'issue de l'opération, une page affiche le même résumé que le CLI (« N added, N updated, N unchanged »)
- [ ] Après un import réussi, les pages servies (accueil, page système, fiche d'un jeu) listent les entrées nouvellement importées sans redémarrage du serveur, et le site statique du registre est régénéré
- [ ] Un dossier de ROMs configuré introuvable produit un message d'erreur nommant ce dossier, et l'instantané servi reste cohérent avec ce que contient le registre sur disque
- [ ] Aucun dossier de ROMs configuré n'est un cas valide, pas une erreur : la page affiche un résumé à zéro accompagné d'un message explicite

## Notes
`webui.Handler(reg, registryFolder)` ne connaît pas la configuration : les dossiers de ROMs devront lui être transmis, sans dupliquer la lecture de `internal/config` déjà faite par la commande `serve`. L'import est une opération longue et batch (`registry.ImportFromRomsFolder` avec son callback `ProgressEvent`), alors que toutes les routes mutantes actuelles sont instantanées : garder le verrou d'écriture pendant toute la durée bloquerait aussi les lectures. Question ouverte à trancher à l'implémentation : exécution synchrone rendant une page de résultat, ou exécution en tâche de fond avec une page d'état — sachant que la web UI n'embarque pas de JavaScript, ce qui exclut toute barre de progression côté client et rend le rafraîchissement méta ou le rechargement manuel les seules options. Réutiliser la discipline en place pour les changements du registre : appliquer sur un `Clone()`, persister via `internal/store`, ne basculer l'instantané servi qu'une fois l'écriture réussie, et traiter l'échec de régénération du site comme une réserve et non comme un échec. La route doit être un `POST` avec le même contrôle `crossSite` que l'édition et la suppression, suivi d'un `303`.
