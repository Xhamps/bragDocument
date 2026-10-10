# Card

The glass container every surface is built from: `container-bg`, `container-border`, `shadow-glass`, `radius-lg`, padded `space-6`.

- **Provide** children, or `title` (in `title-3`) and `description` (in `body`, `fg-secondary`) for a simple card; `compact` pads `space-4`.
- Always place it over the mood backdrop (`bd-backdrop`) so the blur has something to absorb.
- Nest smaller glass (rows, buttons) inside it at `radius-md`; don't nest a Card in a Card.
- `tint` layers a transparent gradient over the glass: `sheen` (default, a white light catch from the top-left, `glass-sheen`), `accent` (blue to violet), `violet` (a corner glow), `rose` (warm toward the bottom), `cool` (teal to blue), or `none` for flat glass. Tints use the `glass-tint-*` tokens and stay faint enough that `fg-primary` and `fg-secondary` keep their contrast.
- Use tints to group or highlight, not decorate every card: one tinted card per row (the featured plan, the active course), the rest on `sheen`.
- `interactive` (or `onClick`) lifts the card 4px with `shadow-lg` and sweeps a sheen across it on hover; it presses back to `shadow-md`. Only for cards that open something.
- `elevation` picks a step of the shadow scale (`sm` → `xl`) for the resting state, e.g. `xl` for a floating card over busy content.
- `animate` plays the `bd-enter` fade-and-rise; give a grid `delay` steps of 60ms (0, 60, 120…) so cards arrive in order.
