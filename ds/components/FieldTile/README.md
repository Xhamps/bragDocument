# FieldTile

A large glass tile showing a labelled value that opens a picker on press, as in the flight search form (From / Montreal, Canada · Depart / Dec 15, 2023 · Class / Economy).

- **Provide** `label`, `value` (or `placeholder` while empty), an `icon` (a node or a built-in name) and `onClick` to open the picker (a calendar, a list, a Stepper popover).
- Lay tiles in a two-column grid with `space-4` gaps inside a Card, followed by one primary action ("Search flights").
- It is a button, not a text input: never type into it.
