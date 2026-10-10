# Checkbox

A round check for an independent yes/no choice, as in the baggage filters ("Seat choice included", "No cancel fee").

- **Provide** `label` (or children), `checked`/`onChange` or `defaultChecked`; `description` adds a `footnote` line under it; `disabled` dims it.
- Checked, the circle fills with `button` and draws a `button-fg` check; the label moves from `fg-secondary` to `fg-primary`.
- Stack checkboxes with `space-4` gaps under a `title-3` group heading. For a setting that applies instantly, use Toggle.
