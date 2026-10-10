# Tag

A small pill that labels a status, category or reward, as in "UI/UX DESIGN", "Trial term" and "XP + 10".

- **Provide** one or two words as children, a `variant` and a `tone` (`neutral`, `accent`, `success`, `danger`, `purple`, `teal`, `violet`).
- **dot** (default): a neutral glass pill in `fg-primary` with a coloured dot; use it for statuses in tables ("Active", "Overdue"). The word carries the meaning, the dot only reinforces it.
- **outline**: transparent with a tone-coloured border ("Trial term", "XP + 10").
- **solid**: an uppercase `caption` fill for one category label on a card ("UI/UX DESIGN"). White on `button` is 4.0:1 at 11px, below AA (source kept), so use it only with `neutral` or `accent` tones and never for status.
- Don't make tags clickable; use a SegmentedControl or a filter button instead.
