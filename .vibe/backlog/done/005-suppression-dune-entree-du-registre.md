---
status: done
---
# Suppression d'une Entrée du Registre

## Description
Le projet doit exposer une commande CLI permettant de retirer un jeu précis (un "scrap") du registre, avec toutes ses données associées : sa fiche de metadata et ses fichiers médias (jaquette, vidéo, marquee, thumbnail) déjà copiés dans le registre. Cela permet de nettoyer une entrée obsolète, incorrecte, ou correspondant à un jeu retiré des dossiers de ROMs, sans avoir à manipuler le registre à la main sur le disque.

## Acceptance Criteria
- [ ] L'utilisateur peut lancer une commande CLI dédiée en désignant un jeu précis (par système et chemin de ROM) à retirer du registre
- [ ] Le système supprime la fiche de metadata et l'ensemble des fichiers médias associés à ce jeu dans le registre
- [ ] Le système confirme la suppression effectuée à l'utilisateur
- [ ] Une tentative de suppression d'un jeu absent du registre renvoie une erreur claire, sans modifier le reste du registre

## Notes
Reste à définir : comment l'utilisateur désigne le jeu à supprimer (système + chemin de ROM exact, nom, ou sélection interactive). Aucune dépendance bloquante identifiée : le registre (items 001-002) est déjà en place.
