# MenuList

A vertical list of navigation or settings rows with an optional icon, used for sidebars (All Components, Buttons, Menus…) and account menus (Account, Pricing, Billing…).

- **Provide** `items` (`{label, value?, icon?, trailing?}` or `"separator"`), and `value` + `onSelect` or `defaultValue`.
- The current row is marked by one glass highlight with a 2px `button-text` bar on its leading edge. When the selection changes, the highlight glides to the new row (`duration-slow`, `ease-standard`) and the bar stretches as it travels, then settles.
- In a sidebar that is already glass, pass `glass={false}`. Icons are 20px outline in `currentColor`.
- **Links**: give an item `href` and it renders as `<a>` (`aria-current="page"` when current); `external` opens a new tab and shows an external-link icon.
- **Sub menus**: give an item `children` and it becomes a group with a side chevron. By default (`submenu="flyout"`) the children open in a tooltip-style glass panel beside the row, with a pointer arrow toward it, `shadow-xl`, heavy blur and a spring scale-in; it opens on hover (closing 160ms after the pointer leaves, with a bridge so moving to it doesn't close it), click, focus plus ArrowRight, and closes on Escape or ArrowLeft (focus returns to the row), leaving or choosing an item. The sidebar highlight stays on the group row while its child is current; the current child shows in `button-text` inside the panel. `submenu="inline"` keeps the accordion: rows expand in place, indented along a hairline.
- A plain string in `items` is an uppercase section heading ("Workspace"); `badge` adds a count pill.
- Keep nesting to one level; deeper trees belong in a page, not the sidebar.
