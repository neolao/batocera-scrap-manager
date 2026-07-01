---
status: todo
depends_on: [001]
---
# Commande CLI de Mise à Jour du Registre

## Description
Le projet doit exposer une commande CLI permettant de mettre à jour le registre à partir des dossiers de ROMs Batocera configurés. Cette commande parcourt chaque dossier de ROMs, détecte les changements dans les gamelist.xml et les fichiers media, et synchronise ces informations dans le registry. Elle constitue le point d'entrée principal pour maintenir le registre à jour après un nouveau scraping ou l'ajout de ROMs.

## Acceptance Criteria
- [ ] L'utilisateur peut lancer une commande CLI dédiée (ex. `batocera-scrap-manager update`) pour mettre à jour le registre à partir des dossiers de ROMs configurés
- [ ] Le système parcourt chaque dossier de ROMs configuré et met à jour le registre avec les nouvelles entrées de gamelist et media détectées
- [ ] Le système affiche un résumé (nombre d'entrées ajoutées, mises à jour, inchangées) à la fin de l'exécution
- [ ] Le système retourne un code de sortie non nul en cas d'erreur (dossier de ROMs introuvable, registry non configuré, etc.)

## Notes
Dépend de l'item 001 (configuration du registry et des dossiers de ROMs), qui doit être implémenté au préalable pour fournir les chemins nécessaires à cette commande.
