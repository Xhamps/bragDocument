# Select

A pill-shaped glass dropdown on the native `<select>`, as in "Corner Radius: 10 ⌄" and "Sort by ⌄".

- **Provide** `options` (strings or `{value, label}`), `value`/`onChange` or `defaultValue`; `prefix` puts a lead-in inside the pill ("Corner Radius:"); `placeholder` adds a disabled first option ("Sort by").
- `label`, `hint` and `error` behave as in TextField.
- Use it for 5+ choices or choices that don't fit a SegmentedControl. The menu is the platform's own, so it stays accessible on every device.
