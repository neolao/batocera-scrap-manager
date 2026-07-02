---
status: done
depends_on: [001, 002]
---
# Commande CLI de Complétion des Metadata

## Description
Le projet doit exposer une commande CLI permettant de compléter les metadata manquantes des ROMs référencées dans le registre (nom, description, année, genre, note, jaquette, etc.). Cette commande identifie les ROMs pour lesquelles le registre ne possède pas encore d'entrée gamelist ou de media complets, puis récupère les informations manquantes via un scraping externe. Elle permet ainsi de combler les trous laissés par la synchronisation des données déjà scrappées, pour les jeux jamais scrappés auparavant.

## Acceptance Criteria
- [ ] L'utilisateur peut lancer une commande CLI dédiée (ex. `batocera-scrap-manager scrape`) pour compléter les metadata manquantes des ROMs du registre
- [ ] Le système identifie automatiquement, à partir du registre, les ROMs dont la metadata (gamelist) ou les media sont absents ou incomplets
- [ ] Le système récupère les metadata manquantes via une source de scraping et met à jour le registre en conséquence
- [ ] Le système affiche un résumé (nombre de ROMs traitées, complétées avec succès, en échec) à la fin de l'exécution

## Notes
Dépend de l'item 001 (configuration du registry et des ROMs) et de l'item 002 (mise à jour du registre à partir des dossiers de ROMs), qui doivent être implémentés au préalable pour que le registre reflète l'état actuel des ROMs avant de chercher les metadata manquantes. La source de scraping externe à utiliser (API, service, format d'échange) reste à définir.
