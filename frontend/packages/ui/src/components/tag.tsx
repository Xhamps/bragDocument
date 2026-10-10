import type * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "#lib/utils";

// Tones set --tag-c (the DS .bd-tone-* rules); variants decide where it lands.
const tagVariants = cva(
  "box-border inline-flex h-6 items-center gap-1.5 rounded-pill border border-container-border bg-container px-2 type-footnote font-medium whitespace-nowrap text-fg-primary shadow-sm",
  {
    variants: {
      variant: {
        dot: "",
        outline: "border-(--tag-c) bg-transparent shadow-none",
        solid:
          "h-[22px] rounded-sm border-0 bg-(--tag-c) type-caption text-button-fg",
      },
      tone: {
        neutral: "[--tag-c:var(--fg-tertiary)]",
        accent: "[--tag-c:var(--button-text)]",
        success: "[--tag-c:var(--success)]",
        danger: "[--tag-c:var(--danger)]",
        purple: "[--tag-c:var(--chart-2)]",
        teal: "[--tag-c:var(--chart-4)]",
        violet: "[--tag-c:var(--chart-3)]",
      },
    },
    compoundVariants: [
      {
        variant: "solid",
        tone: "neutral",
        className: "[--tag-c:var(--fg-secondary)]",
      },
      {
        variant: "solid",
        tone: "accent",
        className: "[--tag-c:var(--button)]",
      },
    ],
    defaultVariants: { variant: "dot", tone: "neutral" },
  },
);

type TagProps = React.ComponentProps<"span"> & VariantProps<typeof tagVariants>;

function Tag({
  variant = "dot",
  tone = "neutral",
  className,
  children,
  ...props
}: TagProps) {
  return (
    <span
      data-slot="tag"
      data-variant={variant}
      {...props}
      className={cn(tagVariants({ variant, tone }), className)}
    >
      {variant === "dot" && (
        <span
          data-slot="tag-dot"
          aria-hidden="true"
          className="size-2 shrink-0 rounded-pill bg-(--tag-c)"
        />
      )}
      {children}
    </span>
  );
}

export { Tag, tagVariants, type TagProps };
