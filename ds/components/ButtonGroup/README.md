# ButtonGroup

Joins related Buttons or IconButtons into one glass pill split by `container-divider` hairlines, like the ← → pager and "Join | Subscribed" with its bell and menu chevron.

- **Provide** two to four Buttons or IconButtons as children, and a `label` for the group; `size="sm"` inside cards.
- Children lose their own borders and glass; a `primary` child keeps its fill (the active half).
- Use it for actions that belong together (previous/next, an action plus its options menu). For choosing one of several states, use SegmentedControl.
