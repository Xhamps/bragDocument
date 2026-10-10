import type * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { renderIcon, type IconProp } from "#components/icon";

type FieldTileProps = Omit<React.ComponentProps<"button">, "value"> & {
  label: React.ReactNode;
  value?: React.ReactNode;
  placeholder?: React.ReactNode;
  /** 24px icon: a node or a built-in name. */
  icon?: IconProp;
};

function FieldTile({
  label,
  value,
  placeholder,
  icon,
  className,
  ...props
}: FieldTileProps) {
  return (
    <button
      type="button"
      data-slot="field-tile"
      {...props}
      className={cn(
        "group/tile box-border flex min-h-[72px] w-full cursor-pointer items-center gap-4 rounded-md glass px-5 py-3 text-left text-fg-primary",
        "transition-[border-color,box-shadow,translate,scale] duration-base ease-standard hover:-translate-y-0.5 hover:border-fg-tertiary hover:shadow-md active:scale-[0.99] motion-reduce:hover:translate-y-0",
        focusRing,
        className,
      )}
    >
      {icon && (
        <span
          aria-hidden="true"
          className="inline-flex shrink-0 text-fg-primary transition-transform duration-base ease-spring group-hover/tile:scale-110"
        >
          {renderIcon(icon, 24)}
        </span>
      )}
      <span className="flex min-w-0 flex-col">
        <span className="type-footnote text-fg-secondary">{label}</span>
        <span
          className={cn(
            "truncate text-[17px] leading-6",
            value ? "text-fg-primary" : "text-fg-secondary",
          )}
        >
          {value || placeholder}
        </span>
      </span>
    </button>
  );
}

export { FieldTile, type FieldTileProps };
