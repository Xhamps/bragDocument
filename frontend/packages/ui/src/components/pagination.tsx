import * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { pageList } from "#lib/page-list";
import { Icon } from "#components/icon";
import { IconButton } from "#components/icon-button";
import { Select } from "#components/select";

const pageBtn = cn(
  "box-border inline-flex h-8 min-w-8 cursor-pointer items-center justify-center rounded-pill border border-transparent bg-transparent px-2 type-callout text-fg-secondary tabular-nums",
  "transition-[background-color,color,scale,box-shadow] duration-base ease-spring hover:text-fg-primary active:not-disabled:scale-90",
  "disabled:cursor-default disabled:text-button-inactive",
  "aria-[current=page]:bg-button aria-[current=page]:font-semibold aria-[current=page]:text-button-fg aria-[current=page]:shadow-md",
  focusRing,
);

type PaginationProps = Omit<React.ComponentProps<"nav">, "onChange"> & {
  pageCount: number;
  page?: number;
  defaultPage?: number;
  onChange?: (page: number) => void;
  /** Pages shown either side of the current one (default 1). */
  siblingCount?: number;
  /** numbered (default) or simple ("‹ 3 of 12 ›"). */
  variant?: "numbered" | "simple";
  /** With pageSize, shows "11–20 of 248". */
  total?: number;
  pageSize?: number;
  /** Adds a "Rows per page" Select. */
  pageSizeOptions?: number[];
  onPageSizeChange?: (size: number) => void;
  label?: string;
};

function Pagination({
  pageCount,
  page: pageProp,
  defaultPage = 1,
  onChange,
  siblingCount = 1,
  variant = "numbered",
  total,
  pageSize,
  pageSizeOptions,
  onPageSizeChange,
  label = "Pagination",
  className,
  ...props
}: PaginationProps) {
  const count = Math.max(1, pageCount || 1);
  const [own, setOwn] = React.useState(defaultPage);
  const page = pageProp || own;
  const go = (n: number) => {
    n = Math.max(1, Math.min(count, n));
    if (!pageProp) setOwn(n);
    onChange?.(n);
  };

  const nav =
    variant === "simple" ? (
      <div className="ml-auto inline-flex items-center gap-3">
        <IconButton
          icon="chevron"
          label="Previous page"
          size="sm"
          className="[&>svg]:rotate-180"
          disabled={page <= 1}
          onClick={() => go(page - 1)}
        />
        <span
          aria-live="polite"
          className="type-callout font-normal text-fg-secondary tabular-nums"
        >
          {`${page} of ${count}`}
        </span>
        <IconButton
          icon="chevron"
          label="Next page"
          size="sm"
          disabled={page >= count}
          onClick={() => go(page + 1)}
        />
      </div>
    ) : (
      <div className="ml-auto inline-flex items-center gap-0.5 rounded-pill glass p-[3px] shadow-glass">
        <button
          type="button"
          className={pageBtn}
          aria-label="Previous page"
          disabled={page <= 1}
          onClick={() => go(page - 1)}
        >
          <Icon name="arrowLeft" size={16} />
        </button>
        {pageList(page, count, siblingCount).map((n) =>
          typeof n === "string" ? (
            <span
              key={n}
              aria-hidden="true"
              className="min-w-6 text-center text-fg-tertiary"
            >
              …
            </span>
          ) : (
            <button
              key={n}
              type="button"
              className={pageBtn}
              aria-current={n === page ? "page" : undefined}
              aria-label={`Page ${n}`}
              onClick={() => go(n)}
            >
              {n}
            </button>
          ),
        )}
        <button
          type="button"
          className={pageBtn}
          aria-label="Next page"
          disabled={page >= count}
          onClick={() => go(page + 1)}
        >
          <Icon name="arrowRight" size={16} />
        </button>
      </div>
    );

  const summary =
    total != null && pageSize ? (
      <span>{`${total ? (page - 1) * pageSize + 1 : 0}–${Math.min(total, page * pageSize)} of ${total}`}</span>
    ) : null;
  const size = pageSizeOptions ? (
    // Compact Select: .bd-pg-size .bd-select overrides.
    <label className="inline-flex items-center gap-2 [&_[data-slot=select]]:h-8 [&_[data-slot=select]]:pl-3 [&_[data-slot=select]>svg]:right-2 [&_select]:pr-6 [&_select]:text-[13px]">
      Rows per page
      <Select
        aria-label="Rows per page"
        options={pageSizeOptions.map(String)}
        value={String(pageSize)}
        onChange={(e) => onPageSizeChange?.(Number(e.target.value))}
      />
    </label>
  ) : null;

  return (
    <nav
      data-slot="pagination"
      aria-label={label}
      {...props}
      className={cn(
        "flex w-full flex-wrap items-center justify-between gap-4",
        className,
      )}
    >
      {(summary || size) && (
        <div className="inline-flex items-center gap-4 type-footnote text-fg-secondary">
          {size}
          {summary}
        </div>
      )}
      {nav}
    </nav>
  );
}

export { Pagination, type PaginationProps };
