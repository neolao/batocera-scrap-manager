# CLAUDE.md — batocera-scrap-manager

> Auto-généré par /vibe:init. Modifiable librement — relance /vibe:init pour régénérer.
> Ajoute `<!-- keep -->` sur un titre de section pour le préserver lors d'une régénération.

## Aperçu du projet

Outil en ligne de commande pour gérer les données de scraping (métadonnées, jaquettes, etc.) des jeux sur Batocera.

**Stack :** Go 1.26 (module `github.com/neolao/batocera-scrap-manager`)
**Type :** CLI

## Architecture

```
batocera-scrap-manager/
├── main.go            # point d'entrée, délègue à internal/cli
├── internal/
│   └── cli/            # logique de la CLI (parsing, commandes, sorties)
└── go.mod
```

> Lance `/vibe:sync` pour générer `.vibe/` — cartographie détaillée des modules, modèles de données et glossaire du langage omniprésent, à l'usage de Claude.

## Workflow de développement (Vibe Coding)

L'utilisateur est **uniquement Product Owner**. Il décrit les besoins et évalue les résultats — il n'écrit ni ne teste manuellement le code.

### Definition of Done

Une tâche est terminée UNIQUEMENT si TOUS les points suivants sont vrais :

- [ ] Le code d'implémentation est écrit
- [ ] Des tests couvrant le chemin nominal existent et passent
- [ ] Des tests couvrant les cas limites et les chemins d'erreur existent et passent
- [ ] `gofmt -l .` ne retourne aucun fichier (formatage propre)
- [ ] `go vet ./...` ne remonte aucune alerte
- [ ] `go test ./...` passe intégralement
- [ ] Aucun artefact de debug laissé dans le code (fmt.Println de debug, etc.)

**Ne jamais présenter un résultat à l'utilisateur si les tests échouent.**

### Workflow TDD

Pour chaque fonctionnalité ou correction :

1. **Écrire les tests d'abord** — décrire le comportement attendu via des tests avant d'écrire l'implémentation
2. **Confirmer que les tests échouent** — lancer `go test ./...` et vérifier que les nouveaux tests échouent (rouge)
3. **Écrire l'implémentation** — code minimal pour faire passer les tests
4. **Confirmer que les tests passent** — relancer `go test ./...` et vérifier que tout passe (vert)
5. **Refactorer si nécessaire** — nettoyer en gardant les tests au vert
6. **Lancer le lint** — `gofmt -l .` et `go vet ./...`, corriger tout écart
7. **Présenter le résultat** — résumer ce qui a été fait, ce qui a été testé, et la sortie des tests

### Workflow de correction de bug

1. **Reproduire d'abord dans un test** — écrire un test qui échoue et qui capture le bug
2. Corriger le bug
3. Confirmer que le test passe
4. Présenter le résultat avec le nom du test qui couvre désormais le bug

### Boucle d'auto-correction

Si les tests ou le lint échouent :
- Ne PAS demander de l'aide à l'utilisateur
- Diagnostiquer l'échec, corriger, relancer
- Répéter jusqu'au vert (3 tentatives d'auto-correction maximum)
- N'escalader vers l'utilisateur que si l'échec révèle une exigence ambiguë

## Conventions de test

- Emplacement des tests : fichiers `*_test.go` colocalisés avec le code (ex. `internal/cli/cli_test.go`)
- Framework de test : `testing` (bibliothèque standard Go)
- Un fichier de test par fichier source
- Les noms de tests doivent décrire un comportement : `TestExecute_UnknownCommand_ReturnsErrorCode` plutôt que `TestExecute2`
- Toujours couvrir : chemin nominal + au moins 2 cas limites + entrée invalide/erreur

## Contraintes

- Jamais de code mort ni d'imports inutilisés
- Jamais de secret en dur — utiliser des variables d'environnement
- Ne jamais ignorer un test pour tenir un délai — corriger le code à la place
- Le style est imposé par l'outillage, pas par convention — toujours lancer `gofmt -l .` et `go vet ./...` avant de présenter un résultat

## Agents de revue

Agents actifs pour `/vibe:review` sur ce projet :

| Agent | Actif | Raison |
|---|---|---|
| `review-coverage` | ✅ | toujours actif |
| `review-naming` | ✅ | toujours actif |
| `review-complexity` | ✅ | toujours actif |
| `review-solid` | ✅ | architecture modulaire par packages (`internal/cli`), amenée à grandir |
| `review-ddd` | ❌ | pas de couche domaine explicite (pas de `domain/`, `entities/`, etc.) |
