import * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { Icon } from "#components/icon";
import { Button } from "#components/button";
import { Tag, type TagProps } from "#components/tag";
import { MediaCell, type MediaCellProps } from "#components/media-cell";

type Key = string | number;
type Field<Row> = string | ((row: Row) => React.ReactNode);

type DataTableColumn<Row = Record<string, unknown>> = {
  key: string;
  header: React.ReactNode;
  sortable?: boolean;
  /** Value used for sorting when it differs from row[key]. */
  sortValue?: (row: Row) => string | number;
  render?: (row: Row) => React.ReactNode;
  /** Render row[key] (a string or string[]) as Tags; a function maps each value to Tag props. */
  tags?: boolean | ((value: string, row: Row) => Partial<TagProps>);
  align?: "left" | "right" | "center";
  /** The row's name column: fg-primary, medium weight. */
  primary?: boolean;
  /** Render row[key] as a MediaCell title with an image and subtitle; each field is a row key or a function. */
  media?: {
    image?: string | ((row: Row) => string);
    subtitle?: Field<Row>;
    meta?: Field<Row>;
    badge?: Field<Row>;
    shape?: MediaCellProps["shape"];
    size?: MediaCellProps["size"];
  };
  /** Two-line cell: row[key] above a fg-secondary subtitle (a row key or a function). */
  subtitle?: Field<Row>;
  /** Let this column's text wrap (descriptions). */
  wrap?: boolean;
  maxWidth?: number | string;
  width?: number | string;
};

type DataTableSort = { key: string; dir: "asc" | "desc" };

type DataTableProps<Row = Record<string, unknown>> = Omit<
  React.ComponentProps<"div">,
  "title" | "children"
> & {
  columns: DataTableColumn<Row>[];
  rows: Row[];
  /** Field name (default "id") or function giving each row a unique key. */
  rowKey?: string | ((row: Row) => Key);
  /** Short row name for checkbox labels ("Select row Hola Spine"). */
  rowLabel?: (row: Row) => string;
  title?: React.ReactNode;
  /** Right side of the header bar when nothing is selected (search, filters, Add). */
  toolbar?: React.ReactNode;
  caption?: string;
  selectable?: boolean;
  selected?: Key[];
  defaultSelected?: Key[];
  onSelectionChange?: (keys: Key[]) => void;
  /** Buttons shown in the bar while rows are selected; receives the keys and a clear() callback. */
  bulkActions?: (keys: Key[], clear: () => void) => React.ReactNode;
  /** Per-row buttons in the last column (IconButtons, sm). */
  rowActions?: (row: Row) => React.ReactNode;
  sort?: DataTableSort | null;
  defaultSort?: DataTableSort;
  onSortChange?: (sort: DataTableSort | null) => void;
  /** Leave sorting to the caller (server-side); the table only reports onSortChange. */
  manualSort?: boolean;
  /** Under the rows, usually a Pagination. */
  footer?: React.ReactNode;
  empty?: React.ReactNode;
  density?: "comfortable" | "compact" | "spacious";
};

// Nulls last; numbers numerically; strings case-insensitive with numeric runs.
function cmp(a: unknown, b: unknown) {
  if (a == null) return 1;
  if (b == null) return -1;
  if (typeof a === "number" && typeof b === "number") return a - b;
  return String(a).localeCompare(String(b), undefined, {
    numeric: true,
    sensitivity: "base",
  });
}

const get = (row: unknown, key: string) =>
  (row as Record<string, unknown>)[key];

function pick<Row>(spec: Field<Row> | undefined, row: Row) {
  if (spec == null) return undefined;
  return typeof spec === "function"
    ? spec(row)
    : (get(row, spec) as React.ReactNode);
}

const ALIGN = {
  left: "",
  right: "text-right tabular-nums",
  center: "text-center",
} as const;

// Native checkbox (for `indeterminate`) drawn as the DS square box.
function Check({
  indeterminate,
  ...props
}: React.ComponentProps<"input"> & { indeterminate?: boolean }) {
  const ref = React.useRef<HTMLInputElement>(null);
  React.useEffect(() => {
    if (ref.current) ref.current.indeterminate = !!indeterminate;
  });
  return (
    <label className="group/check relative inline-flex cursor-pointer align-middle">
      <input
        ref={ref}
        type="checkbox"
        {...props}
        className="peer absolute m-0 size-5 cursor-pointer opacity-0"
      />
      <span
        aria-hidden="true"
        className={cn(
          "box-border inline-flex size-5 flex-none items-center justify-center rounded-sm border-[1.5px] border-fg-secondary text-transparent",
          "transition-[background-color,border-color,scale,box-shadow] duration-base ease-standard group-active/check:scale-[0.88]",
          "peer-checked:border-button peer-checked:bg-button peer-checked:text-button-fg peer-checked:shadow-sm",
          "peer-indeterminate:border-button peer-indeterminate:bg-button peer-indeterminate:text-button-fg",
          "[&>svg]:scale-[0.4] [&>svg]:opacity-0 [&>svg]:transition-[scale,opacity] [&>svg]:duration-base [&>svg]:ease-spring peer-checked:[&>svg]:scale-100 peer-checked:[&>svg]:opacity-100 peer-indeterminate:[&>svg]:scale-100 peer-indeterminate:[&>svg]:opacity-100",
          "peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-focus-ring peer-focus-visible:outline-solid",
        )}
      >
        <Icon name={indeterminate ? "minus" : "check"} size={14} />
      </span>
    </label>
  );
}

