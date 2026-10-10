# SegmentedControl

A pill of two to four mutually exclusive options that switches a view in place (Glass / Outline / Flat, Day / Week / Month, sm / md / lg).

- **Provide** `options` (strings or `{value, label}`), and either `value` + `onChange` or `defaultValue`; give it a `label` for assistive tech.
- `tone="neutral"` raises the chosen option as a glass pill; `tone="accent"` colours the chosen label `button`, used above charts.
- Use `size="sm"` inside cards. More than four options or options that navigate: use a MenuList or tabs instead.
- The selection is one glass pill that slides to the clicked option with a small spring (`duration-slow`, `ease-spring`) and resizes to its width; in `tone="accent"` it is a 2px `button-text` underline that slides instead.
- Arrow keys move the selection; only the chosen option is in the tab order.
