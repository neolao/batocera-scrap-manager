---
status: done
---
# Configuration du Registry et des ROMs

## Description
Le projet doit permettre de configurer un dossier "registry" qui centralise les données de scraping (gamelist et media) déjà collectées, ainsi qu'un ou plusieurs dossiers de ROMs Batocera à surveiller. L'objectif est de pouvoir peupler ce registry à partir des gamelist.xml et des fichiers media déjà scrappés présents dans les dossiers de ROMs, sans avoir à relancer un scraping complet. Cette fonctionnalité constitue la base de toutes les futures opérations de gestion du scraping.

## Acceptance Criteria
- [ ] L'utilisateur peut configurer le chemin d'un dossier "registry" dans lequel les données scrappées seront centralisées
- [ ] L'utilisateur peut configurer un ou plusieurs dossiers de ROMs Batocera à associer au registry
- [ ] Le système détecte et lit les gamelist.xml existants dans chaque dossier de ROMs configuré
- [ ] Le système importe dans le registry les entrées de gamelist et les fichiers media déjà présents, sans dupliquer les entrées déjà importées

## Notes
Le format des gamelist.xml et l'arborescence des dossiers media (images, vidéos, manuels) suivent la convention EmulationStation/Batocera. La structure exacte du registry (format de stockage : fichiers plats, base de données, etc.) reste à définir lors de l'implémentation.
