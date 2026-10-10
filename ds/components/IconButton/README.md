# IconButton

A button that is only an icon, with a required accessible `label` that is also shown as its Tooltip, like the moon/sun theme switch, the pager arrows, the play discs and the glowing + diamond.

- **Provide** `icon` (a node or a built-in name) and `label` ("Next card", "Dark mode"). The tooltip defaults to the label; pass `tooltip` to change it, `shortcut` to add a key cap, or `tooltip={false}` when a visible label sits next to it.
- `variant`: `glass` (default), `primary`, `ghost`, `tinted` (bare icon in a toolbar) or `gradient`.
- `shape`: `circle` (default), `square` (`radius-md`, toolbars) or `diamond` (the floating + action, pair with `glow`).
- `size`: `sm` 28, `md` 36, `lg` 44, `xl` 64 (media play).
- `pressed` makes it a toggle (theme, visibility eye): the pressed state shows a `button-selected` ring and `button` icon.
- `badge`: `true` for an unread dot, a number for a count.
- Don't use an icon whose meaning isn't obvious without its tooltip for a primary action; give it a text Button instead.
