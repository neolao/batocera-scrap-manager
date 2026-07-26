---
status: todo
---
# Serveur Web De Consultation Du Registre

## Description
Le registre ne se consulte aujourd'hui qu'à travers le site HTML statique régénéré par `update` (items 007/008/010), qui ne permet aucune interaction. Une nouvelle commande `serve` doit démarrer un serveur HTTP servant le contenu du registre depuis le programme lui-même : une page listant les jeux groupés par système, et une fiche par jeu affichant ses métadonnées et sa jaquette. Cette commande est le socle des items d'édition qui suivent (015, 016, 017) ; à ce stade elle reste en lecture seule. Le site statique continue d'exister en parallèle, inchangé.

## Acceptance Criteria
- [ ] `serve` démarre un serveur HTTP écoutant sur `0.0.0.0:8080` par défaut, affiche l'adresse d'écoute au démarrage, et `--addr` permet de changer l'adresse et le port
- [ ] La page d'accueil liste tous les jeux du registre groupés par système, avec nom, description courte et jaquette, et permet d'accéder à la fiche d'un jeu
- [ ] La fiche d'un jeu affiche ses métadonnées complètes (description, note, année, développeur, éditeur, genre, nombre de joueurs) et ses médias disponibles
- [ ] Une URL de fiche désignant un système ou un jeu inconnu répond 404 au lieu d'une page vide ou d'une erreur serveur
- [ ] `serve --help` affiche l'aide propre à la commande sans exiger de registre configuré, et `serve` sans registre configuré affiche le message d'erreur habituel

## Notes
Un jeu est identifié dans les URLs par la même clé que celle utilisée sur disque : le nom de base du ROM sans extension (`gameID` dans `internal/registry/registry.go`), pour ne pas introduire une seconde règle de correspondance (cf. `decisions/014`). Les médias sont servis depuis le dossier du registre via un `http.FileServer` (qui neutralise le path traversal). La mise en forme et les helpers de formatage du site statique (`groupBySystem`, `formatStars`, `formatYear`, `escapeMediaPath` dans `internal/site/site.go`) doivent être réutilisés plutôt que dupliqués. Nouveau package `internal/webui` pour les handlers et templates, commande dans `internal/cli/serve.go` suivant le pattern des commandes existantes (`--help` traité en premier, `loadConfigAndRegistry`). Tests via `net/http/httptest`, sans ouvrir de port.
