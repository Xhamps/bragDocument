# Button

Glass is the default button; Primary color is for the one call to action a view exists for, and Ghost sits beside it as the alternative.

- **Provide** a short verb-first label as children. Icons: `icon` (leading) and `trailingIcon`, each a node or a built-in name, 16px; `chevron` adds the trailing › when the action navigates ("Subscribe ›").
- **Glass** (`variant="glass"`): the everyday action on a glass container (Browse templates, Download with a download icon, Buy now with a card icon, Add card with a plus-circle).
- **Primary** (`variant="primary"`): one per view; best on background-less surfaces. Hover goes to `button-hover`.
- **Ghost** (`variant="ghost"`): a secondary action next to a primary one.
- **Tinted** (`variant="tinted"`): text-only in `button` for low-emphasis actions inside rows ("See all", "Clear").
- **Gradient** (`variant="gradient"`): the single marketing "Get Started" on `gradient-red-3` with `shadow-cta`. One per page, never in app chrome. White on the peach end is below 4.5:1, so keep it at `lg` size and semibold.
- `pill` rounds it fully (Subscribe, Start a free trial); `fullWidth` stretches it (Learn in a course card); `disabled` renders Inactive; `glow` adds `shadow-glow` once per view.
- Icon placement: a leading icon names the object (card, plus), a trailing icon names the motion (arrow, download, chevron). Never both plus a chevron.
- Sizes: `sm` (28px), `md` (36px), `lg` (44px). For an icon with no text, use IconButton.
