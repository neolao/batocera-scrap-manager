---
status: todo
depends_on: [015]
---
# Affichage Et Édition Du Chemin De La ROM

## Description
Dans un `gamelist.xml` Batocera, un jeu est identifié par son chemin de ROM — nom de fichier et éventuel sous-dossier — et non par son nom affiché. Ce chemin n'apparaît nulle part dans la web UI aujourd'hui : la fiche ne montre que les métadonnées, et la correction de métadonnées exclut délibérément le champ `Path`. Il faut pouvoir lire ce chemin sur la fiche d'un jeu et le corriger depuis le formulaire d'édition, le registre se réorganisant en conséquence.

## Acceptance Criteria
- [ ] La fiche d'un jeu affiche le chemin de la ROM tel qu'il est stocké (relatif au dossier système, sous-dossier inclus), et le formulaire d'édition le présente dans un champ modifiable pré-rempli avec cette même valeur
- [ ] Après enregistrement d'un nouveau chemin, l'entrée est stockée sous le fichier JSON dérivé de ce chemin (l'ancien fichier n'existe plus), et la fiche du jeu reste atteignable à l'URL correspondant au nouvel identifiant
- [ ] Un chemin refusé — vide, absolu, ou sortant du dossier système via `..` — produit un message d'erreur explicite et laisse le registre strictement inchangé
- [ ] Un chemin dont l'identifiant dérivé est déjà celui d'un autre jeu du même système est refusé avec un message explicite, sans écraser l'entrée existante

## Notes
Le chemin n'est pas une métadonnée de plus : `registry.GameID` (`filepath.Base` sans extension) en dérive à la fois le nom du fichier JSON de stockage, la clé de déduplication, la clé d'URL de la web UI, et le critère de correspondance avec les jeux du `gamelist.xml` (`internal/registry/registry.go`). Le modifier implique donc de déplacer le fichier du jeu dans le dossier du registre et de rediriger vers la nouvelle URL, en respectant la règle « écrire puis basculer » : ne remplacer l'instantané servi qu'une fois l'écriture réussie. Le champ est absent de `editableFields` (`internal/registry/metadata.go`) par choix — l'y ajouter tel quel lui donnerait aussi la mécanique `ManualFields`, ce qui n'a pas le même sens pour une identité que pour une valeur ; question ouverte à trancher à l'implémentation. Autre question ouverte : un chemin corrigé ne correspondra plus au fichier réellement présent dans le dossier ROMs, et un `update` ultérieur risque de recréer une entrée pour l'ancien nom — décider si l'opération doit renommer les médias associés et/ou avertir l'utilisateur. Comme toute modification du registre, persister via `internal/store` pour que le site statique soit régénéré.
