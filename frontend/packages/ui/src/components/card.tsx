import type * as React from "react";
import { cva } from "class-variance-authority";
import { cn, focusRing } from "#lib/utils";

// Gradients from .bd-card-tint-* layered over the glass fill.
const cardTintVariants = cva("", {
  variants: {
    tint: {
      sheen:
        "bg-[image:linear-gradient(135deg,var(--glass-sheen),transparent_55%)]",
      accent:
        "bg-[image:linear-gradient(135deg,var(--glass-sheen),transparent_45%),linear-gradient(135deg,var(--glass-tint-blue),transparent_55%,var(--glass-tint-violet))]",
      violet:
        "bg-[image:linear-gradient(135deg,var(--glass-sheen),transparent_45%),radial-gradient(120%_90%_at_100%_0%,var(--glass-tint-violet),transparent_65%)]",
      rose: "bg-[image:linear-gradient(135deg,var(--glass-sheen),transparent_45%),linear-gradient(160deg,transparent_30%,var(--glass-tint-rose))]",
      cool: "bg-[image:linear-gradient(135deg,var(--glass-sheen),transparent_45%),linear-gradient(200deg,var(--glass-tint-teal),transparent_50%,var(--glass-tint-blue))]",
      none: "",
    },
  },
  defaultVariants: { tint: "sheen" },
});

const elevationClass = {
  sm: "shadow-sm",
  md: "shadow-md",
  lg: "shadow-lg",
  xl: "shadow-xl",
} as const;

// .bd-card-interactive: lift, press, and a sheen sweeping across on hover.
const interactiveClass = [
  "relative cursor-pointer overflow-hidden hover:-translate-y-1 hover:border-fg-tertiary hover:shadow-lg active:-translate-y-px active:shadow-md active:duration-fast motion-reduce:hover:translate-y-0",
  "after:pointer-events-none after:absolute after:inset-0 after:-translate-x-full after:bg-[image:linear-gradient(115deg,transparent_35%,var(--glass-sheen)_50%,transparent_65%)] after:opacity-60 after:transition-none hover:after:translate-x-full hover:after:transition-transform hover:after:duration-(--duration-enter) hover:after:ease-standard",
  focusRing,
];

type CardProps = Omit<React.ComponentProps<"div">, "title" | "onClick"> & {
  tint?: "sheen" | "accent" | "violet" | "rose" | "cool" | "none";
  /** Hover lift and sheen sweep; implied by onClick. */
  interactive?: boolean;
  /** Also fires on Enter/Space; makes the card focusable. */
  onClick?: () => void;
  elevation?: keyof typeof elevationClass;
  /** Fade-and-rise entrance; stagger with delay in ms. */
  animate?: boolean;
  delay?: number;
  title?: React.ReactNode;
  description?: React.ReactNode;
  /** Header slot to the right of the title (e.g. an IconButton). */
  action?: React.ReactNode;
  compact?: boolean;
  as?: React.ElementType;
};

function Card({
  tint = "sheen",
  interactive,
  onClick,
  elevation,
  animate,
  delay,
  title,
  description,
  action,
  compact,
  as: Comp = "div",
  className,
  style,
  onKeyDown,
  children,
  ...props
}: CardProps) {
  const isInteractive = interactive || !!onClick;
  return (
    <Comp
      data-slot="card"
      data-interactive={isInteractive || undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick}
      onKeyDown={(e: React.KeyboardEvent<HTMLDivElement>) => {
        onKeyDown?.(e);
        if (
          onClick &&
          e.target === e.currentTarget &&
          (e.key === "Enter" || e.key === " ")
        ) {
          e.preventDefault();
          onClick();
        }
      }}
      style={
        animate && delay
          ? ({ "--delay": `${delay}ms`, ...style } as React.CSSProperties)
          : style
      }
      {...props}
      className={cn(
        "box-border rounded-lg glass p-6 text-fg-primary shadow-glass transition-[translate,box-shadow,border-color] duration-slow ease-standard",
        compact && "p-4",
        cardTintVariants({ tint }),
        elevation && elevationClass[elevation],
        isInteractive && interactiveClass,
        animate && "animate-enter",
        className,
      )}
    >
      {(title || action) && (
        <div data-slot="card-header" className="flex items-start gap-4">
          {title && <h3 className="mb-2 flex-1 type-title-3">{title}</h3>}
          {action && <div className="ml-auto shrink-0">{action}</div>}
        </div>
      )}
      {description && (
        <p className="type-body text-fg-secondary">{description}</p>
      )}
      {children}
    </Comp>
  );
}

export { Card, type CardProps };
