# DataTable

A glass table for records with sortable headers, row checkboxes with bulk actions, tag columns and per-row action buttons.

- **Provide** `columns` and `rows`. Each column takes `key`, `header`, and optionally `sortable`, `sortValue`, `render`, `tags`, `align` (`right` for numbers and money), `primary` (the name column) and `width`.
- **Sorting:** click a sortable header to cycle ascending → descending → off; the active header shows a `button`-coloured arrow and sets `aria-sort`. Pass `defaultSort`, or control it with `sort` + `onSortChange`; add `manualSort` for server-side sorting.
- **Selection:** `selectable` adds a checkbox column with select-all (indeterminate when partly selected). While rows are selected, the header bar swaps the title for "N selected", your `bulkActions(keys, clear)` buttons and Clear. Selected rows get a glass fill and a 2px `button` edge.
- **Tags:** `tags: true` renders `row[key]` (a string or array) as Tags; a function `(value, row) => ({ tone })` picks each tag's tone.
- **Row actions:** `rowActions(row)` renders the last column; use two or three `sm` tinted IconButtons (edit, delete), or one "more" IconButton that opens a menu.
- `title` and `toolbar` fill the header bar; `footer` holds a Pagination; `empty` replaces the rows when there are none; `density="compact"` tightens rows.
- Give `rowKey` (default `id`) and `rowLabel` so checkboxes have names like "Select row Hola Spine".
- Keep it to about seven columns; on narrow screens the table scrolls sideways instead of wrapping cells.
