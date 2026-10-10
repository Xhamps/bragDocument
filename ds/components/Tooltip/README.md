# Tooltip

A small dark label that appears above (or beside) a control on hover and keyboard focus, naming what it does.

- **Provide** one focusable child and `content`; `placement` is `top` (default), `bottom`, `left` or `right`; `shortcut` adds a key cap ("⌘K").
- It inverts the theme: `fg-primary` fill with `page` text, `radius-sm`, `footnote` size, so it reads on any backdrop.
- Escape closes it; it is linked with `aria-describedby`. Keep it to a few words, no punctuation, never interactive content or essential information.
- Every IconButton already carries one.
