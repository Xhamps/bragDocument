# Pagination

Page navigation for lists and tables: previous/next arrows and page numbers in a glass pill, the current page filled with `button`.

- **Provide** `pageCount` and `page` + `onChange` (or `defaultPage`). `siblingCount` (default 1) sets how many neighbours show around the current page; distant pages collapse into "…".
- With `total` and `pageSize` it shows "11–20 of 248" on the left; `pageSizeOptions` + `onPageSizeChange` add a "Rows per page" Select.
- `variant="simple"` shows only "‹ 3 of 12 ›" for cards and narrow spaces.
- Put it in a DataTable's `footer`, or under a list, right-aligned. Hide it when there is a single page.
