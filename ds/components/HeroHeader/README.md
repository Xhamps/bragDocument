# HeroHeader

A page or section header with a gradient headline, an optional announcement pill, a subtitle and actions, as in the kit's "Power your Finances with AI" hero.

- **Provide** `title` (the words that carry the gradient), optional `lead` (plain words before it), `subtitle` in `fg-secondary`, an `eyebrow` ("Announcing Series B" with `onEyebrowClick` for a chevron) and `actions` (usually one glass and one primary Button).
- `gradient`: `primary` (ink, `tg-primary-*`), `secondary` (indigo, `tg-secondary-*`) or `accent` (`button-text` → `chart-2` → `chart-3`); `none` for a solid title. Every stop clears 3:1 on the page in both themes, so keep gradient text at `title-1` (34px) or larger.
- `size`: `display` for the one hero per page, `title-1` for section headers, `title-2` inside cards. `align`: `center` (heroes) or `left` (sections, dashboards).
- Put it on the mood backdrop (`bd-backdrop`). One gradient headline per view; the rest of the page uses solid `fg-primary` headings.
- Don't put gradient text on body copy, buttons or links, and don't combine it with the `gradient-*` illustration fills behind it.
