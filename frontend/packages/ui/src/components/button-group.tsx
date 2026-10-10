import type * as React from "react";
import { cn } from "#lib/utils";

type ButtonGroupProps = React.ComponentProps<"div"> & {
  /** Buttons or IconButtons joined into one pill. */
  children: React.ReactNode;
  label?: string;
  size?: "sm" | "md";
};

// Children lose their own border, glass, radius and motion; a primary child
// keeps its fill. `!` beats the children's own variant classes.
function ButtonGroup({
  label,
  size = "md",
  className,
  children,
  ...props
}: ButtonGroupProps) {
  return (
    <div
      role="group"
      aria-label={label}
      data-slot="button-group"
      className={cn(
        "inline-flex items-stretch overflow-hidden rounded-pill glass shadow-button",
        "*:translate-none! *:scale-none! *:rounded-none! *:border-0! *:shadow-none! *:[backdrop-filter:none]!",
        "[&>*+*]:border-l! [&>*+*]:border-divider!",
        "[&>:not([data-variant=primary])]:bg-transparent [&>:not([data-variant=primary])]:hover:bg-container",
        "[&>[data-slot=icon-button]]:w-11",
        size === "sm" &&
          "*:h-7! [&>[data-slot=button]]:type-footnote [&>[data-slot=icon-button]]:w-9",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export { ButtonGroup, type ButtonGroupProps };
