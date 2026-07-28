---
status: todo
depends_on: [014]
---
# Complétion Des Dossiers De ROMs Depuis La Web UI

## Description
Une fois les métadonnées corrigées dans la web UI, les renvoyer vers Batocera impose de quitter l'interface et de lancer `scrape` en ligne de commande. La web UI doit pouvoir déclencher cette même complétion des dossiers de ROMs configurés — réécriture des `gamelist.xml` et recopie des médias depuis le registre — et en afficher le résultat. C'est le pendant, dans le sens registre → ROMs, de l'item 022 qui traite le sens inverse.

## Acceptance Criteria
- [ ] Avant toute écriture, une page de confirmation nomme les dossiers de ROMs qui seront modifiés et offre un moyen d'annuler qui ne touche à rien
- [ ] Après confirmation, les `gamelist.xml` et les médias des dossiers configurés sont complétés depuis le registre, et une page affiche le même résumé que le CLI
- [ ] Un dossier de ROMs configuré introuvable produit un message d'erreur nommant ce dossier, et le rapport affiché reflète fidèlement les dossiers déjà traités avant l'échec
- [ ] L'opération ne modifie aucun fichier du dossier du registre : les JSON des jeux et le site statique sont inchangés après son exécution

## Notes
Contrairement à toutes les routes mutantes existantes, celle-ci écrit **en dehors** du registre, dans les dossiers Batocera de l'utilisateur, en réécrivant des `gamelist.xml` qui ne sont pas versionnés — d'où la page de confirmation préalable, sur le modèle du flux de suppression (`internal/webui/delete.go`, [`decisions/018`](../decisions/018-metadata-editing-on-its-own-page-with-post-redirect-get.md)). Côté domaine, `registry.CompleteRomsFolder` existe déjà et lit le registre sans jamais l'écrire : le verrou de lecture suffit, aucun `Clone()` ni `internal/store` n'est nécessaire, et rien ne doit basculer l'instantané servi. Comme pour l'item 022, `webui.Handler` devra recevoir les dossiers de ROMs configurés, et la durée de l'opération pose la même question ouverte (synchrone avec page de résultat, ou tâche de fond avec page d'état) — à trancher de la même façon que 022 pour que les deux commandes se ressemblent. Même contrôle `crossSite` et même schéma `POST` + `303` que les autres routes mutantes.
