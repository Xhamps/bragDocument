# MediaCell

An image beside a title and subtitle (and an optional third meta line), for table rows and lists that describe a person, course or file, like "Riley Anderson / UI/UX Designer" or a course thumbnail with its title and channel.

- **Provide** `title`, `subtitle`, and an `image`: a photo URL, or a `gradient-*` token name as an artwork placeholder; with no image it draws initials. `meta` adds a small `fg-tertiary` line; `badge` overlays the image corner (a duration "4:30").
- `shape`: `circle` for people, `rounded` for app or brand icons, `thumb` for 16:9 video and course thumbnails; `size="lg"` for spacious tables.
- The title clamps at two lines and the subtitle at one, so rows keep an even height.
- In a DataTable, set a column's `media: { image, subtitle, meta, badge, shape }` (row keys or functions) instead of writing a `render`; for text-only two-line cells ("$120" over "per year") use the column's `subtitle`.
- Always give a real `imageAlt` when the image carries information the title doesn't.
