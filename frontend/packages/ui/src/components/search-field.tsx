import type * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { Icon } from "#components/icon";

type SearchFieldProps = React.ComponentProps<"input"> & {
  /** Accessible name when the placeholder is not "Search". */
  label?: string;
};

function SearchField({ label, className, ...props }: SearchFieldProps) {
  return (
    <label
      data-slot="search-field"
      className={cn(
        "box-border flex h-11 min-w-60 items-center gap-2 rounded-pill glass pr-4 pl-5 shadow-glass",
        "transition-[border-color,box-shadow] duration-base ease-standard focus-within:shadow-md focus-within:inset-shadow-ds",
        className,
      )}
    >
      <input
        type="search"
        placeholder="Search"
        aria-label={label || props.placeholder || "Search"}
        {...props}
        className={cn(
          "min-w-0 flex-1 rounded-sm border-0 bg-transparent text-[15px] text-fg-primary placeholder:text-fg-secondary",
          focusRing,
        )}
      />
      <Icon name="search" className="size-5 shrink-0 text-fg-primary" />
    </label>
  );
}

export { SearchField, type SearchFieldProps };
