# PageHeader

The heading block at the top of a page's content: breadcrumbs, a title with optional tags and gradient, a subtitle, a meta row, actions on the right, and view tabs underneath.

- **Provide** `title` and any of `breadcrumbs` (`{label, href}`; the last is the current page), `eyebrow` (Tags), `subtitle`, `meta` (dates, owners, counts), `actions` (one primary Button plus glass ones) and `tabs` (a SegmentedControl).
- The title is `title-1` (34px). `gradient` applies a text gradient; keep it for top-level pages only.
- Place it directly under the TopBar, on the backdrop, aligned to the content grid. Actions wrap under the title on narrow screens.
