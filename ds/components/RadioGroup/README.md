# RadioGroup

A set of mutually exclusive options with ring-and-dot radios, as in Stops ("Any number of stops", "Nonstop only") and trip type ("Roundtrip", "One way", "Multi-City").

- **Provide** `options` (strings or `{value, label, disabled}`), `value`/`onChange` or `defaultValue`, and a `label` for the group.
- `direction="row"` lays options side by side (trip type); the default column suits filters.
- Selected: a `button` ring with a `button` dot. Always preselect a sensible default.
- Two to five options; more than five, use Select.