function DataTable<Row>({
  columns,
  rows,
  rowKey = "id",
  rowLabel,
  title,
  toolbar,
  caption,
  selectable,
  selected: selectedProp,
  defaultSelected = [],
  onSelectionChange,
  bulkActions,
  rowActions,
  sort: sortProp,
  defaultSort,
  onSortChange,
  manualSort,
  footer,
  empty,
  density = "comfortable",
  className,
  ...props
}: DataTableProps<Row>) {
  const keyOf = (r: Row): Key =>
    typeof rowKey === "function" ? rowKey(r) : (get(r, rowKey) as Key);
  const [ownSort, setOwnSort] = React.useState<DataTableSort | null>(
    defaultSort ?? null,
  );
  const sort = sortProp !== undefined ? sortProp : ownSort;
  const [ownSel, setOwnSel] = React.useState<Key[]>(defaultSelected);
  const selected = selectedProp ?? ownSel;

  const setSort = (key: string) => {
    const next: DataTableSort | null =
      !sort || sort.key !== key
        ? { key, dir: "asc" }
        : sort.dir === "asc"
          ? { key, dir: "desc" }
          : null;
    if (sortProp === undefined) setOwnSort(next);
    onSortChange?.(next);
  };
  const setSel = (next: Key[]) => {
    if (!selectedProp) setOwnSel(next);
    onSelectionChange?.(next);
  };

  let view = rows;
  if (sort && !manualSort) {
    const col = columns.find((c) => c.key === sort.key);
    const value = col?.sortValue ?? ((r: Row) => get(r, sort.key));
    view = rows.slice().sort((a, b) => {
      const d = cmp(value(a), value(b));
      return sort.dir === "desc" ? -d : d;
    });
  }
  const keys = view.map(keyOf);
  const selKeys = keys.filter((k) => selected.includes(k));
  const nSel = selKeys.length;
  const all = keys.length > 0 && nSel === keys.length;
  const some = nSel > 0 && !all;
  const toggle = (k: Key) =>
    setSel(
      selected.includes(k) ? selected.filter((x) => x !== k) : [...selected, k],
    );
  const clear = () => setSel([]);

  const cellY =
    density === "compact" ? "py-2" : density === "spacious" ? "py-4" : "py-3";
  const th = cn(
    "border-b border-divider px-4 text-left align-middle type-footnote font-medium whitespace-nowrap text-fg-secondary",
    density === "compact" ? "py-2" : "py-3",
  );
  const td = cn(
    "border-b border-divider px-4 align-middle whitespace-nowrap text-fg-secondary transition-[background-color,box-shadow] duration-fast ease-standard",
    cellY,
  );
  const colCount = columns.length + (selectable ? 1 : 0) + (rowActions ? 1 : 0);
  const showBar = title || toolbar || (selectable && nSel > 0);

  return (
    <div
      data-slot="data-table"
      {...props}
      className={cn(
        "overflow-hidden rounded-lg glass text-fg-primary shadow-glass",
        className,
      )}
    >
      {showBar && (
        <div
          className={cn(
            "box-border flex min-h-[60px] items-center justify-between gap-3 border-b border-divider px-5 py-3 transition-colors duration-base ease-standard",
            nSel > 0 && "bg-container",
          )}
        >
          {selectable && nSel > 0 ? (
            <>
              <span
                aria-live="polite"
                className="animate-enter text-[14px] font-semibold text-button-text [animation-duration:var(--duration-base)]"
              >
                {nSel} selected
              </span>
              <span className="inline-flex items-center gap-2">
                {bulkActions?.(selKeys, clear)}
                <Button variant="tinted" size="sm" onClick={clear}>
                  Clear
                </Button>
              </span>
            </>
          ) : (
            <>
              <span className="type-title-3">{title}</span>
              <span className="inline-flex items-center gap-2">{toolbar}</span>
            </>
          )}
        </div>
      )}
      <div className="overflow-x-auto">
        <table className="w-full border-collapse type-callout font-normal">
          {caption && <caption className="sr-only">{caption}</caption>}
          <thead>
            <tr>
              {selectable && (
                <th scope="col" className={cn(th, "w-11 pr-0")}>
                  <Check
                    checked={all}
                    indeterminate={some}
                    aria-label="Select all rows"
                    onChange={() => setSel(all ? [] : keys)}
                  />
                </th>
              )}
              {columns.map((c) => {
                const active = sort?.key === c.key;
                const ariaSort = active
                  ? sort.dir === "asc"
                    ? "ascending"
                    : "descending"
                  : c.sortable
                    ? "none"
                    : undefined;
                return (
                  <th
                    key={c.key}
                    scope="col"
                    aria-sort={ariaSort}
                    className={cn(th, c.align && ALIGN[c.align])}
                    style={c.width ? { width: c.width } : undefined}
                  >
                    {c.sortable ? (
                      <button
                        type="button"
                        onClick={() => setSort(c.key)}
                        className={cn(
                          "-mx-1.5 -my-1 inline-flex cursor-pointer items-center gap-1 rounded-sm border-0 bg-transparent px-1.5 py-1 align-middle text-inherit [font:inherit] hover:text-fg-primary [&>svg]:text-fg-tertiary",
                          c.align === "right" && "flex-row-reverse",
                          active && "text-fg-primary [&>svg]:text-button-text",
                          focusRing,
                        )}
                      >
                        {c.header}
                        <Icon
                          name={
                            active
                              ? sort.dir === "asc"
                                ? "sortUp"
                                : "sortDown"
                              : "sort"
                          }
                          size={16}
                        />
                      </button>
                    ) : (
                      c.header
                    )}
                  </th>
                );
              })}
              {rowActions && (
                <th
                  scope="col"
                  className={cn(th, "w-[1%] text-right whitespace-nowrap")}
                >
                  <span className="sr-only">Actions</span>
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {view.length ? (
              view.map((r) => {
                const k = keyOf(r);
                const on = selected.includes(k);
                return (
                  <tr
                    key={k}
                    aria-selected={selectable ? on : undefined}
                    className={cn(
                      "group/row last:[&>td]:border-b-0 hover:[&>td]:bg-container",
                      on &&
                        "[&>td]:bg-container [&>td:first-child]:shadow-[inset_2px_0_0_var(--button-text)]",
                    )}
                  >
                    {selectable && (
                      <td className={cn(td, "w-11 pr-0")}>
                        <Check
                          checked={on}
                          aria-label={`Select row ${rowLabel ? rowLabel(r) : k}`}
                          onChange={() => toggle(k)}
                        />
                      </td>
                    )}
                    {columns.map((c) => {
                      let v = c.render
                        ? c.render(r)
                        : (get(r, c.key) as React.ReactNode);
                      if (c.media && !c.render) {
                        const m = c.media;
                        v = (
                          <MediaCell
                            title={v}
                            image={
                              typeof m.image === "function"
                                ? m.image(r)
                                : m.image != null
                                  ? (get(r, m.image) as string)
                                  : undefined
                            }
                            subtitle={pick(m.subtitle, r)}
                            meta={pick(m.meta, r)}
                            badge={pick(m.badge, r)}
                            shape={m.shape}
                            size={m.size}
                          />
                        );
                      } else if (c.subtitle && !c.render) {
                        v = (
                          <span
                            className={cn(
                              "inline-flex flex-col",
                              c.align === "right" && "items-end",
                            )}
                          >
                            <span className="font-medium text-fg-primary">
                              {v}
                            </span>
                            <span className="type-footnote text-fg-secondary">
                              {pick(c.subtitle, r)}
                            </span>
                          </span>
                        );
                      }
                      if (c.tags && !c.render) {
                        const tags = c.tags;
                        const tv = (Array.isArray(v) ? v : [v]) as string[];
                        v = (
                          <span className="inline-flex gap-1">
                            {tv.map((t, i) => (
                              <Tag
                                key={i}
                                {...(typeof tags === "function"
                                  ? tags(t, r)
                                  : {})}
                              >
                                {t}
                              </Tag>
                            ))}
                          </span>
                        );
                      }
                      return (
                        <td
                          key={c.key}
                          className={cn(
                            td,
                            c.align && ALIGN[c.align],
                            c.primary && "font-medium text-fg-primary",
                            c.wrap && "min-w-[200px] whitespace-normal",
                          )}
                          style={
                            c.maxWidth ? { maxWidth: c.maxWidth } : undefined
                          }
                        >
                          {v}
                        </td>
                      );
                    })}
                    {rowActions && (
                      <td
                        className={cn(
                          td,
                          "w-[1%] text-right whitespace-nowrap",
                        )}
                      >
                        <span className="inline-flex items-center gap-1">
                          {rowActions(r)}
                        </span>
                      </td>
                    )}
                  </tr>
                );
              })
            ) : (
              <tr>
                <td
                  colSpan={colCount}
                  className="px-4 py-12 text-center text-fg-secondary"
                >
                  {empty || "No results"}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {footer && (
        <div className="flex justify-end border-t border-divider px-5 py-3">
          {footer}
        </div>
      )}
    </div>
  );
}

export {
  DataTable,
  type DataTableColumn,
  type DataTableProps,
  type DataTableSort,
};
