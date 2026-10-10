# Elevation

The shadow scale and the motion it pairs with: a reference page, not a component to import.

- Rest on `shadow-glass` (containers) or `shadow-button` (glass buttons); hover raises one step; press drops to `shadow-inset`.
- Steps: `shadow-sm` tags and resting rows; `shadow-md` hovered controls, selected segment, dropdowns; `shadow-lg` hovered interactive cards, tooltips; `shadow-xl` dialogs and floating cards.
- Pair each lift with `duration-base` (controls) or `duration-slow` (cards) on `ease-standard`; settle small objects with `ease-spring`.
- Use the `bd-elev-sm|md|lg|xl` classes, the Card `elevation` prop, or the tokens directly in your own CSS.
