---
status: done
---
# Improve The HTML Site Design And Navigation

## Description
The HTML site browsing the registry (item 007) is currently raw HTML, with no styling and no way to move quickly between systems. Its rendering must be improved visually (a consistent layout for every game) and the page must gain a navigation making it easy to move between the different systems.

## Acceptance Criteria
- [x] The site displays a table of contents or a navigation bar giving direct access to a given system without scrolling through the whole page
- [x] Every game is presented with a consistent layout (cover art, name, description aligned) through an applied stylesheet, rather than unstyled HTML
- [x] A link makes it possible to go back to the top of the page or to the systems' table of contents from any section
- [x] The page stays readable on a small screen, with no content overflowing

## Notes
Visual and ergonomic refinement of the feature delivered by item 007. The style may be embedded directly in the generated `index.html` file (no external CSS file), since the site remains a self-contained static artifact.
