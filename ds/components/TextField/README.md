# TextField

A single-line glass input with an optional label, leading icon, hint and error, like the "Email address" field on the plan card.

- **Provide** `label` (or a clear `placeholder` when the label would repeat it), `value`/`onChange` or `defaultValue`, and `type` (`email`, `password`, `number`…). All input attributes pass through.
- `icon` takes a node or a built-in name (`"mail"`, `"search"`); `trailing` holds a unit or icon button.
- `hint` sits under the field in `fg-secondary`; `error` replaces it in `danger`, turns the border `danger` and sets `aria-invalid`. Write errors as the fix: "Enter an email like name@example.com".
- Height 44px, `radius-md`. Use SearchField, not TextField, for search.
