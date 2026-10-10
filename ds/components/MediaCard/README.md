# MediaCard

A glass card led by an image: courses, articles, templates, videos and offers, like "Designing a Travel App" with its 3D artwork, tag, progress and author.

- **Provide** `title`, an `image` (URL; or a `gradient-*` token as an artwork placeholder) with `imageAlt` when it carries meaning, and any of `tag`, `eyebrow`, `description` (clamped to 3 lines), `progress` (0–100), `author` `{name, role, avatar}`, `badge` (a duration over the image) and `actions`.
- **layout**:
  - `vertical` (default): image on top inset by `space-2`, 16:9. Grids of courses and templates.
  - `horizontal`: square image on the left. Lists and sidebars.
  - `overlay`: the image fills the card and white text sits on a navy scrim at the bottom, 4:5. One featured item per row; the scrim keeps white text above 4.5:1 whatever the image.
- Interaction: give `href` (the whole card becomes the link, buttons inside still work) or `onClick`; the card lifts with `shadow-lg` and the image zooms 5%. `mediaAction` puts a small IconButton on the image (save, play).
- Use the system's artwork in `assets/Images/` or the `gradient-*` tokens; never stretch an image, it is always cropped with `object-fit: cover`.
- Keep cards in one row the same layout and aspect; `animate` with 60ms `delay` steps for entrance.
