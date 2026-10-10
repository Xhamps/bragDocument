# UserMenu

The signed-in person's button, with their avatar, name and role, that opens an account menu: Account, Billing, Sign out.

- **Provide** `user` `{name, email, role, avatar, status}` (initials when there is no avatar; `status` online/busy/away adds a dot), and optionally `items` (`{label, icon, href, onSelect, tone, shortcut}` or `"separator"`) and `onSelect(label)`. The default items are Account, Billing, a separator and Sign out (in `danger`).
- `variant="pill"` (default) is a glass pill for sidebars and toolbars; `variant="avatar"` is the avatar alone for top-right headers, with a `button-selected` ring while open.
- `placement`: `bottom-end` (default, header right), `bottom-start`, `top-start` (sidebar footer, opens upward) or `top-end`.
- The menu is a heavy-blur glass panel on `shadow-xl` that scales in from its corner with a spring; rows fade in 30ms apart. It closes on a click outside, Escape (focus returns to the button), Tab or choosing an item. Arrow keys, Home and End move between rows.
- Put Sign out last, after a separator, in `danger`. Keep the menu to account actions; app navigation belongs in a MenuList.
